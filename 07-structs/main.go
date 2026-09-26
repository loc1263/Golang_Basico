// Tema 7: Structs (estructuras)
//
// Un struct agrupa datos relacionados bajo un mismo tipo, como una "ficha"
// con campos. Go no tiene clases: los structs, junto con los métodos (tema 10)
// y las interfaces (tema 11), cumplen ese papel.
//
// Qué aprenderás:
//   - Definir structs y crear valores con y sin nombres de campo.
//   - Structs anidados y embebidos (composición en lugar de herencia).
//   - Que los structs se copian al asignarlos y se pueden comparar con ==.
//
// Ejecutar con: go run ./07-structs
package main

import "fmt"

// Definición de un struct. Los campos que empiezan con MAYÚSCULA son
// visibles desde otros paquetes (tema 21).
type Persona struct {
	Nombre string
	Edad   int
	Correo string
}

type Direccion struct {
	Calle  string
	Ciudad string
}

type Empleado struct {
	Persona   // campo EMBEBIDO: sin nombre, solo el tipo
	Puesto    string
	Direccion Direccion // campo normal cuyo tipo es otro struct (anidado)
}

func main() {
	// Crear un struct indicando el nombre de cada campo (forma recomendada:
	// es más legible y no se rompe si luego se agregan campos al struct)
	p1 := Persona{Nombre: "Ana", Edad: 30, Correo: "ana@correo.com"}
	fmt.Println("Persona 1:", p1)

	// Sin nombres de campo: hay que dar TODOS los valores en el orden exacto.
	// Es frágil; úsalo solo con structs muy pequeños y estables.
	p2 := Persona{"Luis", 25, "luis@correo.com"}
	fmt.Println("Persona 2:", p2)

	// %+v muestra también el nombre de cada campo: muy útil para depurar
	fmt.Printf("Persona 2 con %%+v: %+v\n", p2)

	// Acceder y modificar campos con "."
	p1.Edad = 31
	fmt.Println("\nEdad actualizada de p1:", p1.Edad)

	// Un struct sin inicializar tiene cada campo en su valor cero
	var p3 Persona
	fmt.Printf("\nStruct vacío: %+v\n", p3)

	// Struct con un campo embebido y otro anidado
	emp := Empleado{
		Persona:   Persona{Nombre: "María", Edad: 28, Correo: "maria@correo.com"},
		Puesto:    "Desarrolladora",
		Direccion: Direccion{Calle: "Calle 10", Ciudad: "Bogotá"},
	}
	// Los campos del struct embebido se "promueven": se accede a ellos como
	// si fueran del propio Empleado (emp.Nombre en vez de emp.Persona.Nombre).
	// El struct anidado, en cambio, requiere el camino completo.
	fmt.Println("\nEmpleado:", emp.Nombre, "-", emp.Puesto, "-", emp.Direccion.Ciudad)

	// Go 1.27+: los campos promovidos también se pueden usar al crear el
	// struct, sin escribir el struct embebido explícitamente.
	emp2 := Empleado{Nombre: "Pedro", Edad: 35, Puesto: "Analista"}
	fmt.Printf("Empleado 2: %+v\n", emp2)

	// Struct anónimo: un tipo de un solo uso, sin nombre. Común en tests
	// y para datos temporales.
	punto := struct{ X, Y int }{X: 1, Y: 2}
	fmt.Println("\nStruct anónimo:", punto)

	// Los structs son VALORES: al asignarlos se copian todos sus campos
	copia := p1
	copia.Nombre = "Otro nombre"
	fmt.Println("\nEl original no cambia al modificar la copia:")
	fmt.Println("Original:", p1.Nombre)
	fmt.Println("Copia:", copia.Nombre)

	// Se pueden comparar con == si todos sus campos son comparables
	// (no lo son, por ejemplo, los slices y los maps)
	a := Persona{Nombre: "Ana", Edad: 31, Correo: "ana@correo.com"}
	fmt.Println("\n¿p1 y a son iguales?", p1 == a)
}
