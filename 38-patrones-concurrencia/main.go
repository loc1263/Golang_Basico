// Tema 38: Patrones de concurrencia (pipelines, fan-out/fan-in, errgroup)
// Combinando goroutines, channels y context (temas 14, 20 y 23) aparecen
// patrones reutilizables:
//   - Pipeline: etapas conectadas por channels; cada etapa lee de un channel
//     de entrada, transforma y escribe en uno de salida.
//   - Fan-out: varias goroutines leen del MISMO channel para repartir trabajo.
//   - Fan-in: varios channels se combinan en uno solo.
//   - errgroup (golang.org/x/sync/errgroup): lanza goroutines, espera a todas
//     y devuelve el primer error, cancelando a las demás vía context.
//
// Ejecutar con: go run ./38-patrones-concurrencia
package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// ===================== 1) PIPELINE =====================

// generar es la primera etapa: produce números y cierra el channel al terminar.
// Devolver <-chan (solo lectura) deja claro quién es el dueño del channel.
func generar(ctx context.Context, nums ...int) <-chan int {
	salida := make(chan int)
	go func() {
		defer close(salida) // quien escribe es quien cierra
		for _, n := range nums {
			select {
			case salida <- n:
			case <-ctx.Done(): // si nadie lee más, no quedarse bloqueado
				return
			}
		}
	}()
	return salida
}

// cuadrado es una etapa intermedia: lee, transforma y reenvía
func cuadrado(ctx context.Context, entrada <-chan int) <-chan int {
	salida := make(chan int)
	go func() {
		defer close(salida)
		for n := range entrada {
			select {
			case salida <- n * n:
			case <-ctx.Done():
				return
			}
		}
	}()
	return salida
}

// ============== 2) FAN-OUT / FAN-IN ==============

// trabajoLento simula una operación costosa (p. ej. una llamada HTTP)
func trabajoLento(n int) string {
	time.Sleep(100 * time.Millisecond)
	return fmt.Sprintf("item-%d procesado", n)
}

// trabajador lee del channel compartido (fan-out) y escribe resultados
func trabajador(ctx context.Context, id int, entrada <-chan int) <-chan string {
	salida := make(chan string)
	go func() {
		defer close(salida)
		for n := range entrada {
			select {
			case salida <- fmt.Sprintf("[w%d] %s", id, trabajoLento(n)):
			case <-ctx.Done():
				return
			}
		}
	}()
	return salida
}

// unir combina varios channels en uno (fan-in). Un WaitGroup detecta cuándo
// terminaron todos para cerrar el channel de salida.
func unir(ctx context.Context, canales ...<-chan string) <-chan string {
	var wg sync.WaitGroup
	salida := make(chan string)

	// Una goroutine por channel de entrada, que reenvía todo a "salida"
	for _, c := range canales {
		wg.Go(func() {
			for v := range c {
				select {
				case salida <- v:
				case <-ctx.Done():
					return
				}
			}
		})
	}
	// Cuando terminan todas, se cierra "salida" para avisar al consumidor
	go func() {
		wg.Wait()
		close(salida)
	}()
	return salida
}

// ===================== 3) ERRGROUP =====================

// descargar simula traer una URL; "falla" si la URL contiene "roto"
func descargar(ctx context.Context, url string) (int, error) {
	demora := time.Duration(len(url)*10) * time.Millisecond
	select {
	case <-time.After(demora):
	case <-ctx.Done():
		return 0, ctx.Err() // cancelado porque otra goroutine falló
	}
	if url == "https://roto.example" {
		return 0, fmt.Errorf("descargar %s: 503 Service Unavailable", url)
	}
	return len(url) * 100, nil // "bytes descargados"
}

func descargarTodas(urls []string) (map[string]int, error) {
	// WithContext devuelve un ctx que se cancela cuando alguna goroutine
	// devuelve error (o cuando Wait termina)
	g, ctx := errgroup.WithContext(context.Background())

	var mu sync.Mutex
	resultados := make(map[string]int)

	for _, url := range urls {
		// Desde Go 1.22 cada iteración tiene su propia variable url,
		// así que es seguro capturarla en el closure.
		g.Go(func() error {
			n, err := descargar(ctx, url)
			if err != nil {
				fmt.Printf("   %-26s error: %v\n", url, err)
				return err
			}
			fmt.Printf("   %-26s ok (%d bytes)\n", url, n)
			mu.Lock()
			resultados[url] = n
			mu.Unlock()
			return nil
		})
	}
	// Wait bloquea hasta que TODAS terminan y devuelve el primer error
	return resultados, g.Wait()
}

func main() {
	ctx := context.Background()

	// 1) Pipeline: generar -> cuadrado -> cuadrado -> consumir
	fmt.Println("1) Pipeline generar -> cuadrado -> cuadrado:")
	for v := range cuadrado(ctx, cuadrado(ctx, generar(ctx, 1, 2, 3, 4))) {
		fmt.Print("   ", v)
	}
	fmt.Println()

	// Cancelación del pipeline: el consumidor corta antes de terminar y
	// cancel() libera las goroutines que quedaron esperando para enviar.
	fmt.Println("\n   Consumir solo los 2 primeros y cancelar el resto:")
	ctxPipe, cancelar := context.WithCancel(ctx)
	resultados := cuadrado(ctxPipe, generar(ctxPipe, 10, 20, 30, 40, 50))
	fmt.Println("  ", <-resultados, <-resultados)
	cancelar()

	// 2) Fan-out / fan-in: 9 tareas de 100ms repartidas en 3 trabajadores
	fmt.Println("\n2) Fan-out a 3 trabajadores y fan-in de resultados:")
	inicio := time.Now()
	tareas := generar(ctx, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	var canales []<-chan string
	for i := 1; i <= 3; i++ {
		canales = append(canales, trabajador(ctx, i, tareas))
	}
	var lineas []string
	for r := range unir(ctx, canales...) {
		lineas = append(lineas, r)
	}
	sort.Strings(lineas) // el orden de llegada no es determinista
	for _, l := range lineas {
		fmt.Println("  ", l)
	}
	fmt.Printf("   9 tareas de 100ms en ~%v (en serie serían ~900ms)\n",
		time.Since(inicio).Round(100*time.Millisecond))

	// 3) errgroup: todas bien
	fmt.Println("\n3a) errgroup, todas las descargas exitosas:")
	res, err := descargarTodas([]string{"https://a.example", "https://bb.example", "https://ccc.example"})
	fmt.Println("   resultados:", len(res), "error:", err)

	// errgroup con un fallo: las descargas más lentas se cancelan
	fmt.Println("\n3b) errgroup, una falla y cancela a las demás:")
	res, err = descargarTodas([]string{
		"https://roto.example",
		"https://a.example",
		"https://una-url-muy-larga-y-lenta.example",
	})
	fmt.Println("   resultados:", len(res), "primer error:", err)
	// Wait devuelve el PRIMER error real, no el context.Canceled de las demás
	fmt.Println("   ¿el error devuelto es la cancelación?", errors.Is(err, context.Canceled))

	// 4) errgroup.SetLimit: como un worker pool, máximo N goroutines a la vez
	fmt.Println("\n4) errgroup con SetLimit(2):")
	var g errgroup.Group
	g.SetLimit(2)
	var mu sync.Mutex
	activas, maxActivas := 0, 0
	for i := 1; i <= 6; i++ {
		g.Go(func() error { // Go bloquea si ya hay 2 corriendo
			mu.Lock()
			activas++
			maxActivas = max(maxActivas, activas)
			mu.Unlock()

			time.Sleep(50 * time.Millisecond)

			mu.Lock()
			activas--
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil { // ninguna función devuelve error aquí, pero se revisa igual
		fmt.Println("   error:", err)
	}
	fmt.Println("   máximo de goroutines simultáneas:", maxActivas)
}
