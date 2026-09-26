package main

import (
	"context"
	"sync"
	"time"
)

// ==================== MENSAJERÍA ASÍNCRONA (pub/sub) ====================
// No todo debe ser una llamada HTTP síncrona. Si el servicio de pedidos
// llamara directamente a notificaciones, inventario, facturación... cada
// caída de uno de ellos rompería la creación de pedidos. En cambio, PUBLICA
// un evento ("pedido.creado") en un broker y cada interesado lo consume a su
// ritmo: desacoplamiento temporal y de disponibilidad.
// Brokers reales: RabbitMQ, Kafka, NATS, Google Pub/Sub, AWS SQS/SNS.
//
// Este broker en memoria imita garantías típicas:
//   - "al menos una vez" (at-least-once): si el consumidor falla, se reentrega
//     => el consumidor debe ser IDEMPOTENTE (procesar dos veces = una vez)
//   - dead letter queue (DLQ): tras N fallos el mensaje se aparta para
//     revisarlo, en vez de bloquear la cola para siempre

type Mensaje struct {
	ID       string
	Tema     string
	Datos    map[string]string
	Intentos int
}

type Consumidor func(ctx context.Context, m Mensaje) error

type suscripcion struct {
	nombre string
	cola   chan Mensaje
}

type Broker struct {
	mu            sync.Mutex
	suscripciones map[string][]*suscripcion
	maxIntentos   int
	dlq           []Mensaje
	wg            sync.WaitGroup
	log           func(formato string, args ...any)
}

func NuevoBroker(maxIntentos int, log func(string, ...any)) *Broker {
	return &Broker{suscripciones: map[string][]*suscripcion{}, maxIntentos: maxIntentos, log: log}
}

// Suscribir crea una cola propia para el consumidor: cada suscriptor recibe
// su copia del mensaje (fan-out), y procesa los suyos en orden.
func (b *Broker) Suscribir(ctx context.Context, tema, nombre string, f Consumidor) {
	s := &suscripcion{nombre: nombre, cola: make(chan Mensaje, 100)}
	b.mu.Lock()
	b.suscripciones[tema] = append(b.suscripciones[tema], s)
	b.mu.Unlock()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case m := <-s.cola:
				m.Intentos++
				if err := f(ctx, m); err != nil {
					if m.Intentos >= b.maxIntentos {
						b.log("   [broker] %s agotó %d intentos con %s -> DLQ (%v)", nombre, m.Intentos, m.ID, err)
						b.mu.Lock()
						b.dlq = append(b.dlq, m)
						b.mu.Unlock()
						b.wg.Done()
						continue
					}
					b.log("   [broker] %s falló con %s (intento %d): %v -> se reentrega", nombre, m.ID, m.Intentos, err)
					go func() { // reentregar tras una pausa, sin bloquear la cola
						time.Sleep(20 * time.Millisecond)
						s.cola <- m
					}()
					continue
				}
				b.wg.Done() // "ack": mensaje procesado
			}
		}
	}()
}

// Publicar entrega el mensaje a todas las suscripciones del tema
func (b *Broker) Publicar(m Mensaje) {
	b.mu.Lock()
	subs := b.suscripciones[m.Tema]
	b.mu.Unlock()
	for _, s := range subs {
		b.wg.Add(1)
		s.cola <- m
	}
}

// Reentregar simula que el broker envía otra vez un mensaje ya procesado
// (ocurre en la vida real: timeouts de ack, reinicios, rebalanceos de Kafka)
func (b *Broker) Reentregar(m Mensaje) { b.Publicar(m) }

// Esperar bloquea hasta que todo lo publicado fue procesado o fue a la DLQ
func (b *Broker) Esperar() { b.wg.Wait() }

func (b *Broker) DLQ() []Mensaje {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]Mensaje(nil), b.dlq...)
}

// Deduplicador hace idempotente a un consumidor: recuerda los IDs ya
// procesados. En producción se guarda en la base de datos, idealmente en la
// misma transacción que el efecto del mensaje (patrón "inbox").
type Deduplicador struct {
	mu     sync.Mutex
	vistos map[string]bool
}

func (d *Deduplicador) Envolver(nombre string, log func(string, ...any), f Consumidor) Consumidor {
	return func(ctx context.Context, m Mensaje) error {
		d.mu.Lock()
		if d.vistos[m.ID] {
			d.mu.Unlock()
			log("   [%s] %s ya procesado: se ignora el duplicado", nombre, m.ID)
			return nil
		}
		d.mu.Unlock()
		if err := f(ctx, m); err != nil {
			return err
		}
		d.mu.Lock()
		d.vistos[m.ID] = true
		d.mu.Unlock()
		return nil
	}
}

func NuevoDeduplicador() *Deduplicador { return &Deduplicador{vistos: map[string]bool{}} }
