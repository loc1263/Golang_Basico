package aplicacion_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang-basico/50-arquitectura-limpia/internal/adaptadores/memoria"
	"golang-basico/50-arquitectura-limpia/internal/aplicacion"
	"golang-basico/50-arquitectura-limpia/internal/dominio"
)

// Gracias a los puertos, los casos de uso se testean sin HTTP ni base de
// datos real: usamos el adaptador en memoria y un notificador espía (tema 41).

type notifEspia struct{ mensajes []string }

func (n *notifEspia) Notificar(_ context.Context, cliente, msg string) error {
	n.mensajes = append(n.mensajes, cliente+": "+msg)
	return nil
}

func nuevoServicio() (*aplicacion.ServicioPedidos, *notifEspia) {
	n := &notifEspia{}
	reloj := func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
	id := func() string { return "P-1" }
	return aplicacion.NuevoServicioPedidos(memoria.NuevoRepositorio(), n, reloj, id), n
}

func TestCrearYPagarPedido(t *testing.T) {
	svc, notif := nuevoServicio()
	ctx := context.Background()

	p, err := svc.CrearPedido(ctx, aplicacion.CrearPedidoInput{
		Cliente: "ana",
		Items:   []dominio.Item{{Producto: "libro", Cantidad: 2, PrecioUnitario: 60}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := p.Total(); got != 108 { // 120 con 10% de descuento
		t.Errorf("Total = %v; se esperaba 108", got)
	}

	p, err = svc.PagarPedido(ctx, "P-1")
	if err != nil {
		t.Fatal(err)
	}
	if p.Estado != dominio.EstadoPagado {
		t.Errorf("Estado = %s; se esperaba pagado", p.Estado)
	}
	if len(notif.mensajes) != 2 {
		t.Errorf("se esperaban 2 notificaciones, hubo %d: %v", len(notif.mensajes), notif.mensajes)
	}
}

func TestReglasDeNegocio(t *testing.T) {
	svc, _ := nuevoServicio()
	ctx := context.Background()

	_, err := svc.CrearPedido(ctx, aplicacion.CrearPedidoInput{Cliente: "ana"})
	if !errors.Is(err, dominio.ErrDatosInvalidos) {
		t.Errorf("pedido sin ítems: err = %v", err)
	}

	_, err = svc.PagarPedido(ctx, "no-existe")
	if !errors.Is(err, dominio.ErrPedidoNoEncontrado) {
		t.Errorf("pagar inexistente: err = %v", err)
	}

	svc.CrearPedido(ctx, aplicacion.CrearPedidoInput{Cliente: "ana", Items: []dominio.Item{{Producto: "x", Cantidad: 1, PrecioUnitario: 1}}})
	svc.CancelarPedido(ctx, "P-1")
	_, err = svc.PagarPedido(ctx, "P-1")
	if !errors.Is(err, dominio.ErrTransicionInvalida) {
		t.Errorf("pagar cancelado: err = %v", err)
	}
}
