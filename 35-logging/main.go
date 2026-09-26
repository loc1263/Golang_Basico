// Tema 35: Logging estructurado (log/slog)
// El paquete log clásico escribe texto libre. slog (Go 1.21+) escribe logs
// ESTRUCTURADOS: un mensaje más pares clave=valor, fáciles de filtrar y de
// procesar por herramientas (Loki, Elasticsearch, CloudWatch...).
//   - Niveles: Debug < Info < Warn < Error
//   - Handlers: TextHandler (clave=valor) y JSONHandler (una línea JSON)
//   - Logger.With agrega campos fijos a todos los logs de ese logger
//
// Ejecutar con: go run ./35-logging
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// Usuario implementa slog.LogValuer para controlar cómo se loguea:
// así evitamos filtrar datos sensibles como la contraseña.
type Usuario struct {
	ID       int
	Email    string
	Password string
}

func (u Usuario) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("id", u.ID),
		slog.String("email", u.Email),
		// Password se omite a propósito
	)
}

// quitarHora elimina el campo time de la salida para que el ejemplo sea
// reproducible (en un programa real normalmente se deja).
func quitarHora(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey && len(groups) == 0 {
		return slog.Attr{} // un Attr vacío se descarta
	}
	return a
}

func procesarPedido(logger *slog.Logger, pedidoID int) error {
	// Logger derivado: todos sus logs incluyen pedido_id automáticamente
	log := logger.With("pedido_id", pedidoID)
	log.Info("procesando pedido")
	if pedidoID%2 == 0 {
		err := errors.New("stock insuficiente")
		log.Error("no se pudo procesar", "err", err)
		return err
	}
	log.Info("pedido completado", "duracion", 120*time.Millisecond)
	return nil
}

func main() {
	// 1) Logger por defecto: slog.Info usa un logger global que escribe con
	// el formato del paquete log clásico
	fmt.Println("1) Logger por defecto")
	slog.Info("aplicación iniciada", "version", "1.0.0", "puerto", 8080)

	// 2) TextHandler: formato clave=valor, legible para humanos
	fmt.Println("\n2) TextHandler")
	texto := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: quitarHora,
	}))
	texto.Info("usuario conectado", "usuario", "ana", "intentos", 1)
	texto.Warn("disco casi lleno", "uso_pct", 91.5)
	texto.Debug("esto NO aparece") // el nivel mínimo por defecto es Info

	// 3) JSONHandler: una línea JSON por evento, ideal para producción
	fmt.Println("\n3) JSONHandler")
	jsonLog := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: quitarHora,
	}))
	jsonLog.Info("request atendido", "metodo", "GET", "ruta", "/tareas", "status", 200)

	// 4) Atributos tipados: slog.String, slog.Int, etc. evitan errores de
	// pares clave/valor desbalanceados y son más eficientes.
	fmt.Println("\n4) Atributos tipados y grupos")
	jsonLog.Info("consulta a la base",
		slog.String("tabla", "productos"),
		slog.Duration("tiempo", 35*time.Millisecond), // en JSON se escribe en nanosegundos
		slog.Group("paginacion", slog.Int("pagina", 2), slog.Int("tamano", 20)),
	)

	// 5) Loggers derivados con With y WithGroup
	fmt.Println("\n5) Logger.With (campos comunes)")
	appLog := texto.With("servicio", "pedidos", "entorno", "dev")
	// procesarPedido ya registra sus propios errores en el log, así que
	// aquí se descartan de forma explícita con "_ ="
	_ = procesarPedido(appLog, 7)
	_ = procesarPedido(appLog, 8)

	// 6) Nivel configurable en tiempo de ejecución con slog.LevelVar
	fmt.Println("\n6) Nivel dinámico")
	nivel := new(slog.LevelVar) // arranca en Info
	dinamico := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:       nivel,
		ReplaceAttr: quitarHora,
	}))
	dinamico.Debug("debug oculto")
	nivel.Set(slog.LevelDebug) // p. ej. activado por una variable de entorno
	dinamico.Debug("debug visible", "detalle", "cache miss")

	// 7) LogValuer: el tipo decide cómo se muestra (sin la contraseña)
	fmt.Println("\n7) LogValuer para ocultar datos sensibles")
	u := Usuario{ID: 1, Email: "ana@ejemplo.com", Password: "super-secreta"}
	jsonLog.Info("login exitoso", "usuario", u)

	// 8) Establecer un logger como global: slog.Info y log.Print lo usarán
	fmt.Println("\n8) slog.SetDefault")
	slog.SetDefault(jsonLog)
	slog.Info("ahora el logger global escribe JSON")

	// Las variantes *Context permiten a handlers personalizados extraer datos
	// del context (por ejemplo un trace_id)
	slog.InfoContext(context.Background(), "log con context")
}
