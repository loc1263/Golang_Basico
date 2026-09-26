// Tema 27: Interfaces estándar (fmt.Stringer, io.Reader, io.Writer)
//
// La librería estándar define varias interfaces pequeñas que se usan en
// todas partes. Si un tipo propio las implementa, funciona automáticamente
// con cientos de funciones existentes.
//
//	fmt.Stringer:  String() string                       cómo se imprime un valor
//	io.Writer:     Write(p []byte) (n int, err error)    algo donde se escriben bytes
//	io.Reader:     Read(p []byte) (n int, err error)     algo de donde se leen bytes
//
// Archivos, conexiones de red, respuestas HTTP, buffers y compresores son
// todos io.Reader y/o io.Writer, por eso se pueden combinar libremente.
//
// Ejecutar con: go run ./27-interfaces-estandar
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// fmt.Stringer: fmt usa el método String() al imprimir con Println, %v o %s
type Punto struct {
	X, Y int
}

func (p Punto) String() string {
	return fmt.Sprintf("(%d, %d)", p.X, p.Y)
}

// io.Writer propio que convierte a mayúsculas y reenvía a otro Writer.
// Este patrón de "envolver" un Writer (o Reader) es muy común.
type MayusculasWriter struct {
	Destino io.Writer
}

// El contrato de io.Writer exige devolver cuántos bytes de "datos" se
// consumieron (como máximo len(datos)), y un error si fueron menos.
// Por eso devolvemos len(datos) y no la longitud del texto transformado,
// que podría ser distinta.
func (w MayusculasWriter) Write(datos []byte) (int, error) {
	mayusculas := strings.ToUpper(string(datos))
	if _, err := io.WriteString(w.Destino, mayusculas); err != nil {
		return 0, err
	}
	return len(datos), nil
}

func main() {
	// Sin el método String(), fmt imprimiría {3 4}
	p := Punto{X: 3, Y: 4}
	fmt.Println("Punto:", p)
	fmt.Printf("Punto con %%v: %v\n", p)
	fmt.Printf("Punto con %%s: %s\n", p)

	// io.Writer: fmt.Fprintf escribe en CUALQUIER io.Writer
	var builder strings.Builder
	fmt.Fprintf(&builder, "Hola, %s!", "Mundo") // *strings.Builder es un io.Writer
	fmt.Println("\nEscrito en un strings.Builder:", builder.String())
	fmt.Fprintln(os.Stdout, "Escrito directamente en os.Stdout (también es un io.Writer)")

	// Nuestro Writer envolviendo a otro
	var destino strings.Builder
	writerMayusculas := MayusculasWriter{Destino: &destino}
	fmt.Fprintf(writerMayusculas, "este texto se vuelve mayúsculas")
	fmt.Println("Resultado del Writer personalizado:", destino.String())

	// io.Reader: strings.NewReader convierte un string en un io.Reader
	lector := strings.NewReader("línea uno\nlínea dos\nlínea tres")

	// bufio.Reader envuelve un io.Reader y agrega métodos como ReadString
	lectorConBuffer := bufio.NewReader(lector)
	fmt.Println("\nLeyendo línea por línea con bufio.Reader:")
	for {
		linea, err := lectorConBuffer.ReadString('\n')
		if linea != "" {
			fmt.Println("->", strings.TrimSuffix(linea, "\n"))
		}
		// io.EOF ("end of file") no es un fallo: indica que no hay más datos
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil { // cualquier otro error sí es un problema real
			fmt.Println("Error leyendo:", err)
			break
		}
	}

	// io.Copy copia de cualquier io.Reader a cualquier io.Writer, en bloques,
	// sin cargar todo en memoria (sirve igual para un archivo de 10 GB)
	fmt.Println("\nUsando io.Copy para copiar de un Reader a un Writer:")
	origen := strings.NewReader("contenido copiado con io.Copy\n")
	n, err := io.Copy(os.Stdout, origen)
	fmt.Println("bytes copiados:", n, "error:", err)

	// Los Readers se pueden encadenar: aquí se leen solo los primeros 10 bytes
	limitado := io.LimitReader(strings.NewReader("0123456789ABCDEF"), 10)
	datos, _ := io.ReadAll(limitado)
	fmt.Println("io.LimitReader (10 bytes):", string(datos))
}
