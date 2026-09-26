// Tema 25: Expresiones regulares (paquete regexp)
//
// Una expresión regular (regex) es un patrón para buscar, validar o
// reemplazar texto. Go usa la sintaxis RE2: garantiza tiempo lineal (una
// regex nunca "se cuelga" con entradas maliciosas), a cambio de no admitir
// algunas funciones como las referencias hacia atrás (\1).
//
// Qué aprenderás:
//   - Compilar un patrón y buscar coincidencias.
//   - Validar formatos, reemplazar y dividir texto.
//   - Extraer partes con grupos de captura (numerados y con nombre).
//
// Ejecutar con: go run ./25-regexp
package main

import (
	"fmt"
	"regexp"
)

// Compilar una regex tiene un costo: lo habitual es hacerlo UNA vez, en una
// variable de paquete, y no dentro de una función que se llama muchas veces.
// MustCompile provoca un panic si el patrón es inválido; es apropiado para
// patrones fijos escritos en el código. Para patrones que vienen del usuario
// usa regexp.Compile, que devuelve un error.
var patronNumeros = regexp.MustCompile(`\d+`)

func main() {
	// Las comillas invertidas `...` crean un "raw string": la barra \ no
	// necesita escaparse, lo que hace los patrones mucho más legibles
	texto := "Tengo 3 gatos y 12 peces, en total 15 mascotas"

	fmt.Println("¿Contiene números?", patronNumeros.MatchString(texto))
	fmt.Println("Primer número encontrado:", patronNumeros.FindString(texto))
	// El segundo argumento limita la cantidad de resultados; -1 significa "todos"
	fmt.Println("Todos los números:", patronNumeros.FindAllString(texto, -1))

	// Validar un formato: ^ y $ anclan el patrón al inicio y al final, para
	// que TODO el texto deba coincidir (no solo una parte).
	// Este patrón de correo es simplificado: validar correos al 100% con una
	// regex es impráctico; para eso existe net/mail (tema 39).
	patronCorreo := regexp.MustCompile(`^[\w.-]+@[\w.-]+\.\w+$`)
	correos := []string{"ana@correo.com", "no-es-correo", "luis.perez@empresa.co"}
	fmt.Println("\nValidando correos:")
	for _, correo := range correos {
		fmt.Printf("%-25s -> válido: %v\n", correo, patronCorreo.MatchString(correo))
	}

	// ReplaceAllString: reemplazar todas las coincidencias
	patronEspacios := regexp.MustCompile(`\s+`)
	textoConEspacios := "Hola     Mundo   desde      Go"
	normalizado := patronEspacios.ReplaceAllString(textoConEspacios, " ")
	fmt.Printf("\nTexto original:    %q\n", textoConEspacios)
	fmt.Printf("Texto normalizado: %q\n", normalizado)

	// Grupos de captura (entre paréntesis): extraen partes de la coincidencia.
	// FindStringSubmatch devuelve [coincidencia completa, grupo 1, grupo 2, ...]
	// o nil si no hubo coincidencia.
	patronFecha := regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})`)
	fechaTexto := "La reunión es el 2025-12-25 por la mañana"
	if grupos := patronFecha.FindStringSubmatch(fechaTexto); grupos != nil {
		fmt.Println("\nFecha completa:", grupos[0])
		fmt.Println("Año:", grupos[1], "- Mes:", grupos[2], "- Día:", grupos[3])
	}

	// Grupos con nombre (?P<nombre>...): más legibles que los índices.
	// SubexpIndex devuelve la posición de un grupo a partir de su nombre.
	patronConNombre := regexp.MustCompile(`(?P<anio>\d{4})-(?P<mes>\d{2})-(?P<dia>\d{2})`)
	if m := patronConNombre.FindStringSubmatch(fechaTexto); m != nil {
		fmt.Println("\nGrupos con nombre:")
		fmt.Println("anio:", m[patronConNombre.SubexpIndex("anio")])
		fmt.Println("mes: ", m[patronConNombre.SubexpIndex("mes")])
		fmt.Println("dia: ", m[patronConNombre.SubexpIndex("dia")])
	}

	// Reemplazo usando los grupos: $1, $2... o ${nombre}
	fmt.Println("\nFecha reordenada:", patronConNombre.ReplaceAllString(fechaTexto, "${dia}/${mes}/${anio}"))

	// Split: dividir usando el patrón como separador
	patronComa := regexp.MustCompile(`\s*,\s*`)
	lista := patronComa.Split("manzana, banana,cereza ,  durazno", -1)
	fmt.Printf("\nLista dividida con regexp: %q\n", lista)

	// Consejo: para búsquedas simples (¿contiene "abc"?) usa el paquete
	// strings: es más rápido y más claro que una regex.
}
