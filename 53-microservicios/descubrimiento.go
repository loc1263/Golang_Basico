package main

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// ==================== SERVICE DISCOVERY ====================
// En un sistema de microservicios las instancias aparecen y desaparecen
// (escalado, despliegues, caídas), así que las direcciones no se escriben a
// mano: cada instancia se REGISTRA y envía "latidos" (heartbeats); los
// clientes consultan el registro. Herramientas reales: Consul, etcd, el DNS
// interno de Kubernetes (Services).

type Instancia struct {
	ID           string
	Servicio     string
	URL          string
	ultimoLatido time.Time
}

type Registro struct {
	mu         sync.RWMutex
	instancias map[string]*Instancia // por ID
	ttl        time.Duration         // sin latido en este tiempo = instancia muerta
	ahora      func() time.Time
}

func NuevoRegistro(ttl time.Duration) *Registro {
	return &Registro{instancias: map[string]*Instancia{}, ttl: ttl, ahora: time.Now}
}

func (r *Registro) Registrar(servicio, id, url string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.instancias[id] = &Instancia{ID: id, Servicio: servicio, URL: url, ultimoLatido: r.ahora()}
}

func (r *Registro) Latido(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if i, ok := r.instancias[id]; ok {
		i.ultimoLatido = r.ahora()
	}
}

func (r *Registro) Baja(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.instancias, id)
}

// Sanas devuelve las instancias vivas de un servicio (orden estable por ID)
func (r *Registro) Sanas(servicio string) []Instancia {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Instancia
	for _, i := range r.instancias {
		if i.Servicio == servicio && r.ahora().Sub(i.ultimoLatido) < r.ttl {
			out = append(out, *i)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].ID < out[b].ID })
	return out
}

// ==================== BALANCEO DE CARGA (del lado del cliente) ====================

// RoundRobin reparte los requests entre las instancias por turnos
type RoundRobin struct {
	registro *Registro
	servicio string
	contador atomic.Uint64
}

// Siguientes devuelve TODAS las instancias sanas empezando por la que toca:
// si la primera falla, el cliente puede reintentar con la siguiente.
func (rr *RoundRobin) Siguientes() ([]Instancia, error) {
	sanas := rr.registro.Sanas(rr.servicio)
	if len(sanas) == 0 {
		return nil, fmt.Errorf("no hay instancias sanas de %q", rr.servicio)
	}
	// El módulo se calcula en uint64 ANTES de convertir a int: convertir un
	// uint64 muy grande a int podría desbordarse y dar un índice negativo.
	inicio := int((rr.contador.Add(1) - 1) % uint64(len(sanas))) //nolint:gosec // el módulo garantiza que cabe en int
	return append(sanas[inicio:], sanas[:inicio]...), nil
}
