// Tema 15: Strings, Runes y Bytes
//
// En Go un string es una secuencia INMUTABLE de bytes, normalmente texto
// codificado en UTF-8. Un carácter como "é" o "ñ" ocupa más de un byte.
//   - byte (alias de uint8): un byte.
//   - rune (alias de int32): un carácter Unicode ("punto de código").
//
// Qué aprenderás:
//   - Por qué len() cuenta bytes y cómo contar caracteres.
//   - Convertir entre string, []byte y []rune.
//   - Las funciones más usadas de los paquetes strings y strconv.
//
// Ejecutar con: go run ./15-strings-runes-bytes
package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

func main() {
	texto := "café"

	// len() de un string cuenta BYTES, no caracteres
	fmt.Println("Texto:", texto)
	fmt.Println("len(texto) en bytes:", len(texto))
	fmt.Println("Cantidad real de caracteres (runas):", utf8.RuneCountInString(texto))

	// Indexar un string devuelve un BYTE, no un carácter
	fmt.Printf("texto[3] es el byte %d, no la letra 'é'\n", texto[3])

	// Los strings son inmutables: texto[0] = 'C' no compila.
	// Para modificarlos se convierten a []rune o []byte y se crea uno nuevo.
	runas := []rune(texto)
	runas[0] = 'C'
	fmt.Println("\nTexto modificado vía []rune:", string(runas))

	bytesTexto := []byte("abc")
	bytesTexto[0] = 'x'
	fmt.Println("Texto modificado vía []byte:", string(bytesTexto))

	// Paquete strings: funciones comunes. Ninguna modifica el original:
	// todas devuelven un string nuevo.
	frase := "  Hola Mundo desde Go  "
	fmt.Printf("\nOriginal:  %q\n", frase)
	fmt.Printf("TrimSpace: %q\n", strings.TrimSpace(frase))
	fmt.Println("ToUpper:", strings.ToUpper(frase))
	fmt.Println("ToLower:", strings.ToLower(frase))
	fmt.Println("Contains 'Mundo':", strings.Contains(frase, "Mundo"))
	fmt.Println("Index de 'Mundo':", strings.Index(frase, "Mundo"))
	fmt.Println("ReplaceAll Mundo->Go:", strings.ReplaceAll(frase, "Mundo", "Go"))
	fmt.Printf("Split por \",\": %q\n", strings.Split("a,b,,c", ","))
	fmt.Printf("Fields (separa por espacios, ignora repetidos): %q\n", strings.Fields(frase))
	fmt.Println("Join con guiones:", strings.Join([]string{"uno", "dos", "tres"}, "-"))
	fmt.Println("HasPrefix 'Hola':", strings.HasPrefix(strings.TrimSpace(frase), "Hola"))
	fmt.Println("Repeat:", strings.Repeat("ab", 3))

	// strings.Cut divide en dos partes alrededor del primer separador
	clave, valor, encontrado := strings.Cut("usuario=ana", "=")
	fmt.Printf("Cut: clave=%q valor=%q encontrado=%v\n", clave, valor, encontrado)

	// strings.Builder: la forma eficiente de construir un string en partes.
	// Concatenar con + dentro de un ciclo crea un string nuevo en cada vuelta.
	var sb strings.Builder
	for range 3 {
		sb.WriteString("Go! ")
	}
	fmt.Println("\nConstruido con strings.Builder:", sb.String())

	// strconv: convertir entre strings y números o booleanos
	numero, err := strconv.Atoi("42")
	if err != nil {
		fmt.Println("Error al convertir:", err)
	} else {
		fmt.Println("\nString a int:", numero)
	}
	fmt.Println("Int a string:", strconv.Itoa(100))

	flotante, err := strconv.ParseFloat("3.14", 64)
	fmt.Println("String a float64:", flotante, "error:", err)

	// Una conversión inválida devuelve un error descriptivo
	_, err = strconv.Atoi("doce")
	fmt.Println("Atoi(\"doce\"):", err)

	// Cuidado: string(65) NO produce "65", sino el carácter con código 65 ("A").
	// Para convertir un número a texto se usa strconv.Itoa o fmt.Sprint.
	fmt.Println("string(rune(65)) =", string(rune(65)), "| strconv.Itoa(65) =", strconv.Itoa(65))

	// Recorrer un string con for-range entrega runas (caracteres completos),
	// junto con la posición en bytes donde empieza cada una
	fmt.Println("\nRecorriendo runas de:", texto)
	for i, r := range texto {
		fmt.Printf("byte inicial %d: rune %c (código %d, ocupa %d bytes)\n", i, r, r, utf8.RuneLen(r))
	}
}
