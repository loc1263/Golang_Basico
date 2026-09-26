// Tema 8: Funciones
//
// Qué aprenderás:
//   - Parámetros, valores de retorno múltiples y retornos con nombre.
//   - Funciones variádicas (con cantidad variable de argumentos).
//   - Funciones como valores: anónimas, como parámetros y closures.
//
// Ejecutar con: go run ./08-funciones
package main

import (
	"errors"
	"fmt"
)

// Función con un parámetro y un valor de retorno
func saludar(nombre string) string {
	return "Hola, " + nombre
}

// Si varios parámetros seguidos tienen el mismo tipo, se escribe una sola vez
func sumar(a, b int) int {
	return a + b
}

// Varios valores de retorno. El patrón más común en Go es (resultado, error):
// si todo sale bien, el error es nil (tema 12).
func dividir(a, b float64) (float64, error) {
	if b == 0 {
		return 0, errors.New("no se puede dividir por cero")
	}
	return a / b, nil
}

// Retornos con nombre: documentan qué significa cada valor devuelto y se
// inicializan en su valor cero. Un "return" solo (sin valores) devuelve sus
// valores actuales; en funciones largas eso dificulta la lectura, así que
// úsalo con moderación.
func dividirConResto(a, b int) (cociente int, resto int) {
	cociente = a / b // si b fuera 0, esto provocaría un panic (tema 13)
	resto = a % b
	return
}

// Función variádica: "numeros ...int" recibe cero o más enteros.
// Dentro de la función, numeros es un []int.
func sumarTodos(numeros ...int) int {
	total := 0
	for _, n := range numeros {
		total += n
	}
	return total
}

// Función de orden superior: recibe otra función como parámetro.
// El tipo "func(int, int) int" describe cualquier función con esa firma.
func aplicarOperacion(a, b int, operacion func(int, int) int) int {
	return operacion(a, b)
}

func main() {
	fmt.Println(saludar("Ana"))
	fmt.Println("Suma:", sumar(3, 4))

	resultado, err := dividir(10, 2)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("\nDivisión:", resultado)
	}

	// "_" descarta el valor que no interesa
	_, err = dividir(10, 0)
	if err != nil {
		fmt.Println("Error al dividir por cero:", err)
	}

	cociente, resto := dividirConResto(17, 5)
	fmt.Println("\n17 / 5 -> cociente:", cociente, "resto:", resto)

	fmt.Println("\nSuma variádica:", sumarTodos(1, 2, 3, 4, 5))
	fmt.Println("Suma variádica sin argumentos:", sumarTodos())
	// Un slice existente se "expande" como argumentos con "..."
	valores := []int{10, 20, 30}
	fmt.Println("Suma variádica de un slice:", sumarTodos(valores...))

	// Función anónima (sin nombre) guardada en una variable
	multiplicar := func(a, b int) int {
		return a * b
	}
	fmt.Println("\nMultiplicación con función anónima:", multiplicar(3, 4))

	// Pasar funciones como argumento
	fmt.Println("Operación aplicada (suma):", aplicarOperacion(5, 6, sumar))
	fmt.Println("Operación aplicada (multiplicación):", aplicarOperacion(5, 6, multiplicar))

	// Closure: una función que "recuerda" las variables del lugar donde se
	// creó. Cada contador tiene su propia variable "cuenta".
	contadorA := crearContador()
	contadorB := crearContador()
	fmt.Println("\nContador A:", contadorA())
	fmt.Println("Contador A:", contadorA())
	fmt.Println("Contador A:", contadorA())
	fmt.Println("Contador B:", contadorB(), "(independiente de A)")
}

// crearContador devuelve un closure que incrementa una variable interna.
// "cuenta" sigue existiendo mientras exista la función devuelta.
func crearContador() func() int {
	cuenta := 0
	return func() int {
		cuenta++
		return cuenta
	}
}
