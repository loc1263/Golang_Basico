// Tema 2: Condicionales (if/else, switch)
//
// Qué aprenderás:
//   - if / else if / else, y el "if con inicialización".
//   - switch con valor, switch sin expresión y casos con varios valores.
//   - fallthrough: la excepción a la regla de que cada caso termina solo.
//
// Ejecutar con: go run ./02-condicionales
package main

import "fmt"

func main() {
	edad := 20

	// if / else if / else. En Go la condición NO lleva paréntesis,
	// pero las llaves { } son obligatorias aunque haya una sola línea.
	if edad < 13 {
		fmt.Println("Eres un niño")
	} else if edad < 18 {
		fmt.Println("Eres un adolescente")
	} else {
		fmt.Println("Eres un adulto")
	}

	// if con inicialización: la variable "nota" solo existe dentro del if/else.
	// Es muy común con funciones que devuelven un error (tema 12):
	//     if err := hacerAlgo(); err != nil { ... }
	if nota := 8.5; nota >= 6 {
		fmt.Println("Aprobado con nota", nota)
	} else {
		fmt.Println("Reprobado con nota", nota)
	}

	// switch con un valor: compara "dia" contra cada caso.
	// No hace falta "break": cada caso termina automáticamente.
	dia := 3
	switch dia {
	case 1:
		fmt.Println("Lunes")
	case 2:
		fmt.Println("Martes")
	case 3:
		fmt.Println("Miércoles")
	default: // se ejecuta si ningún caso coincide
		fmt.Println("Otro día")
	}

	// switch sin expresión: equivale a una cadena de if/else if más legible.
	// Se evalúan los casos en orden y se ejecuta el primero que sea verdadero.
	temperatura := 28
	switch {
	case temperatura < 10:
		fmt.Println("Hace frío")
	case temperatura < 25:
		fmt.Println("Clima templado")
	default:
		fmt.Println("Hace calor")
	}

	// Varios valores en un mismo caso, separados por comas
	letra := "a"
	switch letra {
	case "a", "e", "i", "o", "u":
		fmt.Println("Es una vocal")
	default:
		fmt.Println("Es una consonante")
	}

	// fallthrough: fuerza a continuar con el SIGUIENTE caso sin evaluarlo.
	// Se usa poco; aquí sirve para mostrar niveles acumulativos.
	fmt.Println("\nPermisos del nivel 2:")
	nivel := 2
	switch nivel {
	case 2:
		fmt.Println("- puede editar")
		fallthrough
	case 1:
		fmt.Println("- puede leer")
	}
}
