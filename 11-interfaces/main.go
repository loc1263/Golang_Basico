// Tema 11: Interfaces
//
// Una interfaz describe un COMPORTAMIENTO: un conjunto de métodos. Un tipo
// cumple una interfaz simplemente por tener esos métodos; no hace falta
// declararlo con "implements" (las interfaces se satisfacen implícitamente).
//
// Qué aprenderás:
//   - Definir interfaces y escribir funciones que aceptan cualquier tipo
//     que las cumpla (polimorfismo).
//   - Recuperar el tipo concreto con type assertion y type switch.
//   - La interfaz vacía (any).
//
// Ejecutar con: go run ./11-interfaces
package main

import (
	"fmt"
	"math"
)

// Figura: cualquier tipo que tenga AMBOS métodos, Area y Perimetro,
// cumple esta interfaz.
type Figura interface {
	Area() float64
	Perimetro() float64
}

type Rectangulo struct {
	Ancho, Alto float64
}

func (r Rectangulo) Area() float64 {
	return r.Ancho * r.Alto
}

func (r Rectangulo) Perimetro() float64 {
	return 2 * (r.Ancho + r.Alto)
}

type Circulo struct {
	Radio float64
}

func (c Circulo) Area() float64 {
	return math.Pi * c.Radio * c.Radio
}

func (c Circulo) Perimetro() float64 {
	return 2 * math.Pi * c.Radio
}

// Verificación en tiempo de compilación: si a Circulo le faltara un
// método, esta línea no compilaría. No genera código ni ocupa memoria.
var _ Figura = Circulo{}

// describir acepta CUALQUIER tipo que cumpla la interfaz Figura
func describir(f Figura) {
	fmt.Printf("Área: %.2f, Perímetro: %.2f\n", f.Area(), f.Perimetro())
}

func main() {
	r := Rectangulo{Ancho: 4, Alto: 3}
	c := Circulo{Radio: 5}

	describir(r)
	describir(c)

	// Un slice de interfaz puede contener distintos tipos concretos
	figuras := []Figura{r, c, Rectangulo{Ancho: 2, Alto: 2}}
	fmt.Println("\nDescribiendo varias figuras:")
	for _, f := range figuras {
		describir(f)
	}

	// Type assertion: recuperar el tipo concreto guardado en la interfaz.
	// Con la forma ", ok" no hay riesgo: si el tipo no coincide, ok es false.
	// Sin ", ok" (rect := f.(Rectangulo)) un tipo incorrecto provoca un panic.
	var f Figura = r
	if rectangulo, ok := f.(Rectangulo); ok {
		fmt.Println("\nEs un rectángulo con ancho:", rectangulo.Ancho)
	}
	if _, ok := f.(Circulo); !ok {
		fmt.Println("No es un círculo")
	}

	// Type switch: elegir el comportamiento según el tipo concreto
	fmt.Println("\nType switch sobre cada figura:")
	for _, fig := range figuras {
		switch v := fig.(type) {
		case Rectangulo:
			fmt.Println("Es un Rectangulo con ancho", v.Ancho)
		case Circulo:
			fmt.Println("Es un Circulo con radio", v.Radio)
		default:
			fmt.Println("Tipo desconocido")
		}
	}

	// La interfaz vacía no exige ningún método, así que acepta cualquier
	// valor. Se escribe "any" (alias de interface{}). Úsala con moderación:
	// se pierde la verificación de tipos del compilador.
	var cualquiera any = 42
	fmt.Println("\nInterfaz vacía guardando un int:", cualquiera)
	cualquiera = "ahora un string"
	fmt.Println("Interfaz vacía guardando un string:", cualquiera)

	// Consejo de diseño: en Go se prefieren interfaces PEQUEÑAS (de uno o dos
	// métodos), como io.Reader o fmt.Stringer (tema 27).
}
