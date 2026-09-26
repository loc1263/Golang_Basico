// Tema 14: Goroutines y Channels (concurrencia básica)
//
// Una goroutine es una función que se ejecuta de forma concurrente con el
// resto del programa. Son muy livianas: un programa puede tener miles.
// Un channel es un "tubo" tipado para enviar datos entre goroutines de
// forma segura, sin compartir variables.
//
//	"No te comuniques compartiendo memoria; comparte memoria comunicándote."
//
// Qué aprenderás:
//   - Lanzar goroutines con "go" y esperarlas con sync.WaitGroup.
//   - Channels sin buffer y con buffer; cerrar y recorrer un channel.
//   - El patrón worker pool.
//
// Ejecutar con: go run ./14-goroutines-channels
package main

import (
	"fmt"
	"sync"
)

func saludar(nombre string, wg *sync.WaitGroup) {
	defer wg.Done() // avisa al WaitGroup que esta goroutine terminó
	fmt.Println("Hola desde", nombre)
}

// worker recibe trabajos de un channel y envía resultados a otro.
// Las flechas en los tipos restringen el uso:
//
//	<-chan int  solo se puede RECIBIR de él
//	chan<- int  solo se puede ENVIAR a él
func worker(id int, trabajos <-chan int, resultados chan<- string) {
	for trabajo := range trabajos { // termina cuando el channel se cierra
		resultados <- fmt.Sprintf("worker %d procesó %d -> %d", id, trabajo, trabajo*2)
	}
}

func main() {
	// sync.WaitGroup espera a que un grupo de goroutines termine
	var wg sync.WaitGroup
	nombres := []string{"Ana", "Luis", "María"}

	fmt.Println("Lanzando goroutines (el orden de salida puede variar):")
	for _, nombre := range nombres {
		wg.Add(1)               // una goroutine más por esperar
		go saludar(nombre, &wg) // "go" ejecuta la función en una nueva goroutine
	}
	wg.Wait() // bloquea hasta que todas llamen a wg.Done()

	// Go 1.25+: wg.Go(f) hace Add(1), lanza f en una goroutine y llama a
	// Done al terminar. Es la forma recomendada hoy.
	fmt.Println("\nLo mismo con wg.Go:")
	for _, nombre := range nombres {
		wg.Go(func() {
			fmt.Println("Hola de nuevo,", nombre)
		})
	}
	wg.Wait()

	// Channel sin buffer: el envío espera a que alguien reciba, y la
	// recepción espera a que alguien envíe. Sirve para sincronizar.
	fmt.Println("\nUsando un channel simple:")
	mensajes := make(chan string)
	go func() {
		mensajes <- "mensaje enviado desde una goroutine" // enviar
	}()
	recibido := <-mensajes // recibir (bloquea hasta que llegue un valor)
	fmt.Println(recibido)

	// Channel con buffer: admite hasta N valores sin que nadie los reciba
	fmt.Println("\nUsando un channel con buffer:")
	buffer := make(chan int, 3)
	buffer <- 1
	buffer <- 2
	buffer <- 3
	// close indica que no se enviarán más valores. Solo debe cerrarlo quien
	// envía. Enviar a un channel cerrado provoca un panic.
	close(buffer)
	for valor := range buffer { // range recibe hasta vaciarlo y detecta el cierre
		fmt.Println("Valor recibido:", valor)
	}

	// Recibir de un channel cerrado y vacío devuelve el valor cero;
	// el segundo valor (ok) permite saber si el channel sigue abierto.
	v, ok := <-buffer
	fmt.Println("Recibir de un channel cerrado:", v, "ok =", ok)

	// Patrón worker pool: varias goroutines toman trabajos de un mismo channel
	fmt.Println("\nPatrón worker pool (3 workers, 5 trabajos):")
	trabajos := make(chan int, 5)
	resultados := make(chan string, 5)

	for w := 1; w <= 3; w++ {
		go worker(w, trabajos, resultados)
	}

	for i := 1; i <= 5; i++ {
		trabajos <- i
	}
	close(trabajos) // los workers saldrán de su for-range al vaciarse

	for range 5 { // recibir exactamente 5 resultados
		fmt.Println(<-resultados)
	}
	// El tema 38 muestra cómo cerrar el channel de resultados de forma
	// segura cuando no se sabe de antemano cuántos habrá.
}
