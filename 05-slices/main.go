// Tema 5: Slices (listas dinámicas)
//
// Un slice es una "vista" de tamaño variable sobre un array interno. Por
// dentro tiene tres datos: un puntero al array, la longitud (len) y la
// capacidad (cap). Es la estructura de lista que se usa en casi todo el
// código Go.
//
// Qué aprenderás:
//   - Crear slices, agregar elementos con append y sub-slices con [inicio:fin].
//   - La diferencia entre len y cap, y cómo crece un slice.
//   - Por qué dos slices pueden compartir memoria, y cómo evitarlo.
//   - El paquete estándar "slices" (Go 1.21+) con operaciones comunes.
//
// Ejecutar con: go run ./05-slices
package main

import (
	"fmt"
	"slices"
)

func main() {
	// Slice literal: igual que un array, pero sin tamaño entre los corchetes
	frutas := []string{"manzana", "banana", "cereza"}
	fmt.Println("Slice inicial:", frutas, "- longitud:", len(frutas))

	// append agrega elementos y devuelve el slice resultante.
	// Siempre hay que asignar el resultado: frutas = append(frutas, ...)
	frutas = append(frutas, "durazno")
	frutas = append(frutas, "kiwi", "mango") // se pueden agregar varios a la vez
	fmt.Println("Después de append:", frutas)

	// make crea un slice con una longitud inicial (elementos en valor cero)
	numeros := make([]int, 3)
	numeros[0] = 1
	numeros[1] = 2
	numeros[2] = 3
	fmt.Println("\nSlice creado con make:", numeros)

	// Sub-slice con [inicio:fin]: incluye "inicio" y excluye "fin"
	sub := frutas[1:3]
	fmt.Println("\nSub-slice frutas[1:3]:", sub)

	// len = elementos actuales; cap = espacio reservado en el array interno.
	// Cuando append supera la capacidad, Go reserva un array más grande y
	// copia los elementos. Por eso cap crece "a saltos".
	fmt.Println("\nCómo crece la capacidad con append:")
	var crece []int
	for i := range 6 {
		crece = append(crece, i)
		fmt.Printf("len=%d cap=%d\n", len(crece), cap(crece))
	}
	// Si se conoce el tamaño final, reservarlo evita esas copias:
	reservado := make([]int, 0, 100) // len 0, cap 100
	fmt.Println("Slice con capacidad reservada: len", len(reservado), "cap", cap(reservado))

	// Un sub-slice COMPARTE el array interno con el slice original:
	// modificar uno modifica el otro.
	original := []int{1, 2, 3, 4, 5}
	vista := original[1:4]
	vista[0] = 99
	fmt.Println("\nModificar la vista afecta al original:")
	fmt.Println("Original:", original)
	fmt.Println("Vista:", vista)

	// Para obtener una copia independiente: slices.Clone (o make + copy)
	copiaIndependiente := slices.Clone(original)
	copiaIndependiente[0] = -1
	fmt.Println("\nUna copia independiente no afecta al original:")
	fmt.Println("Original:", original)
	fmt.Println("Copia:", copiaIndependiente)

	// Eliminar elementos: slices.Delete(s, i, j) quita los elementos [i, j).
	// Cuidado: trabaja sobre el MISMO array interno, así que modifica el
	// slice que se le pasa. Si necesitas conservar el original, clónalo antes.
	numerosABorrar := []int{10, 20, 30, 40, 50}
	sinElemento := slices.Delete(slices.Clone(numerosABorrar), 2, 3)
	fmt.Println("\nOriginal:", numerosABorrar)
	fmt.Println("Sin el elemento del índice 2:", sinElemento)

	// Otras funciones útiles del paquete slices
	letras := []string{"c", "a", "b"}
	fmt.Println("\n¿Contiene \"b\"?", slices.Contains(letras, "b"))
	fmt.Println("Índice de \"c\":", slices.Index(letras, "c"))
	slices.Sort(letras) // ordena en el lugar (modifica el slice)
	fmt.Println("Ordenado:", letras)
	fmt.Println("Máximo:", slices.Max(letras))
	fmt.Println("Insertar \"z\" en la posición 1:", slices.Insert(letras, 1, "z"))
}
