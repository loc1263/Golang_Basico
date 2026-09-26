// Package memoria es un adaptador de salida: implementa el puerto
// aplicacion.RepositorioPedidos guardando los pedidos en un map.
// Útil para tests y prototipos.
package memoria

import (
	"context"
	"sort"
	"sync"

	"golang-basico/50-arquitectura-limpia/internal/dominio"
)

type Repositorio struct {
	mu      sync.RWMutex
	pedidos map[string]dominio.Pedido
}

func NuevoRepositorio() *Repositorio {
	return &Repositorio{pedidos: map[string]dominio.Pedido{}}
}

// Guardamos COPIAS (dominio.Pedido, no *dominio.Pedido) para que quien
// llama no pueda modificar el almacenamiento sin pasar por Guardar,
// igual que ocurriría con una base de datos real.
func (r *Repositorio) Guardar(_ context.Context, p *dominio.Pedido) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copia := *p
	copia.Items = append([]dominio.Item(nil), p.Items...)
	r.pedidos[p.ID] = copia
	return nil
}

func (r *Repositorio) Buscar(_ context.Context, id string) (*dominio.Pedido, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.pedidos[id]
	if !ok {
		return nil, dominio.ErrPedidoNoEncontrado
	}
	return &p, nil
}

func (r *Repositorio) Listar(_ context.Context) ([]*dominio.Pedido, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	lista := make([]*dominio.Pedido, 0, len(r.pedidos))
	for _, p := range r.pedidos {
		lista = append(lista, &p)
	}
	sort.Slice(lista, func(i, j int) bool { return lista[i].ID < lista[j].ID })
	return lista, nil
}
