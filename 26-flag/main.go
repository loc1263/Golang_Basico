// Tema 26: flag (argumentos de línea de comandos)
//
// El paquete flag interpreta opciones como -nombre=Ana y genera una ayuda
// automática con -h.
//
// Qué aprenderás:
//   - Definir flags de distintos tipos y leerlos después de flag.Parse().
//   - Diferenciar flags de argumentos posicionales.
//   - La sintaxis que acepta el paquete flag.
//
// Ejecutar con valores por defecto: go run ./26-flag
// Ejecutar con argumentos:          go run ./26-flag -nombre=Luis -edad=25 -activo=false -espera=2s
// Ver la ayuda generada:            go run ./26-flag -h
package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	// flag.String/Int/Bool/Duration reciben: nombre, valor por defecto y
	// descripción (la que muestra -h). Devuelven un PUNTERO al valor.
	nombre := flag.String("nombre", "Invitado", "nombre de la persona a saludar")
	edad := flag.Int("edad", 18, "edad de la persona")
	activo := flag.Bool("activo", true, "si la persona está activa")
	espera := flag.Duration("espera", 0, "pausa antes de terminar (por ejemplo 500ms o 2s)")

	// Alternativa: flag.StringVar escribe en una variable existente
	var saludo string
	flag.StringVar(&saludo, "saludo", "Hola", "palabra de saludo")

	// flag.Parse lee os.Args. Debe llamarse después de definir todos los
	// flags y antes de leerlos.
	flag.Parse()

	fmt.Println("Nombre:", *nombre) // "*" para obtener el valor apuntado
	fmt.Println("Edad:", *edad)
	fmt.Println("Activo:", *activo)
	fmt.Println("Espera:", *espera)

	// Sintaxis aceptada: -edad=25, -edad 25, --edad=25.
	// Excepción: los booleanos requieren "=" para indicar false (-activo=false),
	// porque "-activo" solo ya significa true.

	// os.Args tiene los argumentos crudos; os.Args[0] es la ruta del programa
	fmt.Println("\nArgumentos crudos (os.Args):", os.Args[1:])

	// flag.Args() devuelve los argumentos posicionales: los que quedan
	// después del último flag. El análisis de flags termina en el primer
	// argumento que no empieza con "-" (o en "--").
	restantes := flag.Args()
	if len(restantes) > 0 {
		fmt.Println("Argumentos posicionales:", restantes)
	} else {
		fmt.Println("No se pasaron argumentos posicionales.")
		fmt.Println("Prueba: go run ./26-flag -nombre=Ana extra1 extra2")
	}

	estado := "no está activo/a"
	if *activo {
		estado = "está activo/a"
	}
	fmt.Printf("\n%s, %s (%d años) %s.\n", saludo, *nombre, *edad, estado)

	if *espera > 0 {
		time.Sleep(*espera)
		fmt.Println("Terminé de esperar", *espera)
	}
}
