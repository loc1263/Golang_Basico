// Tema 29: Reflection (paquete reflect)
//
// Reflection permite examinar el tipo y el valor de una variable en tiempo
// de ejecución, sin conocerlos al compilar. Es lo que usan por dentro
// encoding/json (para leer los tags), fmt (para imprimir cualquier valor) o
// los validadores (tema 39).
//
// Es una herramienta avanzada: el código con reflection es más lento, más
// difícil de leer y sus errores aparecen al ejecutar (panics) en lugar de al
// compilar. En código de aplicación casi siempre hay una alternativa mejor:
// interfaces o generics.
//
// Qué aprenderás:
//   - reflect.TypeOf / reflect.ValueOf, y la diferencia entre Type y Kind.
//   - Recorrer los campos de un struct y leer sus tags.
//   - Modificar un valor a través de reflection.
//
// Ejecutar con: go run ./29-reflection
package main

import (
	"fmt"
	"reflect"
)

type Persona struct {
	Nombre string `etiqueta:"nombre_completo"`
	Edad   int    `etiqueta:"edad_anios"`
}

func (p Persona) Saludar() string {
	return "Hola, soy " + p.Nombre
}

type Temperatura float64

func main() {
	// TypeOf: información sobre el TIPO. ValueOf: el VALOR envuelto.
	var x float64 = 3.14
	fmt.Println("Tipo de x:", reflect.TypeOf(x))
	fmt.Println("Valor de x:", reflect.ValueOf(x))

	// Type vs Kind: el Type es el tipo concreto (main.Temperatura); el Kind
	// es la categoría básica sobre la que está construido (float64).
	var t Temperatura = 21.5
	fmt.Println("\nType de t:", reflect.TypeOf(t), "| Kind de t:", reflect.TypeOf(t).Kind())

	// Recorrer los campos de un struct
	p := Persona{Nombre: "Ana", Edad: 30}
	tipoStruct := reflect.TypeOf(p)
	valorStruct := reflect.ValueOf(p)

	fmt.Println("\nInspeccionando el struct Persona:")
	fmt.Println("Nombre del tipo:", tipoStruct.Name())
	fmt.Println("Cantidad de campos:", tipoStruct.NumField())

	for i := range tipoStruct.NumField() {
		campo := tipoStruct.Field(i)       // descripción del campo (nombre, tipo, tags)
		valorCampo := valorStruct.Field(i) // su valor en "p"
		etiqueta := campo.Tag.Get("etiqueta")
		fmt.Printf("Campo %d: %s (tipo %s) = %v | tag etiqueta=%q\n",
			i, campo.Name, campo.Type, valorCampo, etiqueta)
	}

	// Métodos del tipo. Methods() devuelve un iterador (Go 1.26+); la forma
	// clásica es for i := range t.NumMethod() { t.Method(i) }
	fmt.Println("\nMétodos de Persona:")
	for metodo := range tipoStruct.Methods() {
		fmt.Println("-", metodo.Name, metodo.Type)
	}

	// Llamar un método por su nombre (así funcionan algunos frameworks)
	resultado := valorStruct.MethodByName("Saludar").Call(nil)
	fmt.Println("Resultado de llamar Saludar por reflection:", resultado[0])

	// Para MODIFICAR un valor se necesita un puntero y Elem(). Con
	// reflect.ValueOf(n) se recibiría una copia "no modificable", y llamar a
	// SetInt sobre ella provocaría un panic.
	n := 10
	valorPuntero := reflect.ValueOf(&n).Elem() // Elem() accede al valor apuntado
	fmt.Println("\n¿Se puede modificar?", valorPuntero.CanSet())
	fmt.Println("Valor antes de modificar con reflection:", n)
	valorPuntero.SetInt(99)
	fmt.Println("Valor después de modificar con reflection:", n)

	// Una función que acepta cualquier valor y describe su tipo
	describirValor(42)
	describirValor("hola")
	describirValor(true)
	describirValor(p)
	describirValor(nil)
}

func describirValor(v any) {
	t := reflect.TypeOf(v)
	if t == nil { // TypeOf(nil) devuelve nil: hay que verificarlo
		fmt.Println("\nValor nil: no tiene tipo")
		return
	}
	fmt.Printf("\nValor: %v | Tipo: %v | Kind: %v\n", v, t, t.Kind())
}
