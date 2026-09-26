// Tema 47: Concurrencia avanzada
//
// Complementa los temas 14, 20 y 38 con las primitivas de más bajo nivel de
// los paquetes sync y sync/atomic.
//
// Qué aprenderás:
//   - Qué es una condición de carrera y cómo detectarla con -race.
//   - atomic: operaciones indivisibles sin mutex.
//   - sync.OnceValue, sync.RWMutex, sync.Pool y sync.Cond.
//   - Worker pools y semáforos ponderados.
//
// Ejecutar con:         go run ./47-concurrencia-avanzada
// Detector de carreras: go run -race ./47-concurrencia-avanzada -carrera
//
// El flag -race necesita cgo: en Windows requiere un compilador de C (gcc,
// por ejemplo el de MSYS2). En Linux y macOS suele funcionar sin más.
package main

import (
	"cmp"
	"context"
	"flag"
	"fmt"
	"math/rand/v2"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/semaphore"
)

// ============ 1) Condición de carrera y sus soluciones ============

// carrera tiene un ERROR a propósito: muchas goroutines modifican
// "contador" sin sincronización. contador++ son tres pasos (leer, sumar,
// escribir); si dos goroutines los intercalan, se pierde un incremento.
func carrera(n int) int {
	contador := 0
	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			contador++ // condición de carrera: acceso concurrente sin protección
		})
	}
	wg.Wait()
	return contador
}

// conAtomic resuelve el problema con atomic.Int64: sus operaciones son
// indivisibles a nivel de CPU. Es más liviano que un mutex, pero solo sirve
// para operaciones simples sobre un único valor (contadores, flags).
func conAtomic(n int) int64 {
	var contador atomic.Int64
	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			contador.Add(1)
		})
	}
	wg.Wait()
	return contador.Load()
}

// ============ 2) sync.OnceValue: inicializar una sola vez ============

type Config struct{ URL string }

// sync.OnceValue envuelve una función para que se ejecute UNA sola vez,
// aunque muchas goroutines la llamen al mismo tiempo; las siguientes
// llamadas devuelven el resultado guardado. Ideal para inicialización
// perezosa (conexiones, configuraciones, cachés).
var cargarConfig = sync.OnceValue(func() *Config {
	fmt.Println("   (cargando configuración... solo una vez)")
	time.Sleep(50 * time.Millisecond)
	return &Config{URL: "postgres://localhost/db"}
})

// ============ 3) RWMutex: muchas lecturas, pocas escrituras ============

// Cache permite varias lecturas simultáneas (RLock) y solo bloquea a todos
// durante una escritura (Lock). Conviene cuando se lee mucho más de lo que
// se escribe; si no, un sync.Mutex común es más simple e igual de rápido.
type Cache struct {
	mu    sync.RWMutex
	datos map[string]string
}

func (c *Cache) Get(k string) (string, bool) {
	c.mu.RLock() // varios lectores a la vez
	defer c.mu.RUnlock()
	v, ok := c.datos[k]
	return v, ok
}

func (c *Cache) Set(k, v string) {
	c.mu.Lock() // exclusivo: espera a que terminen los lectores
	defer c.mu.Unlock()
	c.datos[k] = v
}

// ============ 4) sync.Pool: reutilizar objetos temporales ============

// Un Pool guarda objetos ya creados para reutilizarlos y así reducir el
// trabajo del recolector de basura. El runtime puede vaciarlo en cualquier
// momento, por lo que solo sirve para objetos temporales (buffers), nunca
// para guardar estado.
var creados atomic.Int32

var poolBuffers = sync.Pool{
	New: func() any { // se llama cuando el pool está vacío
		creados.Add(1)
		b := make([]byte, 0, 4096)
		return &b // se guarda un puntero para evitar una asignación extra en Put
	},
}

func procesarConPool(i int) int {
	bp := poolBuffers.Get().(*[]byte)
	buf := (*bp)[:0] // reutilizar el buffer, vaciado
	buf = fmt.Appendf(buf, "petición-%d", i)
	n := len(buf)
	*bp = buf
	poolBuffers.Put(bp) // devolverlo para que otra goroutine lo reutilice
	return n
}

// ============ 5) Worker pool con cancelación ============

type Trabajo struct{ ID, Valor int }
type Resultado struct {
	TrabajoID int
	Salida    int
	Worker    int
}

// workerPool reparte los trabajos entre numWorkers goroutines. Tres roles:
// un productor que llena "entrada", los workers, y una goroutine que cierra
// "salida" cuando todos los workers terminaron.
func workerPool(ctx context.Context, numWorkers int, trabajos []Trabajo) []Resultado {
	entrada := make(chan Trabajo)
	salida := make(chan Resultado)

	var wg sync.WaitGroup
	for w := 1; w <= numWorkers; w++ {
		wg.Go(func() {
			for t := range entrada {
				time.Sleep(time.Duration(rand.IntN(20)) * time.Millisecond) // simular trabajo
				select {
				case salida <- Resultado{TrabajoID: t.ID, Salida: t.Valor * t.Valor, Worker: w}:
				case <-ctx.Done(): // si se cancela, dejar de trabajar
					return
				}
			}
		})
	}

	// Productor: alimenta la cola y la cierra al terminar
	go func() {
		defer close(entrada)
		for _, t := range trabajos {
			select {
			case entrada <- t:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Cerrar "salida" cuando todos los workers terminaron; así el for-range
	// de abajo sabe cuándo parar
	go func() {
		wg.Wait()
		close(salida)
	}()

	var res []Resultado
	for r := range salida {
		res = append(res, r)
	}
	return res
}

// ============ 6) Semáforo ponderado ============

// semaphore.Weighted (golang.org/x/sync) limita el uso de un recurso con
// "peso". Aquí simula memoria: un trabajo grande ocupa más unidades que uno
// chico, y la suma nunca supera la capacidad.
func conSemaforo() {
	sem := semaphore.NewWeighted(10) // capacidad total: 10 "MB"
	ctx := context.Background()
	var wg sync.WaitGroup
	var enUso, maxUso atomic.Int64
	for _, peso := range []int64{6, 3, 5, 2, 4, 1} {
		wg.Go(func() {
			// Acquire espera hasta que haya capacidad; solo falla si ctx se cancela
			if err := sem.Acquire(ctx, peso); err != nil {
				fmt.Println("   no se pudo adquirir:", err)
				return
			}
			defer sem.Release(peso)

			actual := enUso.Add(peso)
			// Registrar el máximo sin mutex usando compare-and-swap (CAS):
			// "si maxUso todavía vale m, cámbialo por actual"; si otra goroutine
			// lo cambió en el medio, se reintenta.
			for {
				m := maxUso.Load()
				if actual <= m || maxUso.CompareAndSwap(m, actual) {
					break
				}
			}
			time.Sleep(30 * time.Millisecond)
			enUso.Add(-peso)
		})
	}
	wg.Wait()
	fmt.Println("   uso máximo simultáneo:", maxUso.Load(), "de 10")
}

// ============ 7) sync.Cond: esperar a que se cumpla una condición ============

// Una barrera de largada: todos los corredores esperan la señal.
// En la práctica los channels cubren casi todos los casos (cerrar un
// channel también "despierta a todos"); Cond es útil cuando muchas
// goroutines esperan cambios repetidos en un estado compartido.
func largada() {
	var mu sync.Mutex
	cond := sync.NewCond(&mu)
	listo := false
	var wg sync.WaitGroup
	var orden []int
	var muOrden sync.Mutex

	for i := 1; i <= 4; i++ {
		wg.Go(func() {
			mu.Lock()
			// Wait siempre dentro de un for: al despertar hay que volver a
			// verificar la condición, porque pudo cambiar otra vez
			for !listo {
				cond.Wait() // libera mu mientras espera y lo vuelve a tomar al despertar
			}
			mu.Unlock()
			muOrden.Lock()
			orden = append(orden, i)
			muOrden.Unlock()
		})
	}
	time.Sleep(30 * time.Millisecond)
	fmt.Println("   todos los corredores esperando... ¡largada!")
	mu.Lock()
	listo = true
	cond.Broadcast() // despierta a TODOS (Signal despertaría solo a uno)
	mu.Unlock()
	wg.Wait()
	slices.Sort(orden)
	fmt.Println("   corredores que largaron:", orden)
}

func main() {
	soloCarrera := flag.Bool("carrera", false, "ejecutar solo el ejemplo con condición de carrera (para -race)")
	flag.Parse()
	if *soloCarrera {
		fmt.Println("contador con carrera:", carrera(1000))
		return
	}

	fmt.Println("1) Condición de carrera (1000 goroutines incrementan un contador)")
	fmt.Println("   sin sincronizar:", carrera(1000), "(puede dar menos de 1000; -race lo detecta)")
	fmt.Println("   con atomic:     ", conAtomic(1000))

	fmt.Println("\n2) sync.OnceValue con 5 goroutines simultáneas")
	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() { cargarConfig() })
	}
	wg.Wait()
	primera, segunda := cargarConfig(), cargarConfig()
	fmt.Println("   config:", primera.URL, "| misma instancia:", primera == segunda)

	fmt.Println("\n3) RWMutex: 100 lectores y 1 escritor concurrentes")
	cache := &Cache{datos: map[string]string{"idioma": "es"}}
	var lecturas atomic.Int32
	for range 100 {
		wg.Go(func() {
			if _, ok := cache.Get("idioma"); ok {
				lecturas.Add(1)
			}
		})
	}
	wg.Go(func() { cache.Set("idioma", "en") })
	wg.Wait()
	v, _ := cache.Get("idioma")
	fmt.Printf("   lecturas exitosas: %d, valor final: %s\n", lecturas.Load(), v)

	fmt.Println("\n4) sync.Pool: 1000 usos de buffers desde 8 goroutines")
	for g := range 8 {
		wg.Go(func() {
			for i := range 125 {
				procesarConPool(g*125 + i)
			}
		})
	}
	wg.Wait()
	fmt.Printf("   buffers creados: %d (en vez de 1000)\n", creados.Load())

	fmt.Println("\n5) Worker pool: 10 trabajos, 3 workers")
	var trabajos []Trabajo
	for i := 1; i <= 10; i++ {
		trabajos = append(trabajos, Trabajo{ID: i, Valor: i})
	}
	res := workerPool(context.Background(), 3, trabajos)
	porWorker := map[int]int{}
	for _, r := range res {
		porWorker[r.Worker]++
	}
	// Los resultados llegan en cualquier orden: se ordenan para mostrarlos
	slices.SortFunc(res, func(a, b Resultado) int { return cmp.Compare(a.TrabajoID, b.TrabajoID) })
	fmt.Printf("   %d resultados, primeros: %+v\n", len(res), res[:2])
	fmt.Println("   trabajos por worker (varía en cada ejecución):", porWorker)

	fmt.Println("\n6) Semáforo ponderado (capacidad 10, pesos 6,3,5,2,4,1)")
	conSemaforo()

	fmt.Println("\n7) sync.Cond con Broadcast")
	largada()

	// atomic.Pointer permite que muchas goroutines lean una configuración
	// mientras otra la reemplaza completa, sin locks: se publica un puntero
	// nuevo y nunca se modifica el objeto ya publicado.
	fmt.Println("\n8) atomic.Pointer: publicar una config nueva sin locks")
	var actual atomic.Pointer[Config]
	actual.Store(&Config{URL: "v1"})
	viejo := actual.Swap(&Config{URL: "v2"}) // reemplazo atómico; devuelve el anterior
	fmt.Println("   anterior:", viejo.URL, "| actual:", actual.Load().URL)
}
