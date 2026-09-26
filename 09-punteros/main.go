// Tema 9: Punteros
//
// Un puntero guarda la DIRECCIÓN de memoria de una variable, no su valor.
// Sirven para que una función pueda modificar una variable de quien la
// llama, y para evitar copiar datos grandes.
// A diferencia de C, Go no permite aritmética de punteros y libera la
// memoria automáticamente (recolector de basura).
//
// Qué aprenderás:
//   - Los operadores & (dirección de) y * (valor apuntado).
//   - Pasar valores vs. pasar punteros a una función.
//   - new, y el valor nil de un puntero.
//
// Ejecutar con: go run ./09-punteros
package main

import "fmt"

type Contador struct {
	Valor int
}

// Recibe una COPIA del struct: los cambios no afectan al original
func incrementarPorValor(c Contador) {
	c.Valor++
}

// Recibe un PUNTERO: los cambios sí afectan al original
func incrementarPorPuntero(c *Contador) {
	c.Valor++ // Go permite escribir c.Valor en vez de (*c).Valor
}

func main() {
	x := 10

	// "&x" es la dirección de memoria de x. Su tipo es *int ("puntero a int").
	punteroX := &x
	fmt.Println("Valor de x:", x)
	fmt.Println("Dirección de memoria de x:", punteroX)

	// "*punteroX" (desreferenciar) lee o modifica el valor apuntado
	fmt.Println("Valor apuntado por punteroX:", *punteroX)
	*punteroX = 20
	fmt.Println("Nuevo valor de x tras modificarlo vía puntero:", x)

	// Pasar por valor vs. pasar un puntero
	c := Contador{Valor: 0}

	incrementarPorValor(c)
	fmt.Println("\nDespués de incrementarPorValor:", c.Valor) // sigue en 0

	incrementarPorPuntero(&c)
	fmt.Println("Después de incrementarPorPuntero:", c.Valor) // ahora es 1

	// Forma más común de obtener un puntero a un struct nuevo: &T{...}
	otro := &Contador{Valor: 5}
	fmt.Println("\nContador creado con &Contador{}:", otro.Valor)

	// new(T) reserva un valor cero de tipo T y devuelve un puntero a él
	nuevoContador := new(Contador)
	nuevoContador.Valor = 100
	fmt.Println("Contador creado con new:", *nuevoContador)

	// Go 1.26+: new también acepta una expresión y devuelve un puntero a una
	// copia de ese valor. Útil para campos opcionales de tipo *int, *string...
	limite := new(50)
	fmt.Println("Puntero creado con new(50):", *limite)

	// El valor cero de un puntero es nil: no apunta a nada.
	// Desreferenciar un puntero nil provoca un panic en tiempo de ejecución,
	// por eso hay que verificarlo cuando puede ser nil.
	var punteroNil *Contador
	fmt.Println("\n¿El puntero es nil?", punteroNil == nil)
	if punteroNil != nil {
		fmt.Println(punteroNil.Valor)
	} else {
		fmt.Println("No se accede: el puntero es nil")
	}
}
