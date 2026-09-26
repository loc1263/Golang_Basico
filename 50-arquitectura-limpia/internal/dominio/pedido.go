// Package dominio contiene las entidades y reglas de negocio puras.
// Es el CENTRO de la arquitectura: no importa ningún otro paquete del
// proyecto, ni base de datos, ni HTTP, ni JSON. Solo Go estándar.
// Si mañana cambiamos SQLite por Postgres o HTTP por gRPC, este código no cambia.
package dominio

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// Errores de dominio: los adaptadores los traducen a su propio lenguaje
// (el adaptador HTTP a status 404/409/422, uno gRPC a codes.NotFound, etc.)
var (
	ErrPedidoNoEncontrado = errors.New("pedido no encontrado")
	ErrTransicionInvalida = errors.New("transición de estado inválida")
	ErrDatosInvalidos     = errors.New("datos inválidos")
)

type EstadoPedido string

const (
	EstadoPendiente EstadoPedido = "pendiente"
	EstadoPagado    EstadoPedido = "pagado"
	EstadoEnviado   EstadoPedido = "enviado"
	EstadoCancelado EstadoPedido = "cancelado"
)

type Item struct {
	Producto       string
	Cantidad       int
	PrecioUnitario float64
}

// Pedido es una entidad: tiene identidad (ID) y reglas que protegen su
// consistencia. Los campos se modifican solo a través de métodos.
type Pedido struct {
	ID       string
	Cliente  string
	Items    []Item
	Estado   EstadoPedido
	CreadoEn time.Time
}

// NuevoPedido es el único modo de crear un pedido válido
func NuevoPedido(id, cliente string, items []Item, ahora time.Time) (*Pedido, error) {
	if strings.TrimSpace(cliente) == "" {
		return nil, fmt.Errorf("%w: el cliente es obligatorio", ErrDatosInvalidos)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("%w: el pedido necesita al menos un ítem", ErrDatosInvalidos)
	}
	for _, it := range items {
		if it.Cantidad <= 0 || it.PrecioUnitario < 0 {
			return nil, fmt.Errorf("%w: ítem %q con cantidad o precio inválido", ErrDatosInvalidos, it.Producto)
		}
	}
	return &Pedido{ID: id, Cliente: cliente, Items: items, Estado: EstadoPendiente, CreadoEn: ahora}, nil
}

// Total es lógica de negocio: vive en la entidad, no en el handler HTTP
func (p *Pedido) Total() float64 {
	var t float64
	for _, it := range p.Items {
		t += float64(it.Cantidad) * it.PrecioUnitario
	}
	// Regla de negocio: 10% de descuento en compras mayores a 100
	if t > 100 {
		t *= 0.9
	}
	return t
}

// Máquina de estados: qué transiciones están permitidas
var transiciones = map[EstadoPedido][]EstadoPedido{
	EstadoPendiente: {EstadoPagado, EstadoCancelado},
	EstadoPagado:    {EstadoEnviado, EstadoCancelado},
}

func (p *Pedido) cambiarA(nuevo EstadoPedido) error {
	if slices.Contains(transiciones[p.Estado], nuevo) {
		p.Estado = nuevo
		return nil
	}
	return fmt.Errorf("%w: de %s a %s", ErrTransicionInvalida, p.Estado, nuevo)
}

func (p *Pedido) Pagar() error    { return p.cambiarA(EstadoPagado) }
func (p *Pedido) Enviar() error   { return p.cambiarA(EstadoEnviado) }
func (p *Pedido) Cancelar() error { return p.cambiarA(EstadoCancelado) }
