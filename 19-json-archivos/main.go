// Tema 19: JSON y archivos
//
// Qué aprenderás:
//   - Convertir structs a JSON (Marshal) y JSON a structs (Unmarshal).
//   - Controlar los nombres de los campos con "tags".
//   - Escribir, leer y borrar archivos con el paquete os.
//   - Leer texto línea por línea con bufio.Scanner.
//   - Las diferencias de encoding/json/v2 (Go 1.27+).
//
// Ejecutar con: go run ./19-json-archivos
package main

import (
	"bufio"
	"encoding/json"
	jsonv2 "encoding/json/v2" // alias: ambos paquetes se llaman "json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Los "tags" (texto entre comillas invertidas) indican cómo se llama cada
// campo en el JSON. Solo se serializan los campos EXPORTADOS (con mayúscula).
type Persona struct {
	Nombre string `json:"nombre"`
	Edad   int    `json:"edad"`
	Correo string `json:"correo,omitempty"` // omitempty: se omite si está vacío
	clave  string // no exportado: encoding/json lo ignora
}

func main() {
	p := Persona{Nombre: "Ana", Edad: 30, Correo: "ana@correo.com", clave: "secreta"}

	// Marshal: struct de Go -> JSON (como []byte)
	datosJSON, err := json.Marshal(p)
	if err != nil {
		fmt.Println("Error al convertir a JSON:", err)
		return
	}
	fmt.Println("JSON generado:", string(datosJSON))

	// MarshalIndent: igual, pero con saltos de línea e indentación
	datosIndentados, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("\nJSON indentado:")
	fmt.Println(string(datosIndentados))

	// Unmarshal: JSON -> struct. Se pasa un PUNTERO para que pueda llenarlo.
	jsonTexto := `{"nombre":"Luis","edad":25,"correo":"luis@correo.com","extra":"se ignora"}`
	var persona2 Persona
	if err := json.Unmarshal([]byte(jsonTexto), &persona2); err != nil {
		fmt.Println("Error al leer JSON:", err)
		return
	}
	fmt.Printf("\nStruct desde JSON: %+v\n", persona2)

	// Un JSON mal formado o con tipos incorrectos produce un error
	var invalida Persona
	err = json.Unmarshal([]byte(`{"nombre":"Eva","edad":"treinta"}`), &invalida)
	fmt.Println("JSON con tipo incorrecto:", err)

	// Un slice de structs se convierte en un array JSON
	personas := []Persona{
		{Nombre: "Ana", Edad: 30},
		{Nombre: "Luis", Edad: 25},
	}
	listaJSON, err := json.MarshalIndent(personas, "", "  ")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("\nLista de personas en JSON:")
	fmt.Println(string(listaJSON))

	// JSON de estructura desconocida: se puede decodificar en un map
	var generico map[string]any
	if err := json.Unmarshal([]byte(`{"activo":true,"puntos":10}`), &generico); err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("\nJSON en map[string]any:", generico)
	// Ojo: en un map[string]any los números JSON siempre llegan como float64
	fmt.Printf("Tipo de \"puntos\": %T\n", generico["puntos"])

	// --- encoding/json/v2 (Go 1.27+) ---
	// Una nueva versión del paquete, con valores por defecto más estrictos y
	// predecibles. encoding/json (v1) sigue existiendo y no cambia su
	// comportamiento, así que el código existente no se rompe.
	fmt.Println("\nDiferencias entre encoding/json (v1) y encoding/json/v2:")
	type Ficha struct {
		Nombre    string   `json:"nombre"`
		Etiquetas []string `json:"etiquetas"`
	}
	f := Ficha{Nombre: "Ana"} // Etiquetas queda nil
	v1, _ := json.Marshal(f)
	v2, _ := jsonv2.Marshal(f)
	fmt.Println("slice nil en v1:", string(v1)) // null
	fmt.Println("slice nil en v2:", string(v2)) // [] (lo que suele esperar quien consume la API)

	// v1 ignora mayúsculas/minúsculas al emparejar nombres; v2 no
	var f1, f2 Ficha
	// (ninguna de las dos devuelve error: un nombre desconocido se ignora)
	if err := json.Unmarshal([]byte(`{"NOMBRE":"Luis"}`), &f1); err != nil {
		fmt.Println("Error v1:", err)
	}
	if err := jsonv2.Unmarshal([]byte(`{"NOMBRE":"Luis"}`), &f2); err != nil {
		fmt.Println("Error v2:", err)
	}
	fmt.Printf("\"NOMBRE\" en v1: %q | en v2: %q\n", f1.Nombre, f2.Nombre)

	// v2 rechaza claves duplicadas (v1 se queda en silencio con la última)
	err = jsonv2.Unmarshal([]byte(`{"nombre":"a","nombre":"b"}`), &f2)
	fmt.Println("Clave duplicada en v2:", err)

	// --- Archivos ---
	// os.TempDir() devuelve la carpeta temporal del sistema; así el ejemplo
	// no deja archivos en el proyecto. filepath.Join arma la ruta con el
	// separador correcto para cada sistema operativo (\ en Windows, / en Linux).
	nombreArchivo := filepath.Join(os.TempDir(), "personas.json")

	// os.WriteFile crea el archivo (o lo sobrescribe completo).
	// 0o644 son los permisos Unix: lectura/escritura para el dueño y lectura
	// para los demás. En Windows se ignoran casi por completo.
	if err := os.WriteFile(nombreArchivo, listaJSON, 0o644); err != nil {
		fmt.Println("Error al escribir archivo:", err)
		return
	}
	// defer asegura que el archivo se borre al terminar main (tema 13)
	defer os.Remove(nombreArchivo)
	fmt.Println("\nArchivo escrito:", nombreArchivo)

	// os.ReadFile lee el archivo completo en memoria
	contenido, err := os.ReadFile(nombreArchivo)
	if err != nil {
		fmt.Println("Error al leer archivo:", err)
		return
	}
	fmt.Println("Leídos", len(contenido), "bytes")

	var personasLeidas []Persona
	if err := json.Unmarshal(contenido, &personasLeidas); err != nil {
		fmt.Println("Error al decodificar:", err)
		return
	}
	fmt.Printf("Structs reconstruidos desde el archivo: %+v\n", personasLeidas)

	// Leer un archivo que no existe: el error se puede identificar
	_, err = os.ReadFile("no-existe.txt")
	fmt.Println("\n¿Error por archivo inexistente?", os.IsNotExist(err))

	// bufio.Scanner lee línea por línea sin cargar todo en memoria: ideal
	// para archivos grandes. Acepta cualquier io.Reader (un *os.File, o aquí
	// un strings.Reader para el ejemplo).
	fmt.Println("\nLeyendo línea por línea con bufio.Scanner:")
	lector := bufio.NewScanner(strings.NewReader("línea 1\nlínea 2\nlínea 3"))
	for lector.Scan() {
		fmt.Println("->", lector.Text())
	}
	if err := lector.Err(); err != nil { // errores ocurridos durante la lectura
		fmt.Println("Error leyendo:", err)
	}
}
