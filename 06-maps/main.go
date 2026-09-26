// Tema 6: Maps (diccionarios clave-valor)
//
// Un map asocia claves con valores y permite buscar por clave de forma muy
// rápida. La clave debe ser de un tipo comparable con == (string, int,
// structs simples...); el valor puede ser de cualquier tipo.
//
// Qué aprenderás:
//   - Crear, leer, actualizar y borrar entradas.
//   - Distinguir "la clave no existe" de "el valor es cero" con el modismo ", ok".
//   - Por qué el orden de recorrido no está garantizado, y cómo ordenar.
//
// Ejecutar con: go run ./06-maps
package main

import (
	"fmt"
	"maps"
	"slices"
)

func main() {
	// Map literal: clave string, valor int
	edades := map[string]int{
		"Ana":   30,
		"Luis":  25,
		"María": 40,
	}
	fmt.Println("Map inicial:", edades) // fmt imprime los maps ordenados por clave

	// Map vacío con make, listo para agregar elementos
	precios := make(map[string]float64)
	precios["pan"] = 2.5
	precios["leche"] = 3.2
	fmt.Println("\nMap creado con make:", precios)

	// Cuidado: un map declarado sin inicializar es nil. Se puede leer, pero
	// escribir en él provoca un panic:
	//     var m map[string]int
	//     m["x"] = 1 // panic: assignment to entry in nil map

	// Leer un valor por su clave
	fmt.Println("\nEdad de Ana:", edades["Ana"])

	// Leer una clave inexistente NO es un error: devuelve el valor cero (0)
	fmt.Println("Edad de Pedro (no existe):", edades["Pedro"])

	// Modismo ", ok": el segundo valor indica si la clave existe.
	// Así se distingue "Pedro no está" de "Pedro tiene 0 años".
	if edad, ok := edades["Pedro"]; ok {
		fmt.Println("Pedro tiene", edad, "años")
	} else {
		fmt.Println("Pedro no está en el map")
	}

	// Asignar a una clave existente la actualiza
	edades["Ana"] = 31
	fmt.Println("\nEdad de Ana actualizada:", edades["Ana"])

	// delete elimina una clave (si no existe, no hace nada)
	delete(edades, "Luis")
	fmt.Println("Después de eliminar a Luis:", edades)

	// Recorrer con for-range: el orden es ALEATORIO a propósito y cambia en
	// cada ejecución. Nunca dependas del orden de un map.
	fmt.Println("\nRecorriendo el map (orden no garantizado):")
	for nombre, edad := range edades {
		fmt.Printf("%s tiene %d años\n", nombre, edad)
	}

	// Para un orden estable: obtener las claves, ordenarlas y recorrerlas.
	// maps.Keys devuelve un iterador y slices.Sorted lo convierte en un
	// slice ordenado (Go 1.23+).
	fmt.Println("\nRecorriendo en orden alfabético:")
	for _, nombre := range slices.Sorted(maps.Keys(edades)) {
		fmt.Printf("%s tiene %d años\n", nombre, edades[nombre])
	}

	fmt.Println("\nCantidad de personas en el map:", len(edades))

	// Los maps son referencias: al asignarlos NO se copian. Para una copia
	// independiente se usa maps.Clone.
	alias := edades
	alias["Nuevo"] = 1
	fmt.Println("¿El original ve el cambio hecho en el alias?", edades["Nuevo"] == 1)
	copia := maps.Clone(edades)
	copia["Otro"] = 2
	_, existe := edades["Otro"]
	fmt.Println("¿El original ve el cambio hecho en la copia?", existe)
}
