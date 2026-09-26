// Tema 20: select y sync.Mutex
//
// Qué aprenderás:
//   - select: esperar sobre varios channels a la vez.
//   - Timeouts con time.After y operaciones no bloqueantes con default.
//   - sync.Mutex: proteger una variable compartida entre goroutines.
//
// Ejecutar con: go run ./20-select-mutex
package main

import (
	"fmt"
	"sync"
	"time"
)

// ContadorSeguro agrupa el dato y el mutex que lo protege: es la forma
// habitual de usar un Mutex. El valor cero de sync.Mutex ya está listo para
// usarse (no necesita inicialización).
type ContadorSeguro struct {
	mu    sync.Mutex
	valor int
}

func (c *ContadorSeguro) Incrementar() {
	c.mu.Lock()         // solo una goroutine a la vez pasa de aquí
	defer c.mu.Unlock() // defer garantiza que se libere incluso ante un panic
	c.valor++           // sección crítica
}

func (c *ContadorSeguro) Valor() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.valor
}

func main() {
	// select espera hasta que ALGUNO de sus casos pueda ejecutarse.
	// Si varios están listos a la vez, elige uno al azar.
	canal1 := make(chan string)
	canal2 := make(chan string)

	go func() {
		time.Sleep(50 * time.Millisecond)
		canal1 <- "resultado de canal1"
	}()
	go func() {
		time.Sleep(20 * time.Millisecond)
		canal2 <- "resultado de canal2"
	}()

	fmt.Println("Esperando ambos canales (llega primero el más rápido):")
	for range 2 {
		select {
		case msg1 := <-canal1:
			fmt.Println("Recibido:", msg1)
		case msg2 := <-canal2:
			fmt.Println("Recibido:", msg2)
		}
	}

	// Timeout: time.After devuelve un channel que recibe un valor pasado el
	// tiempo indicado. Evita esperar para siempre a una operación lenta.
	canalLento := make(chan string, 1) // con buffer: la goroutine no queda bloqueada si nadie recibe
	go func() {
		time.Sleep(200 * time.Millisecond)
		canalLento <- "esto llega tarde"
	}()

	fmt.Println("\nEsperando con timeout:")
	select {
	case msg := <-canalLento:
		fmt.Println("Recibido:", msg)
	case <-time.After(50 * time.Millisecond):
		fmt.Println("Timeout: el canal tardó demasiado")
	}

	// default: si ningún caso está listo, se ejecuta sin esperar
	canalVacio := make(chan int)
	select {
	case v := <-canalVacio:
		fmt.Println("Valor recibido:", v)
	default:
		fmt.Println("\nNingún canal tenía datos disponibles (default)")
	}

	// sync.Mutex: cuando varias goroutines modifican la MISMA variable, hay
	// que protegerla. Sin el mutex habría una "condición de carrera" y el
	// resultado podría ser menor que 100 (el tema 47 lo demuestra).
	fmt.Println("\nContador protegido con Mutex:")
	var contador ContadorSeguro
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(contador.Incrementar) // wg.Go (Go 1.25+) lanza la goroutine y la espera
	}
	wg.Wait()
	fmt.Println("Valor final del contador (esperado 100):", contador.Valor())

	// ¿Channel o Mutex? Regla práctica:
	//   - Channels para PASAR datos o coordinar etapas entre goroutines.
	//   - Mutex para PROTEGER un estado compartido (un contador, un cache).
}
