// Package aplicacion contiene los CASOS DE USO (lo que la aplicación hace:
// crear pedido, pagarlo...) y define los PUERTOS: interfaces que describen
// lo que los casos de uso necesitan del mundo exterior.
//
// Depende solo de dominio. Los adaptadores (base de datos, email, HTTP)
// dependen de ESTE paquete, nunca al revés: esa es la "regla de dependencia"
// (las flechas apuntan hacia el centro).
package aplicacion

import (
	"context"
	"fmt"
	"time"

	"golang-basico/50-arquitectura-limpia/internal/dominio"
)

// ---------- Puertos de salida (driven ports) ----------

// RepositorioPedidos abstrae la persistencia. Implementaciones: memoria y sqlite.
type RepositorioPedidos interface {
	Guardar(ctx context.Context, p *dominio.Pedido) error
	Buscar(ctx context.Context, id string) (*dominio.Pedido, error)
	Listar(ctx context.Context) ([]*dominio.Pedido, error)
}

// Notificador abstrae el envío de avisos (email, SMS, cola de mensajes...)
type Notificador interface {
	Notificar(ctx context.Context, cliente, mensaje string) error
}

// Dependencias técnicas pequeñas también se abstraen para poder testear
type Reloj func() time.Time
type GeneradorID func() string

// ---------- Casos de uso (puerto de entrada / driving port) ----------

// ServicioPedidos es lo que usan los adaptadores de entrada (HTTP, CLI,
// gRPC, un consumidor de colas...). Ninguno de ellos contiene lógica de negocio.
type ServicioPedidos struct {
	repo    RepositorioPedidos
	notif   Notificador
	ahora   Reloj
	nuevoID GeneradorID
}

func NuevoServicioPedidos(repo RepositorioPedidos, notif Notificador, ahora Reloj, nuevoID GeneradorID) *ServicioPedidos {
	return &ServicioPedidos{repo: repo, notif: notif, ahora: ahora, nuevoID: nuevoID}
}

// DTO de entrada: desacopla el caso de uso del formato de transporte (JSON, proto...)
type CrearPedidoInput struct {
	Cliente string
	Items   []dominio.Item
}

func (s *ServicioPedidos) CrearPedido(ctx context.Context, in CrearPedidoInput) (*dominio.Pedido, error) {
	p, err := dominio.NuevoPedido(s.nuevoID(), in.Cliente, in.Items, s.ahora())
	if err != nil {
		return nil, err
	}
	if err := s.repo.Guardar(ctx, p); err != nil {
		return nil, fmt.Errorf("guardar pedido: %w", err)
	}
	// Decisión de negocio: el aviso es "best effort". Si falla, el pedido
	// igual queda creado, así que el error se descarta explícitamente.
	_ = s.notif.Notificar(ctx, p.Cliente, fmt.Sprintf("Recibimos tu pedido %s por $%.2f", p.ID, p.Total()))
	return p, nil
}

func (s *ServicioPedidos) ObtenerPedido(ctx context.Context, id string) (*dominio.Pedido, error) {
	return s.repo.Buscar(ctx, id)
}

func (s *ServicioPedidos) ListarPedidos(ctx context.Context) ([]*dominio.Pedido, error) {
	return s.repo.Listar(ctx)
}

// PagarPedido orquesta: cargar -> aplicar regla de dominio -> guardar -> notificar
func (s *ServicioPedidos) PagarPedido(ctx context.Context, id string) (*dominio.Pedido, error) {
	p, err := s.repo.Buscar(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := p.Pagar(); err != nil { // la regla vive en el dominio
		return nil, err
	}
	if err := s.repo.Guardar(ctx, p); err != nil {
		return nil, fmt.Errorf("guardar pedido: %w", err)
	}
	_ = s.notif.Notificar(ctx, p.Cliente, "Pago confirmado para el pedido "+p.ID) // "best effort"
	return p, nil
}

func (s *ServicioPedidos) CancelarPedido(ctx context.Context, id string) (*dominio.Pedido, error) {
	p, err := s.repo.Buscar(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := p.Cancelar(); err != nil {
		return nil, err
	}
	return p, s.repo.Guardar(ctx, p)
}
