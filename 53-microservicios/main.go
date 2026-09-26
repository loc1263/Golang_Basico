// Tema 53: Microservicios: comunicación y patrones
// Un sistema de microservicios divide la aplicación en servicios pequeños,
// desplegables por separado, que se comunican por red. Eso trae problemas
// que un monolito no tiene: la red falla, las instancias cambian, un servicio
// lento arrastra a los demás. Este tema implementa (solo con la librería
// estándar, para ver cómo funcionan por dentro) los patrones básicos:
//
//	descubrimiento.go  registro de servicios con heartbeats + balanceo round-robin
//	resiliencia.go     circuit breaker + reintentos con backoff exponencial y jitter
//	mensajeria.go      broker pub/sub: at-least-once, reentregas, DLQ, idempotencia
//	main.go            servicios de ejemplo + correlation ID + la simulación
//
// En producción se usan piezas probadas (Kubernetes, Consul, un service mesh
// como Istio/Linkerd, RabbitMQ/Kafka/NATS, gobreaker...), pero los conceptos
// son exactamente estos.
//
// Ejecutar con: go run ./53-microservicios
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"
)

// ==================== CORRELATION ID ====================
// Un request del usuario atraviesa varios servicios. Un ID común, propagado
// en un header, permite encontrar todos sus logs (tema 49 lo generaliza con
// trazas distribuidas).

const headerRequestID = "X-Request-ID"

type claveCtx struct{}

func conRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, claveCtx{}, id)
}

func requestID(ctx context.Context) string {
	id, _ := ctx.Value(claveCtx{}).(string)
	return id
}

// middlewareRequestID toma el ID entrante o genera uno nuevo
func middlewareRequestID(siguiente http.Handler) http.Handler {
	var n atomic.Int64
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(headerRequestID)
		if id == "" {
			id = fmt.Sprintf("req-%03d", n.Add(1))
		}
		w.Header().Set(headerRequestID, id)
		siguiente.ServeHTTP(w, r.WithContext(conRequestID(r.Context(), id)))
	})
}

// ==================== SERVICIO CATÁLOGO (2 instancias) ====================

type Producto struct {
	ID     string  `json:"id"`
	Nombre string  `json:"nombre"`
	Precio float64 `json:"precio"`
}

// instanciaCatalogo puede ponerse "enferma" para simular una caída
type instanciaCatalogo struct {
	id        string
	enferma   atomic.Bool
	atendidos atomic.Int64
}

func (c *instanciaCatalogo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.atendidos.Add(1)
	if c.enferma.Load() {
		http.Error(w, "servicio no disponible", http.StatusServiceUnavailable)
		return
	}
	id := r.PathValue("id")
	if id != "libro-1" {
		http.Error(w, "no existe", http.StatusNotFound)
		return
	}
	w.Header().Set("X-Instancia", c.id)
	json.NewEncoder(w).Encode(Producto{ID: id, Nombre: "El Aleph", Precio: 11})
}

// ==================== CLIENTE RESILIENTE ====================

// ClienteCatalogo combina: descubrimiento + balanceo + circuit breaker por
// instancia + reintentos + timeout + propagación del request ID
type ClienteCatalogo struct {
	balanceador *RoundRobin
	breakers    map[string]*CircuitBreaker
	http        *http.Client
	log         func(string, ...any)
}

func (c *ClienteCatalogo) breaker(id string) *CircuitBreaker {
	if cb, ok := c.breakers[id]; ok {
		return cb
	}
	cb := NuevoCircuitBreaker(id, 2, 300*time.Millisecond, func(nombre string, de, a EstadoCircuito) {
		c.log("   [breaker %s] %s -> %s", nombre, de, a)
	})
	c.breakers[id] = cb
	return cb
}

func (c *ClienteCatalogo) ObtenerProducto(ctx context.Context, id string) (Producto, string, error) {
	// Timeout TOTAL para la operación, incluidos los reintentos
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	instancias, err := c.balanceador.Siguientes()
	if err != nil {
		return Producto{}, "", err
	}

	var prod Producto
	var atendio string
	err = Reintentar(ctx, len(instancias), 20*time.Millisecond, func(intento int) error {
		inst := instancias[(intento-1)%len(instancias)] // cada reintento, otra instancia
		err := c.breaker(inst.ID).Ejecutar(func() error {
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, inst.URL+"/productos/"+id, nil)
			req.Header.Set(headerRequestID, requestID(ctx)) // propagar el correlation ID
			resp, err := c.http.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			switch {
			case resp.StatusCode == http.StatusNotFound:
				// Un 404 no mejora reintentando, y tampoco indica que la
				// instancia esté mal: no debe abrir el circuito
				return nil
			case resp.StatusCode >= 500:
				io.Copy(io.Discard, resp.Body)
				c.log("   [%s] intento %d en %s: status %d", requestID(ctx), intento, inst.ID, resp.StatusCode)
				return fmt.Errorf("%s respondió %d", inst.ID, resp.StatusCode)
			}
			atendio = inst.ID
			return json.NewDecoder(resp.Body).Decode(&prod)
		})
		if errors.Is(err, ErrCircuitoAbierto) {
			c.log("   [%s] intento %d: %s omitida (circuito abierto, no se la llama)", requestID(ctx), intento, inst.ID)
		}
		return err
	})
	if err == nil && atendio == "" {
		return Producto{}, "", ErrNoReintentable{errors.New("producto no encontrado")}
	}
	return prod, atendio, err
}

// ==================== SIMULACIÓN ====================

func main() {
	logf := func(formato string, args ...any) { fmt.Printf(formato+"\n", args...) }
	ctx := context.Background()

	// Levantar dos instancias del servicio catálogo y registrarlas
	registro := NuevoRegistro(time.Second)
	instA := &instanciaCatalogo{id: "catalogo-a"}
	instB := &instanciaCatalogo{id: "catalogo-b"}
	for _, inst := range []*instanciaCatalogo{instA, instB} {
		mux := http.NewServeMux()
		mux.Handle("GET /productos/{id}", inst)
		srv := httptest.NewServer(middlewareRequestID(mux))
		defer srv.Close()
		registro.Registrar("catalogo", inst.id, srv.URL)
	}

	cliente := &ClienteCatalogo{
		balanceador: &RoundRobin{registro: registro, servicio: "catalogo"},
		breakers:    map[string]*CircuitBreaker{},
		http:        &http.Client{Timeout: 500 * time.Millisecond}, // timeout por intento
		log:         logf,
	}
	pedir := func(n int) {
		rctx := conRequestID(ctx, fmt.Sprintf("req-%03d", n))
		p, inst, err := cliente.ObtenerProducto(rctx, "libro-1")
		if err != nil {
			logf("   req-%03d -> ERROR: %v", n, err)
			return
		}
		logf("   req-%03d -> %s ($%.0f) atendido por %s", n, p.Nombre, p.Precio, inst)
	}

	fmt.Println("1) Descubrimiento + balanceo round-robin (2 instancias sanas)")
	for i := 1; i <= 4; i++ {
		pedir(i)
	}

	fmt.Println("\n2) catalogo-b se cae: reintento en otra instancia y el breaker se abre")
	instB.enferma.Store(true)
	for i := 5; i <= 11; i++ {
		pedir(i)
	}
	fmt.Printf("   catalogo-b recibió %d requests en total: con el circuito ABIERTO ya no se lo llama\n", instB.atendidos.Load())

	fmt.Println("\n3) catalogo-b se recupera: tras la espera, una llamada de prueba (SEMI-ABIERTO) cierra el circuito")
	instB.enferma.Store(false)
	time.Sleep(350 * time.Millisecond)
	for _, id := range []string{"catalogo-a", "catalogo-b"} {
		registro.Latido(id) // las instancias siguen enviando latidos
	}
	for i := 12; i <= 15; i++ {
		pedir(i)
	}

	fmt.Println("\n4) Una instancia deja de enviar latidos: el registro la descarta")
	time.Sleep(600 * time.Millisecond)
	registro.Latido("catalogo-a") // solo A sigue viva
	time.Sleep(500 * time.Millisecond)
	fmt.Printf("   instancias sanas: %v\n", idsDe(registro.Sanas("catalogo")))
	pedir(16)
	pedir(17)

	fmt.Println("\n5) Mensajería asíncrona: pedidos publica eventos, otros servicios consumen")
	ctxBroker, cancelar := context.WithCancel(ctx)
	defer cancelar()
	broker := NuevoBroker(3, logf)

	// Consumidor "notificaciones": idempotente, falla la primera vez con p-1
	dedup := NuevoDeduplicador()
	var fallosSimulados atomic.Int32
	broker.Suscribir(ctxBroker, "pedido.creado", "notificaciones", dedup.Envolver("notificaciones", logf,
		func(ctx context.Context, m Mensaje) error {
			if m.Datos["cliente"] == "" {
				logf("   [notificaciones] %s sin cliente: nada que notificar", m.ID)
				return nil
			}
			if m.Datos["pedido"] == "p-1" && fallosSimulados.Add(1) == 1 {
				return errors.New("SMTP temporalmente caído")
			}
			logf("   [notificaciones] email enviado a %s por el pedido %s", m.Datos["cliente"], m.Datos["pedido"])
			return nil
		}))

	// Consumidor "inventario": rechaza siempre un mensaje mal formado
	var stockReservado atomic.Int32
	broker.Suscribir(ctxBroker, "pedido.creado", "inventario", func(ctx context.Context, m Mensaje) error {
		if m.Datos["cliente"] == "" {
			return errors.New("mensaje inválido: falta el cliente")
		}
		stockReservado.Add(1)
		logf("   [inventario] stock reservado para %s", m.Datos["pedido"])
		return nil
	})

	eventos := []Mensaje{
		{ID: "evt-1", Tema: "pedido.creado", Datos: map[string]string{"pedido": "p-1", "cliente": "ana"}},
		{ID: "evt-2", Tema: "pedido.creado", Datos: map[string]string{"pedido": "p-2", "cliente": "bruno"}},
		{ID: "evt-3", Tema: "pedido.creado", Datos: map[string]string{"pedido": "p-3"}}, // "venenoso"
	}
	for _, e := range eventos {
		logf("   [pedidos] publica %s (%s)", e.ID, e.Datos["pedido"])
		broker.Publicar(e)
	}
	broker.Esperar()

	fmt.Println("\n   El broker reentrega evt-2 (p. ej. se perdió el ack):")
	broker.Reentregar(eventos[1])
	broker.Esperar()

	fmt.Printf("\n   Resumen: stock reservado %d veces", stockReservado.Load())
	fmt.Println(" (inventario NO es idempotente: ¡reservó p-2 dos veces!)")
	for _, m := range broker.DLQ() {
		fmt.Printf("   En la DLQ para revisión manual: %s %v (intentos: %d)\n", m.ID, m.Datos, m.Intentos)
	}
}

func idsDe(is []Instancia) []string {
	var ids []string
	for _, i := range is {
		ids = append(ids, i.ID)
	}
	return ids
}
