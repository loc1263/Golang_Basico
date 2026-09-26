// Tema 3: Ciclos (for)
//
// Go tiene una sola palabra para repetir código: "for". No existen "while"
// ni "do-while"; todas esas variantes se escriben con for.
//
// Qué aprenderás:
//   - for clásico, for estilo "while" y for infinito con break.
//   - continue para saltar una iteración.
//   - for-range sobre números, slices y strings.
//
// Ejecutar con: go run ./03-ciclos
package main

import "fmt"

func main() {
	// for clásico: inicialización; condición; incremento
	fmt.Println("For clásico:")
	for i := 0; i < 5; i++ {
		fmt.Println(i)
	}

	// for-range sobre un número (Go 1.22+): recorre 0, 1, 2, ..., n-1.
	// Es la forma más simple de repetir algo n veces.
	fmt.Println("\nFor-range sobre un número:")
	for i := range 3 {
		fmt.Println("vuelta", i)
	}

	// for como "while": solo con condición
	fmt.Println("\nFor como while:")
	contador := 0
	for contador < 3 {
		fmt.Println("contador =", contador)
		contador++
	}

	// for infinito: se sale con break (o con return)
	fmt.Println("\nFor infinito con break:")
	n := 0
	for {
		if n >= 3 {
			break
		}
		fmt.Println("n =", n)
		n++
	}

	// continue: termina la iteración actual y pasa a la siguiente
	fmt.Println("\nFor con continue (solo pares):")
	for i := range 10 {
		if i%2 != 0 {
			continue
		}
		fmt.Println(i)
	}

	// for-range sobre una colección (slice, array, map, string, channel).
	// Devuelve dos valores: el índice y una COPIA del elemento.
	fmt.Println("\nFor-range sobre un slice:")
	frutas := []string{"manzana", "banana", "cereza"}
	for indice, valor := range frutas {
		fmt.Printf("%d: %s\n", indice, valor)
	}

	// "_" (identificador en blanco) descarta un valor que no se necesita.
	// Go no permite variables declaradas y sin usar, por eso existe "_".
	fmt.Println("\nFor-range solo valores:")
	for _, valor := range frutas {
		fmt.Println(valor)
	}

	// for-range sobre un string recorre caracteres (runas, ver tema 15).
	// Atención: el índice es la posición en BYTES, no en caracteres.
	// La "é" ocupa 2 bytes en UTF-8, por eso no aparece la posición 4.
	fmt.Println("\nFor-range sobre un string:")
	for i, c := range "café!" {
		fmt.Printf("byte %d: %c\n", i, c)
	}
}
