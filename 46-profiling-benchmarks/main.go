// Tema 46: Profiling y benchmarking (testing.B, pprof)
// Regla de oro: "medir antes de optimizar". Go trae todo lo necesario:
//   - Benchmarks (testing.B) en archivos _test.go: miden tiempo y memoria
//     por operación. Ver main_test.go.
//   - pprof: perfiles de CPU y memoria para encontrar DÓNDE se va el tiempo.
//
// Comandos (desde la raíz del proyecto):
//
//	go run ./46-profiling-benchmarks                 genera cpu.prof y mem.prof
//	go test -bench=. -benchmem ./46-profiling-benchmarks
//	go test -bench=Concatenar -count=5 ./46-profiling-benchmarks   (repetir para ver variación)
//	go tool pprof -top 46-profiling-benchmarks/cpu.prof            (funciones más costosas)
//	go tool pprof -top -cum 46-profiling-benchmarks/cpu.prof       (ordenado por tiempo acumulado)
//	go tool pprof -top -sample_index=alloc_space 46-profiling-benchmarks/mem.prof   (quién asigna memoria)
//	go tool pprof -http=:8082 46-profiling-benchmarks/cpu.prof     (interfaz web con flame graph)
//	go test -bench=. -cpuprofile=cpu.prof ./46-profiling-benchmarks (perfil de un benchmark)
//
// Para servidores en producción: importar _ "net/http/pprof" expone
// /debug/pprof/ y permite perfilar el proceso en vivo (ver abajo).
package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ---------- Funciones a comparar (usadas también en main_test.go) ----------

// ConcatenarConMas crea un string nuevo en cada iteración: O(n²) en copias
func ConcatenarConMas(partes []string) string {
	s := ""
	for _, p := range partes {
		s += p
	}
	return s
}

// ConcatenarConBuilder reutiliza un buffer que crece de a poco
func ConcatenarConBuilder(partes []string) string {
	var b strings.Builder
	for _, p := range partes {
		b.WriteString(p)
	}
	return b.String()
}

// ConcatenarConBuilderGrow reserva de antemano el tamaño exacto: 1 sola asignación
func ConcatenarConBuilderGrow(partes []string) string {
	total := 0
	for _, p := range partes {
		total += len(p)
	}
	var b strings.Builder
	b.Grow(total)
	for _, p := range partes {
		b.WriteString(p)
	}
	return b.String()
}

// SliceSinCapacidad hace crecer el slice con append (re-asignaciones)
func SliceSinCapacidad(n int) []int {
	var s []int
	for i := range n {
		s = append(s, i)
	}
	return s
}

// SliceConCapacidad reserva la capacidad con make desde el inicio
func SliceConCapacidad(n int) []int {
	s := make([]int, 0, n)
	for i := range n {
		s = append(s, i)
	}
	return s
}

// Fibonacci recursivo (lento a propósito) para tener algo que "pese" en el perfil de CPU
func FibRecursivo(n int) int {
	if n < 2 {
		return n
	}
	return FibRecursivo(n-1) + FibRecursivo(n-2)
}

func FibIterativo(n int) int {
	a, b := 0, 1
	for range n {
		a, b = b, a+b
	}
	return a
}

// ---------- Trabajo a perfilar ----------

// sumidero guarda resultados que no se usan. Si el resultado de una función
// sin efectos secundarios se descarta, el compilador podría eliminar la
// llamada y estaríamos midiendo... nada. Asignarlo a una variable de
// paquete lo impide. (En los benchmarks, b.Loop se encarga de esto solo.)
var sumidero int

func cargaDeTrabajo() {
	// Mezcla de CPU (fib) y asignaciones de memoria (strings)
	for range 3 {
		sumidero += FibRecursivo(35)
	}
	partes := make([]string, 3000)
	for i := range partes {
		partes[i] = strconv.Itoa(i)
	}
	for range 100 {
		ConcatenarConMas(partes)
	}
	palabras := strings.Fields(strings.Repeat("go es simple y rápido ", 50000))
	sort.Strings(palabras)
}

func medir(nombre string, f func()) {
	inicio := time.Now()
	f()
	fmt.Printf("  %-28s %v\n", nombre, time.Since(inicio).Round(time.Microsecond))
}

func main() {
	dir := "46-profiling-benchmarks"

	// 1) Perfil de CPU: muestrea ~100 veces por segundo qué función se ejecuta
	fCPU, err := os.Create(dir + "/cpu.prof")
	if err != nil {
		log.Fatal(err)
	}
	if err := pprof.StartCPUProfile(fCPU); err != nil {
		log.Fatal(err)
	}
	fmt.Println("1) Ejecutando carga de trabajo con perfil de CPU activo...")
	medir("carga total", cargaDeTrabajo)
	pprof.StopCPUProfile()
	fCPU.Close()

	// 2) Perfil de memoria (heap): qué código asignó memoria
	fMem, err := os.Create(dir + "/mem.prof")
	if err != nil {
		log.Fatal(err)
	}
	runtime.GC() // datos de asignación actualizados
	if err := pprof.WriteHeapProfile(fMem); err != nil {
		log.Fatal(err)
	}
	fMem.Close()
	fmt.Println("   perfiles escritos en", dir+"/cpu.prof y", dir+"/mem.prof")
	fmt.Println("   analizar con: go tool pprof -top " + dir + "/cpu.prof")

	// 3) Mediciones rápidas "a mano" (útiles, pero menos precisas que los
	// benchmarks: una sola corrida, sin calentar, sin contar memoria).
	// En Windows el reloj puede tener poca resolución y mostrar 0s para
	// operaciones muy rápidas: otra razón para preferir benchmarks.
	fmt.Println("\n2) Comparación rápida con time.Since:")
	partes := make([]string, 5000)
	for i := range partes {
		partes[i] = "x"
	}
	medir("ConcatenarConMas", func() { ConcatenarConMas(partes) })
	medir("ConcatenarConBuilder", func() { ConcatenarConBuilder(partes) })
	medir("FibRecursivo(32)", func() { sumidero += FibRecursivo(32) })
	medir("FibIterativo(32)", func() { sumidero += FibIterativo(32) })
	fmt.Println("  (suma de los resultados de Fibonacci:", sumidero, ")")

	// 4) Estadísticas de memoria del runtime
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Println("\n3) runtime.MemStats:")
	fmt.Printf("  memoria asignada total: %d MB\n", m.TotalAlloc/1024/1024)
	fmt.Printf("  heap en uso ahora:      %d KB\n", m.HeapAlloc/1024)
	fmt.Printf("  ciclos de GC:           %d\n", m.NumGC)
	fmt.Printf("  goroutines activas:     %d\n", runtime.NumGoroutine())

	fmt.Println("\nSiguiente paso: go test -bench=. -benchmem ./" + dir)

	// Perfilado en vivo de un servidor (no se ejecuta aquí):
	//
	//   import _ "net/http/pprof"
	//   go http.ListenAndServe("localhost:6060", nil)
	//
	//   go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
	//   go tool pprof http://localhost:6060/debug/pprof/heap
	//   curl http://localhost:6060/debug/pprof/goroutine?debug=1   (detectar goroutines colgadas)
	//
	// No lo expongas públicamente: muestra información interna del proceso.
}
