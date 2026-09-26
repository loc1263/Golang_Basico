// Tema 21: Paquetes propios y visibilidad (exportado / no exportado)
//
// Todo archivo .go pertenece a un paquete, y todos los archivos de una misma
// carpeta forman un solo paquete. Un identificador (función, tipo, variable,
// constante, campo) es:
//   - EXPORTADO, visible desde otros paquetes, si empieza con MAYÚSCULA.
//   - No exportado, privado del paquete, si empieza con minúscula.
//
// Estructura de este ejemplo:
//
//	21-paquetes-propios/
//	  main.go           paquete main: el programa que se ejecuta
//	  matematica/
//	    matematica.go   paquete matematica: código reutilizable
//
// Ejecutar con: go run ./21-paquetes-propios
package main

import (
	"fmt"

	// Se importa con: nombre del módulo (go.mod) + ruta de la carpeta.
	// Luego se usa con el nombre del paquete: matematica.Sumar(...)
	"golang-basico/21-paquetes-propios/matematica"
)

func main() {
	fmt.Println("Suma:", matematica.Sumar(4, 5))
	fmt.Println("Resta:", matematica.Restar(10, 3))
	fmt.Println("Pi:", matematica.Pi)
	fmt.Println("Área de círculo con radio 3:", matematica.AreaCirculo(3))

	// matematica.redondear(1.5) // no compila si se descomenta:
	// "redondear" empieza con minúscula, así que no es visible fuera de su paquete.

	// Consejos:
	//   - El nombre del paquete es corto, en minúscula y sin guiones bajos.
	//   - Evita nombres genéricos como "util" o "comun": el nombre debe decir
	//     qué ofrece el paquete.
	//   - Una carpeta llamada "internal" solo puede importarse desde su carpeta
	//     padre y descendientes (el tema 50 lo usa).
}
