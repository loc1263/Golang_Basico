// Package consola implementa el puerto Notificador imprimiendo en pantalla.
// En producción habría otro adaptador (SMTP, SendGrid, una cola...) con la
// misma interfaz.
package consola

import (
	"context"
	"fmt"
	"io"
)

type Notificador struct{ Salida io.Writer }

func (n Notificador) Notificar(_ context.Context, cliente, mensaje string) error {
	_, err := fmt.Fprintf(n.Salida, "   [notificación a %s] %s\n", cliente, mensaje)
	return err
}
