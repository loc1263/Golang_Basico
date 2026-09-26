// Tema 4: Arrays
//
// Un array tiene un tamaño FIJO que forma parte de su tipo: [3]int y [4]int
// son tipos distintos. En la práctica se usan poco de forma directa; lo
// habitual es usar slices (tema 5), que están construidos sobre arrays.
//
// Qué aprenderás:
//   - Declarar e inicializar arrays.
//   - Que los arrays se copian completos al asignarlos (semántica de valor).
//   - Arrays de varias dimensiones.
//
// Ejecutar con: go run ./04-arrays
package main

import "fmt"

func main() {
	// Array de 5 enteros: todos empiezan con el valor cero (0)
	var numeros [5]int
	numeros[0] = 10
	numeros[1] = 20
	fmt.Println("Array con ceros y asignaciones:", numeros)

	// Declaración con valores iniciales
	dias := [3]string{"lunes", "martes", "miércoles"}
	fmt.Println("Array de strings:", dias)

	// "..." le pide al compilador que cuente los elementos
	colores := [...]string{"rojo", "verde", "azul"}
	fmt.Println("Array con tamaño inferido:", colores, "- longitud:", len(colores))

	// Acceder fuera del rango es un error: con un índice constante lo detecta
	// el compilador; con un índice variable, provoca un panic al ejecutar.
	// colores[3] = "negro" // error de compilación: índice 3 fuera de rango

	// Recorrer un array con for-range
	suma := 0
	valores := [4]int{1, 2, 3, 4}
	for _, v := range valores {
		suma += v
	}
	fmt.Println("Suma de valores:", suma)

	// Los arrays son VALORES: al asignarlos (o pasarlos a una función) se
	// copian todos sus elementos. Modificar la copia no afecta al original.
	original := [3]int{1, 2, 3}
	copia := original
	copia[0] = 99
	fmt.Println("\nOriginal no cambia al modificar la copia:")
	fmt.Println("Original:", original)
	fmt.Println("Copia:", copia)

	// Dos arrays del mismo tipo se pueden comparar con ==
	fmt.Println("¿original == [3]int{1, 2, 3}?", original == [3]int{1, 2, 3})

	// Array de dos dimensiones (matriz de 2 filas x 3 columnas)
	var matriz [2][3]int
	for i := range matriz { // i recorre las filas
		for j := range matriz[i] { // j recorre las columnas de la fila i
			matriz[i][j] = i + j
		}
	}
	fmt.Println("\nMatriz 2x3:", matriz)
}
