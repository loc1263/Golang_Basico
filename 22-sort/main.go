// Tema 22: Ordenar slices (paquetes slices y sort)
//
// Desde Go 1.21 la forma recomendada de ordenar es el paquete "slices",
// genérico, más simple y más rápido. El paquete "sort" es la forma clásica:
// sigue funcionando y lo verás en mucho código existente.
//
// Qué aprenderás:
//   - slices.Sort para tipos básicos.
//   - slices.SortFunc con cmp.Compare para ordenar structs por uno o varios criterios.
//   - Orden estable, búsqueda binaria y la interfaz clásica sort.Interface.
//
// Ejecutar con: go run ./22-sort
package main

import (
	"cmp"
	"fmt"
	"slices"
	"sort"
	"strings"
)

type Persona struct {
	Nombre string
	Edad   int
}

// Forma clásica: implementar sort.Interface (Len, Less y Swap)
type PorEdad []Persona

func (p PorEdad) Len() int           { return len(p) }
func (p PorEdad) Less(i, j int) bool { return p[i].Edad < p[j].Edad }
func (p PorEdad) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func main() {
	// slices.Sort ordena "en el lugar": modifica el slice recibido
	numeros := []int{5, 2, 8, 1, 9, 3}
	slices.Sort(numeros)
	fmt.Println("Enteros ordenados:", numeros)

	textos := []string{"banana", "manzana", "cereza"}
	slices.Sort(textos)
	fmt.Println("Strings ordenados:", textos)

	fmt.Println("¿numeros está ordenado?", slices.IsSorted(numeros))

	// Para no modificar el original: ordenar una copia
	original := []int{3, 1, 2}
	ordenada := slices.Sorted(slices.Values(original)) // devuelve un slice nuevo
	fmt.Println("Original intacto:", original, "- copia ordenada:", ordenada)

	// slices.SortFunc recibe una función de comparación que devuelve:
	//   negativo si a va antes que b, 0 si son iguales, positivo si va después.
	// cmp.Compare(a, b) calcula exactamente eso para números y strings.
	personas := []Persona{
		{"Ana", 30},
		{"Luis", 25},
		{"María", 40},
		{"Bruno", 30},
	}

	slices.SortFunc(personas, func(a, b Persona) int {
		return cmp.Compare(a.Edad, b.Edad)
	})
	fmt.Println("\nPor edad:", personas)

	slices.SortFunc(personas, func(a, b Persona) int {
		return strings.Compare(a.Nombre, b.Nombre)
	})
	fmt.Println("Por nombre:", personas)

	// Orden descendente: invertir los argumentos
	slices.SortFunc(personas, func(a, b Persona) int {
		return cmp.Compare(b.Edad, a.Edad)
	})
	fmt.Println("Por edad descendente:", personas)

	// Varios criterios: cmp.Or devuelve el primer resultado distinto de 0.
	// Aquí: por edad y, a igual edad, por nombre.
	slices.SortFunc(personas, func(a, b Persona) int {
		return cmp.Or(
			cmp.Compare(a.Edad, b.Edad),
			cmp.Compare(a.Nombre, b.Nombre),
		)
	})
	fmt.Println("Por edad y luego nombre:", personas)

	// SortStableFunc mantiene el orden original entre elementos "iguales"
	// (útil al ordenar en varias pasadas)
	slices.SortStableFunc(personas, func(a, b Persona) int {
		return cmp.Compare(a.Edad/10, b.Edad/10) // agrupar por década
	})
	fmt.Println("Estable por década:", personas)

	// Búsqueda binaria en un slice YA ordenado: muy rápida en listas grandes
	numerosOrdenados := []int{1, 3, 5, 7, 9, 11}
	indice, encontrado := slices.BinarySearch(numerosOrdenados, 7)
	fmt.Println("\nBinarySearch de 7 -> índice:", indice, "encontrado:", encontrado)
	indice, encontrado = slices.BinarySearch(numerosOrdenados, 6)
	fmt.Println("BinarySearch de 6 -> se insertaría en el índice:", indice, "encontrado:", encontrado)

	// --- Forma clásica con el paquete sort ---
	personas2 := []Persona{{"Pedro", 22}, {"Marta", 35}, {"Julio", 28}}
	sort.Sort(PorEdad(personas2))
	fmt.Println("\nCon sort.Sort + sort.Interface:", personas2)

	sort.Slice(personas2, func(i, j int) bool { // compara por índices
		return personas2[i].Nombre < personas2[j].Nombre
	})
	fmt.Println("Con sort.Slice por nombre:", personas2)
}
