// Tema 17: Generics (parámetros de tipo)
//
// Los generics permiten escribir funciones y tipos que funcionan con
// distintos tipos de datos sin duplicar código y sin perder la verificación
// de tipos del compilador (a diferencia de usar "any").
//
// Qué aprenderás:
//   - Funciones genéricas y restricciones (constraints): comparable,
//     uniones de tipos, "~" y cmp.Ordered.
//   - Tipos genéricos (una Pila que sirve para cualquier tipo).
//   - Métodos genéricos (Go 1.27+).
//
// Ejecutar con: go run ./17-generics
package main

import (
	"cmp"
	"fmt"
	"strconv"
)

// [T comparable] declara un parámetro de tipo T. La restricción
// "comparable" exige que T admita == y !=.
// (La librería estándar ya trae esta función: slices.Contains, tema 5.
// La escribimos aquí para ver cómo funciona por dentro.)
func Contiene[T comparable](lista []T, buscado T) bool {
	for _, v := range lista {
		if v == buscado {
			return true
		}
	}
	return false
}

// Restricción propia: una interfaz que enumera los tipos permitidos.
// El "~" incluye también los tipos definidos a partir de ellos: con ~int,
// un tipo propio como "type Puntos int" también es válido.
type Numero interface {
	~int | ~int64 | ~float64
}

func Sumar[T Numero](numeros []T) T {
	var total T // valor cero de T (0 para cualquier número)
	for _, n := range numeros {
		total += n
	}
	return total
}

// cmp.Ordered agrupa todos los tipos que admiten < y > (números y strings).
// (Para dos valores, Go ya trae la función incorporada max(a, b).)
func Maximo[T cmp.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// Tipo genérico: una pila (stack) que funciona con cualquier tipo T
type Pila[T any] struct {
	elementos []T
}

func (p *Pila[T]) Apilar(valor T) {
	p.elementos = append(p.elementos, valor)
}

// Desapilar devuelve (valor, true), o (valor cero, false) si está vacía
func (p *Pila[T]) Desapilar() (T, bool) {
	var cero T
	if len(p.elementos) == 0 {
		return cero, false
	}
	ultimo := p.elementos[len(p.elementos)-1]
	p.elementos = p.elementos[:len(p.elementos)-1]
	return ultimo, true
}

// Método genérico (Go 1.27+): además del T de la Pila, el método declara su
// propio parámetro de tipo U. Convierte una Pila[T] en una Pila[U].
func (p *Pila[T]) Transformar[U any](f func(T) U) *Pila[U] {
	resultado := &Pila[U]{}
	for _, e := range p.elementos {
		resultado.Apilar(f(e))
	}
	return resultado
}

type Puntos int // tipo propio basado en int (aceptado gracias a ~int)

func main() {
	// El compilador deduce T a partir de los argumentos (inferencia de tipos)
	nombres := []string{"Ana", "Luis", "María"}
	fmt.Println("¿Contiene 'Luis'?", Contiene(nombres, "Luis"))

	numeros := []int{1, 2, 3, 4, 5}
	fmt.Println("¿Contiene 10?", Contiene(numeros, 10))

	// También se puede indicar T explícitamente
	fmt.Println("¿Contiene 2.5?", Contiene[float64]([]float64{1.5, 2.5}, 2.5))

	fmt.Println("\nSuma de enteros:", Sumar([]int{1, 2, 3}))
	fmt.Println("Suma de flotantes:", Sumar([]float64{1.5, 2.5, 3.0}))
	fmt.Println("Suma de Puntos (tipo propio):", Sumar([]Puntos{10, 20}))

	fmt.Println("\nMáximo entre 3 y 7:", Maximo(3, 7))
	fmt.Println("Máximo entre 'pera' y 'manzana':", Maximo("pera", "manzana"))
	fmt.Println("Con la función incorporada max:", max(3, 7, 5))

	// La misma Pila sirve para enteros...
	var pilaEnteros Pila[int]
	pilaEnteros.Apilar(1)
	pilaEnteros.Apilar(2)
	pilaEnteros.Apilar(3)
	valor, ok := pilaEnteros.Desapilar()
	fmt.Println("\nDesapilado de pila de enteros:", valor, ok)

	// ...y para strings, sin duplicar código
	var pilaTextos Pila[string]
	pilaTextos.Apilar("primero")
	pilaTextos.Apilar("segundo")
	textoDesapilado, _ := pilaTextos.Desapilar()
	fmt.Println("Desapilado de pila de strings:", textoDesapilado)

	// Método genérico: de Pila[int] a Pila[string] usando strconv.Itoa
	pilaComoTexto := pilaEnteros.Transformar(strconv.Itoa)
	fmt.Printf("Transformar(strconv.Itoa): %T %q\n", pilaComoTexto, pilaComoTexto.elementos)

	// Consejo: usa generics cuando escribirías el MISMO código para varios
	// tipos. Si la lógica cambia según el tipo, suele ser mejor una interfaz.
}
