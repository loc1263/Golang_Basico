// Tema 28: Labels (etiquetas) en loops y goto
//
// Por defecto, "break" y "continue" afectan solo al loop más interno. Una
// etiqueta permite que actúen sobre un loop EXTERNO. "goto" salta
// directamente a una etiqueta.
//
// Qué aprenderás:
//   - break y continue con etiqueta.
//   - El caso de uso típico: buscar en una matriz y salir de ambos loops.
//   - goto, y por qué casi no se usa.
//
// Ejecutar con: go run ./28-labels-goto
package main

import "fmt"

func main() {
	// Sin etiqueta: break solo rompe el loop interno; el externo continúa
	fmt.Println("Break normal (solo rompe el loop interno):")
	for i := range 3 {
		for j := range 3 {
			if j == 1 {
				break
			}
			fmt.Printf("i=%d j=%d\n", i, j)
		}
	}

	// Con etiqueta: "break Externo" termina el loop marcado con esa etiqueta.
	// La etiqueta se escribe justo antes del for, seguida de ":".
	fmt.Println("\nBreak con etiqueta (rompe el loop externo):")
Externo:
	for i := range 3 {
		for j := range 3 {
			if i == 1 && j == 1 {
				break Externo // termina ambos loops
			}
			fmt.Printf("i=%d j=%d\n", i, j)
		}
	}

	// "continue Etiqueta" pasa directamente a la siguiente iteración del
	// loop externo
	fmt.Println("\nContinue con etiqueta:")
Busqueda:
	for i := range 3 {
		for j := range 3 {
			if j == 1 {
				continue Busqueda // siguiente i, sin terminar los j restantes
			}
			fmt.Printf("i=%d j=%d\n", i, j)
		}
	}

	// Caso de uso típico: buscar un valor en una matriz y dejar de buscar
	// en cuanto se encuentra
	matriz := [][]int{
		{1, 2, 3},
		{4, 5, 6},
		{7, 8, 9},
	}
	buscado := 5
	encontrado := false

Buscar:
	for fila := range matriz {
		for columna := range matriz[fila] {
			if matriz[fila][columna] == buscado {
				fmt.Printf("\nEncontrado %d en fila %d, columna %d\n", buscado, fila, columna)
				encontrado = true
				break Buscar
			}
		}
	}
	if !encontrado {
		fmt.Println("\nNo se encontró el valor")
	}
	// Alternativa, a menudo más clara: poner la búsqueda en una función y
	// usar "return" al encontrar el valor.

	// goto salta a una etiqueta dentro de la misma función. Existe, pero en
	// Go idiomático casi no se usa: dificulta seguir el flujo del programa.
	// Un loop "for" expresa lo mismo con más claridad. Se muestra solo para
	// que lo reconozcas si lo encuentras en código generado o de bajo nivel.
	fmt.Println("\nEjemplo de goto:")
	contador := 0
Reintentar:
	contador++
	fmt.Println("Intento número", contador)
	if contador < 3 {
		goto Reintentar
	}
	fmt.Println("Listo, se realizaron", contador, "intentos")
}
