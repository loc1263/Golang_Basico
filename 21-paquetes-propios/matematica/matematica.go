// Package matematica muestra cómo organizar código reutilizable en un
// paquete propio. Por convención, el nombre del paquete coincide con el de
// su carpeta.
//
// Este comentario, justo antes de "package", es la documentación del
// paquete: la muestran "go doc" y pkg.go.dev. Lo mismo ocurre con los
// comentarios que preceden a cada identificador exportado, que por
// convención empiezan con su nombre.
package matematica

import "math"

// Pi es una constante exportada (empieza con mayúscula).
const Pi = 3.14159

// Sumar devuelve a + b. Es exportada: visible desde otros paquetes.
func Sumar(a, b int) int {
	return a + b
}

// Restar devuelve a - b.
func Restar(a, b int) int {
	return a - b
}

// AreaCirculo devuelve el área de un círculo, redondeada a dos decimales.
// Usa internamente la función no exportada redondear.
func AreaCirculo(radio float64) float64 {
	return redondear(Pi * radio * radio)
}

// redondear NO es exportada (empieza con minúscula): solo se puede usar
// dentro del paquete matematica.
func redondear(valor float64) float64 {
	// math.Round redondea al entero más cercano; multiplicar y dividir por
	// 100 lo aplica a dos decimales.
	return math.Round(valor*100) / 100
}
