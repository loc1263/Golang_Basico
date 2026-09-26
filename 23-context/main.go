// Tema 23: context.Context
//
// Un context viaja a través de las llamadas a funciones y lleva tres cosas:
//   - una señal de CANCELACIÓN (por ejemplo, el usuario cerró la conexión),
//   - un plazo o TIMEOUT,
//   - valores asociados al request (usar con moderación).
//
// Es omnipresente en servidores, clientes HTTP y bases de datos.
//
// Convenciones:
//   - Es el primer parámetro de la función y se llama ctx: func f(ctx context.Context, ...)
//   - Nunca se guarda dentro de un struct; se pasa en cada llamada.
//   - Quien crea un context con cancelación debe llamar a cancel().
//
// Ejecutar con: go run ./23-context
package main

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// tareaLarga simula un trabajo de 10 pasos que revisa en cada paso si debe
// detenerse
func tareaLarga(ctx context.Context, resultado chan<- string) {
	for i := 1; i <= 10; i++ {
		select {
		case <-ctx.Done():
			// Done() se cierra cuando el context se cancela o vence.
			// Err() explica el motivo: context.Canceled o context.DeadlineExceeded.
			resultado <- fmt.Sprintf("tarea detenida en el paso %d: %v", i, ctx.Err())
			return
		case <-time.After(50 * time.Millisecond):
			fmt.Println("procesando paso", i)
		}
	}
	resultado <- "tarea completada"
}

// Tipo propio para las claves de context.WithValue: evita choques con claves
// de otros paquetes. Nunca uses un string suelto como clave.
type claveCtx string

const claveUsuario claveCtx = "usuario"

func mostrarUsuario(ctx context.Context) {
	// ctx.Value devuelve "any": hay que convertirlo con una type assertion
	if usuario, ok := ctx.Value(claveUsuario).(string); ok {
		fmt.Println("Usuario obtenido del contexto:", usuario)
	} else {
		fmt.Println("El contexto no tiene usuario")
	}
}

func main() {
	// context.Background() es el context raíz, vacío, del que derivan los demás

	// WithTimeout: se cancela solo al pasar el tiempo indicado
	fmt.Println("Ejemplo con timeout (200ms); la tarea necesita 500ms:")
	ctx1, cancel1 := context.WithTimeout(context.Background(), 200*time.Millisecond)
	// Siempre llamar a cancel, aunque el timeout venza solo: libera recursos
	// internos antes de tiempo. "defer cancel()" es el patrón habitual.
	defer cancel1()

	resultado1 := make(chan string)
	go tareaLarga(ctx1, resultado1)
	r1 := <-resultado1
	fmt.Println("Resultado:", r1)
	fmt.Println("¿Venció el plazo?", errors.Is(ctx1.Err(), context.DeadlineExceeded))

	// WithCancel: cancelación manual controlada por el programa
	fmt.Println("\nEjemplo con cancelación manual:")
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	resultado2 := make(chan string)
	go tareaLarga(ctx2, resultado2)

	time.Sleep(120 * time.Millisecond)
	cancel2() // cancela ctx2 y todos los contexts derivados de él
	fmt.Println("Resultado:", <-resultado2)

	// La cancelación se propaga de padre a hijos, nunca al revés
	fmt.Println("\nPropagación de padre a hijo:")
	padre, cancelPadre := context.WithCancel(context.Background())
	hijo, cancelHijo := context.WithTimeout(padre, time.Hour)
	defer cancelHijo()
	cancelPadre()
	<-hijo.Done() // el hijo se cancela aunque su propio timeout sea de 1 hora
	fmt.Println("El hijo se canceló por su padre:", hijo.Err())

	// WithValue: adjuntar datos del request (id de usuario, id de traza...).
	// No lo uses para pasar parámetros normales de una función.
	fmt.Println("\nEjemplo con valores en el contexto:")
	ctx3 := context.WithValue(context.Background(), claveUsuario, "Ana")
	mostrarUsuario(ctx3)
	mostrarUsuario(context.Background())

	// Una tarea que termina antes de que venza el plazo
	fmt.Println("\nEjemplo donde la tarea SÍ alcanza a completarse:")
	ctx4, cancel4 := context.WithTimeout(context.Background(), time.Second)
	defer cancel4()
	resultado4 := make(chan string)
	go tareaCorta(ctx4, resultado4)
	fmt.Println("Resultado:", <-resultado4)
}

func tareaCorta(ctx context.Context, resultado chan<- string) {
	select {
	case <-ctx.Done():
		resultado <- fmt.Sprintf("cancelada: %v", ctx.Err())
	case <-time.After(100 * time.Millisecond):
		resultado <- "tarea corta completada a tiempo"
	}
}
