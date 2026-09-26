// Tema 18: Testing
//
// Go incluye un framework de pruebas en la librería estándar (paquete
// "testing") y el comando "go test". No hace falta instalar nada.
// Convenciones:
//   - Los tests van en archivos que terminan en "_test.go" (main_test.go).
//   - Cada test es una función TestXxx(t *testing.T).
//   - go build ignora los archivos _test.go; solo los usa go test.
//
// Ejecutar el programa:            go run ./18-testing
// Ejecutar las pruebas:            go test ./18-testing
// Con detalle de cada caso:        go test -v ./18-testing
// Solo los tests que coinciden:    go test -run TestEsPar -v ./18-testing
// Con porcentaje de cobertura:     go test -cover ./18-testing
package main

import (
	"errors"
	"fmt"
)

// Funciones simples que se prueban en main_test.go

func Sumar(a, b int) int {
	return a + b
}

func EsPar(n int) bool {
	return n%2 == 0
}

var ErrDivisionPorCero = errors.New("no se puede dividir por cero")

func Dividir(a, b float64) (float64, error) {
	if b == 0 {
		return 0, ErrDivisionPorCero
	}
	return a / b, nil
}

func main() {
	fmt.Println("Sumar(2, 3):", Sumar(2, 3))
	fmt.Println("EsPar(4):", EsPar(4))
	resultado, err := Dividir(10, 2)
	fmt.Println("Dividir(10, 2):", resultado, err)

	fmt.Println("\nPara ejecutar las pruebas de este tema usa:")
	fmt.Println("  go test ./18-testing")
	fmt.Println("  go test -v ./18-testing   (con detalle de cada caso)")
}
