package main

import (
	"fmt"
	"strconv"
	"testing"
)

// Un benchmark: función BenchmarkXxx(b *testing.B). El framework la ejecuta
// repetidas veces ajustando la cantidad de iteraciones hasta obtener una
// medición estable, e informa ns/op (y con -benchmem: B/op y allocs/op).
//
// Desde Go 1.24 el bucle idiomático es `for b.Loop() { ... }`:
//   - el código ANTES del bucle (preparación) no se mide
//   - evita que el compilador elimine como "código muerto" la llamada medida
// (Antes se usaba `for i := 0; i < b.N; i++`, que todavía verás mucho.)

var partes = func() []string {
	p := make([]string, 1000)
	for i := range p {
		p[i] = strconv.Itoa(i)
	}
	return p
}()

func BenchmarkConcatenarConMas(b *testing.B) {
	for b.Loop() {
		ConcatenarConMas(partes)
	}
}

func BenchmarkConcatenarConBuilder(b *testing.B) {
	for b.Loop() {
		ConcatenarConBuilder(partes)
	}
}

func BenchmarkConcatenarConBuilderGrow(b *testing.B) {
	b.ReportAllocs() // reportar memoria aunque no se pase -benchmem
	for b.Loop() {
		ConcatenarConBuilderGrow(partes)
	}
}

// Sub-benchmarks con b.Run: el mismo código con distintos tamaños de entrada
// para ver cómo ESCALA (muy útil para detectar O(n²))
func BenchmarkSlices(b *testing.B) {
	for _, n := range []int{100, 10_000, 1_000_000} {
		b.Run(fmt.Sprintf("sin-capacidad/n=%d", n), func(b *testing.B) {
			for b.Loop() {
				SliceSinCapacidad(n)
			}
		})
		b.Run(fmt.Sprintf("con-capacidad/n=%d", n), func(b *testing.B) {
			for b.Loop() {
				SliceConCapacidad(n)
			}
		})
	}
}

func BenchmarkFib(b *testing.B) {
	b.Run("recursivo", func(b *testing.B) {
		for b.Loop() {
			FibRecursivo(20)
		}
	})
	b.Run("iterativo", func(b *testing.B) {
		for b.Loop() {
			FibIterativo(20)
		}
	})
}

// Los benchmarks conviven con tests normales en el mismo archivo.
// Verificar que las optimizaciones no cambian el resultado es fundamental.
func TestConcatenacionesEquivalentes(t *testing.T) {
	a := ConcatenarConMas(partes)
	if b := ConcatenarConBuilder(partes); a != b {
		t.Error("Builder produce un resultado distinto")
	}
	if c := ConcatenarConBuilderGrow(partes); a != c {
		t.Error("BuilderGrow produce un resultado distinto")
	}
	if FibRecursivo(25) != FibIterativo(25) {
		t.Error("las versiones de Fibonacci no coinciden")
	}
}

// Para comparar dos versiones de forma estadística:
//   go test -bench=. -count=10 ./46-profiling-benchmarks > antes.txt
//   (cambiar el código)
//   go test -bench=. -count=10 ./46-profiling-benchmarks > despues.txt
//   go run golang.org/x/perf/cmd/benchstat@latest antes.txt despues.txt
