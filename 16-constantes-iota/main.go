// Tema 16: Constantes e iota
//
// Qué aprenderás:
//   - Constantes con y sin tipo, y por qué importa la diferencia.
//   - iota para generar secuencias de constantes (enumeraciones).
//   - Flags de bits y el patrón "saltar valores" con "_".
//
// Ejecutar con: go run ./16-constantes-iota
package main

import "fmt"

// Constante SIN tipo (untyped): no tiene un tipo fijo hasta que se usa, así
// que se adapta al contexto. Por ejemplo, 3.0 puede usarse como float32 o float64.
const Version = "1.0"

// Constantes CON tipo: solo pueden usarse donde se espera ese tipo exacto
const (
	MinEdad int = 0
	MaxEdad int = 130
)

// iota vale 0 en la primera línea de un bloque const y aumenta en 1 por
// cada línea. Si una línea no tiene expresión, repite la anterior con el
// nuevo iota. Así se crean enumeraciones.
type DiaSemana int

const (
	Lunes     DiaSemana = iota // 0
	Martes                     // 1
	Miercoles                  // 2
	Jueves                     // 3
	Viernes                    // 4
	Sabado                     // 5
	Domingo                    // 6
)

// String() hace que DiaSemana cumpla la interfaz fmt.Stringer (tema 27):
// fmt la usa automáticamente al imprimir con %v o %s.
// El tema 48 muestra cómo generar este método con la herramienta stringer.
func (d DiaSemana) String() string {
	nombres := [...]string{"Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado", "Domingo"}
	if d < 0 || int(d) >= len(nombres) {
		return fmt.Sprintf("DiaSemana(%d)", int(d))
	}
	return nombres[d]
}

// iota con expresiones: potencias de 2 para combinar opciones como bits
type Permiso uint

const (
	PermisoLectura   Permiso = 1 << iota // 1 << 0 = 1 (binario 001)
	PermisoEscritura                     // 1 << 1 = 2 (binario 010)
	PermisoEjecucion                     // 1 << 2 = 4 (binario 100)
)

// "_" descarta un valor de iota: aquí se salta el 0 para empezar en KB
const (
	_  = iota             // 0 (descartado)
	KB = 1 << (10 * iota) // 1 << 10 = 1024
	MB                    // 1 << 20
	GB                    // 1 << 30
)

func main() {
	fmt.Println("Versión:", Version)
	fmt.Println("Rango de edad válido:", MinEdad, "-", MaxEdad)

	fmt.Println("\nDías de la semana (iota):")
	hoy := Miercoles
	fmt.Println("Hoy es:", hoy) // usa String() automáticamente

	for d := Lunes; d <= Domingo; d++ {
		fmt.Printf("%d -> %s\n", d, d) // %d muestra el número, %s el texto
	}
	fmt.Println("Valor fuera de rango:", DiaSemana(9))

	fmt.Println("\nPermisos como flags de bits:")
	misPermisos := PermisoLectura | PermisoEscritura // "|" combina bits
	fmt.Printf("misPermisos = %d (binario %03b)\n", misPermisos, misPermisos)
	fmt.Println("¿Tiene lectura?", misPermisos&PermisoLectura != 0) // "&" consulta un bit
	fmt.Println("¿Tiene escritura?", misPermisos&PermisoEscritura != 0)
	fmt.Println("¿Tiene ejecución?", misPermisos&PermisoEjecucion != 0)

	fmt.Println("\nUnidades de tamaño con iota:")
	fmt.Println("KB:", KB)
	fmt.Println("MB:", MB)
	fmt.Println("GB:", GB)
}
