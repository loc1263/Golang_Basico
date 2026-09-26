// Tema 10: Métodos (funciones asociadas a un tipo)
//
// Un método es una función con un "receptor": el valor sobre el que se
// llama (r.Area()). Se pueden definir métodos en cualquier tipo propio
// del paquete, no solo en structs.
//
// Qué aprenderás:
//   - Receptor por valor vs. receptor por puntero.
//   - Cuándo usar cada uno.
//   - Métodos sobre tipos que no son structs.
//
// Ejecutar con: go run ./10-metodos
package main

import "fmt"

type Rectangulo struct {
	Ancho, Alto float64
}

// Receptor por VALOR: el método recibe una copia de r, así que no puede
// modificar el Rectangulo original. Adecuado para métodos que solo leen.
func (r Rectangulo) Area() float64 {
	return r.Ancho * r.Alto
}

func (r Rectangulo) Perimetro() float64 {
	return 2 * (r.Ancho + r.Alto)
}

// Receptor por PUNTERO: el método recibe la dirección del original y
// puede modificarlo.
func (r *Rectangulo) Escalar(factor float64) {
	r.Ancho *= factor
	r.Alto *= factor
}

// ContadorClicks usa receptor puntero en TODOS sus métodos. La guía de
// estilo de Go recomienda no mezclar: si algún método necesita puntero,
// que todos lo usen. (En Rectangulo se mezclan solo para comparar ambos tipos.)
type ContadorClicks struct {
	total int
}

func (c *ContadorClicks) Click() {
	c.total++
}

func (c *ContadorClicks) Total() int {
	return c.total
}

// Se pueden definir métodos sobre cualquier tipo propio, no solo structs
type Celsius float64

func (c Celsius) AFahrenheit() float64 {
	return float64(c)*9/5 + 32
}

func main() {
	r := Rectangulo{Ancho: 4, Alto: 3}
	fmt.Println("Área:", r.Area())
	fmt.Println("Perímetro:", r.Perimetro())

	// Escalar tiene receptor puntero, pero se puede llamar sobre la variable
	// "r": Go toma su dirección automáticamente, como si fuera (&r).Escalar(2).
	r.Escalar(2)
	fmt.Println("\nDespués de escalar x2:")
	fmt.Println("Ancho:", r.Ancho, "Alto:", r.Alto)
	fmt.Println("Nueva área:", r.Area())

	clicks := ContadorClicks{}
	clicks.Click()
	clicks.Click()
	clicks.Click()
	fmt.Println("\nTotal de clicks:", clicks.Total())

	temperatura := Celsius(25)
	fmt.Printf("\n%.1f °C = %.1f °F\n", temperatura, temperatura.AFahrenheit())

	// Regla práctica para elegir el receptor:
	//   - Puntero: si el método modifica el valor, si el struct es grande
	//     (evita copiarlo en cada llamada) o si contiene un sync.Mutex.
	//   - Valor: si el método solo lee y el tipo es pequeño (como Celsius).
	//   - Ante la duda, usa puntero, y sé consistente dentro del mismo tipo.
}
