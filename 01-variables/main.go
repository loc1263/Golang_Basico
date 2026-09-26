// Tema 1: Variables y tipos básicos
//
// Qué aprenderás:
//   - Las distintas formas de declarar variables (var, :=) y constantes.
//   - El "valor cero": el valor que Go asigna a una variable sin inicializar.
//   - Los tipos numéricos más comunes y cómo convertir entre ellos.
//
// Ejecutar con: go run ./01-variables
package main

import "fmt"

func main() {
	// Declaración explícita con "var": nombre, tipo y valor inicial.
	var nombre string = "Ana"
	var edad int = 30

	// Declaración corta ":=": Go deduce (infiere) el tipo a partir del valor.
	// Solo puede usarse DENTRO de funciones; fuera de ellas se usa "var".
	pais := "Colombia"
	altura := 1.65 // un literal con decimales es float64 por defecto

	// Declaración múltiple en una sola línea
	var a, b int = 1, 2

	// Constante: su valor se fija al compilar y no puede cambiar después.
	const pi = 3.14159

	// Valor cero: toda variable declarada sin valor recibe uno por defecto.
	// No existen variables "sin inicializar" en Go.
	var activo bool  // false
	var contador int // 0
	var texto string // "" (string vacío)
	var lista []int  // nil (un slice vacío; lo veremos en el tema 5)

	fmt.Println("Nombre:", nombre)
	fmt.Println("Edad:", edad)
	fmt.Println("País:", pais)
	fmt.Println("Altura:", altura)
	fmt.Println("a + b =", a+b)
	fmt.Println("Pi:", pi)

	fmt.Println("\nValores cero por defecto:")
	fmt.Println("bool:", activo)
	fmt.Println("int:", contador)
	fmt.Printf("string: %q\n", texto) // %q muestra el texto entre comillas
	fmt.Println("slice nil:", lista, "¿es nil?", lista == nil)

	// %T imprime el TIPO de una variable: útil para ver qué infirió Go
	fmt.Printf("\nTipos inferidos: pais es %T, altura es %T, edad es %T\n", pais, altura, edad)

	// Tipos numéricos con tamaño explícito. "int" ocupa 64 bits en
	// computadoras de 64 bits; los demás tienen un rango fijo.
	var entero8 int8 = 127                   // rango: -128 a 127
	var entero64 int64 = 9223372036854775807 // el máximo de int64
	var sinSigno uint8 = 255                 // uint = sin signo; uint8 es lo mismo que byte
	var flotante32 float32 = 3.14
	fmt.Println("\nTipos numéricos:")
	fmt.Println("int8:", entero8)
	fmt.Println("int64:", entero64)
	fmt.Println("uint8:", sinSigno)
	fmt.Println("float32:", flotante32)

	// Desbordamiento: al superar el máximo, el valor "da la vuelta".
	// Go no avisa en tiempo de ejecución, así que elige un tipo con rango suficiente.
	entero8++
	fmt.Println("int8 127 + 1 =", entero8)

	// Conversión de tipos: Go NUNCA convierte automáticamente entre tipos
	// distintos (ni siquiera de int a float64). Hay que hacerlo explícitamente.
	var enteroParaFloat int = 10
	var comoFloat float64 = float64(enteroParaFloat)
	fmt.Println("\nConversión int -> float64:", comoFloat)

	// Al convertir de float a int, la parte decimal se descarta (no redondea).
	// Con una constante, int(2.99) ni siquiera compila: el compilador no
	// permite perder información de una constante. Con una variable sí:
	decimal := 2.99
	fmt.Println("Conversión float64 -> int de 2.99:", int(decimal))
}
