// Tema 40: Serialización adicional (encoding/xml y encoding/csv)
// Complementa el tema 19 (JSON). La idea es la misma: struct tags indican
// cómo se mapea cada campo, y Marshal/Unmarshal (o Encoder/Decoder para
// streams) convierten entre structs de Go y el formato de texto.
//   - XML: común en integraciones antiguas, SOAP, feeds RSS, facturación.
//   - CSV: exportar/importar datos tabulares (planillas, reportes).
//
// Ejecutar con: go run ./40-serializacion
package main

import (
	"encoding/csv"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

// ======================== XML ========================

// Tags XML más usados:
//
//	xml:"nombre"        elemento hijo <nombre>
//	xml:"id,attr"       atributo del elemento: <libro id="...">
//	xml:",chardata"     texto dentro del elemento
//	xml:"a>b"           anidamiento sin struct intermedio: <a><b>...</b></a>
//	xml:",omitempty"    omitir si está vacío
//	xml:"-"             ignorar el campo
type Libro struct {
	XMLName   xml.Name `xml:"libro"` // nombre del elemento raíz de este struct
	ID        string   `xml:"id,attr"`
	Titulo    string   `xml:"titulo"`
	Autor     Autor    `xml:"autor"`
	Anio      int      `xml:"publicacion>anio"`
	Etiquetas []string `xml:"etiquetas>etiqueta"` // slice -> elementos repetidos
	Notas     string   `xml:"notas,omitempty"`
	Interno   string   `xml:"-"`
}

type Autor struct {
	Pais   string `xml:"pais,attr"`
	Nombre string `xml:",chardata"`
}

type Biblioteca struct {
	XMLName xml.Name `xml:"biblioteca"`
	Libros  []Libro  `xml:"libro"`
}

const xmlEntrada = `<?xml version="1.0" encoding="UTF-8"?>
<biblioteca>
  <libro id="b2">
    <titulo>Cien años de soledad</titulo>
    <autor pais="CO">Gabriel García Márquez</autor>
    <publicacion><anio>1967</anio></publicacion>
    <etiquetas><etiqueta>novela</etiqueta><etiqueta>realismo mágico</etiqueta></etiquetas>
    <campo_desconocido>se ignora</campo_desconocido>
  </libro>
  <libro id="b3">
    <titulo>Rayuela</titulo>
    <autor pais="AR">Julio Cortázar</autor>
    <publicacion><anio>1963</anio></publicacion>
  </libro>
</biblioteca>`

func ejemploXML() {
	fmt.Println("===== XML: Marshal =====")
	libro := Libro{
		ID:        "b1",
		Titulo:    "Ficciones",
		Autor:     Autor{Pais: "AR", Nombre: "Jorge Luis Borges"},
		Anio:      1944,
		Etiquetas: []string{"cuentos", "fantástico"},
		Interno:   "no se serializa",
	}
	// MarshalIndent produce XML legible; xml.Header agrega la declaración <?xml?>
	datos, err := xml.MarshalIndent(libro, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(xml.Header + string(datos))

	fmt.Println("\n===== XML: Unmarshal =====")
	var bib Biblioteca
	if err := xml.Unmarshal([]byte(xmlEntrada), &bib); err != nil {
		log.Fatal(err)
	}
	for _, l := range bib.Libros {
		fmt.Printf("[%s] %q de %s (%s), %d, etiquetas=%v\n",
			l.ID, l.Titulo, l.Autor.Nombre, l.Autor.Pais, l.Anio, l.Etiquetas)
	}

	// XML mal formado produce un *xml.SyntaxError
	var roto Biblioteca
	err = xml.Unmarshal([]byte("<biblioteca><libro></biblioteca>"), &roto)
	if errSintaxis, ok := errors.AsType[*xml.SyntaxError](err); ok {
		fmt.Printf("XML inválido en la línea %d: %s\n", errSintaxis.Line, errSintaxis.Msg)
	}
}

// ======================== CSV ========================

type Venta struct {
	Producto string
	Cantidad int
	Precio   float64
}

// CSV no tiene tags ni tipos: cada fila es un []string y la conversión de
// tipos es manual (strconv).
func escribirCSV(w io.Writer, ventas []Venta) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"producto", "cantidad", "precio"}); err != nil {
		return err
	}
	for _, v := range ventas {
		fila := []string{
			v.Producto,
			strconv.Itoa(v.Cantidad),
			strconv.FormatFloat(v.Precio, 'f', 2, 64),
		}
		if err := cw.Write(fila); err != nil {
			return err
		}
	}
	// El writer usa un buffer: Flush es obligatorio, y los errores de
	// escritura se consultan con Error() después
	cw.Flush()
	return cw.Error()
}

func leerCSV(r io.Reader) ([]Venta, error) {
	cr := csv.NewReader(r)
	cr.Comment = '#' // ignorar líneas que empiezan con #
	cr.TrimLeadingSpace = true

	encabezado, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("leer encabezado: %w", err)
	}
	// Mapear nombre de columna -> índice: tolera columnas en otro orden
	col := make(map[string]int)
	for i, nombre := range encabezado {
		col[nombre] = i
	}

	var ventas []Venta
	for {
		fila, err := cr.Read()      // leer de a una fila: sirve para archivos enormes
		if errors.Is(err, io.EOF) { // fin del archivo: no es un error real
			break
		}
		if err != nil {
			return nil, err
		}
		linea, _ := cr.FieldPos(0)
		cantidad, err := strconv.Atoi(fila[col["cantidad"]])
		if err != nil {
			return nil, fmt.Errorf("línea %d: cantidad inválida %q", linea, fila[col["cantidad"]])
		}
		precio, err := strconv.ParseFloat(fila[col["precio"]], 64)
		if err != nil {
			return nil, fmt.Errorf("línea %d: precio inválido %q", linea, fila[col["precio"]])
		}
		ventas = append(ventas, Venta{Producto: fila[col["producto"]], Cantidad: cantidad, Precio: precio})
	}
	return ventas, nil
}

func ejemploCSV() {
	fmt.Println("\n===== CSV: escribir =====")
	ventas := []Venta{
		{"Teclado", 2, 45.5},
		{"Mouse, inalámbrico", 1, 19.9}, // la coma obliga a usar comillas
		{`Cable "USB-C"`, 5, 7.25},      // las comillas se escapan duplicándolas
	}
	// Escribir a la salida estándar (os.Stdout es un io.Writer)
	if err := escribirCSV(os.Stdout, ventas); err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n===== CSV: leer =====")
	entrada := `# reporte de ventas
precio,producto,cantidad
45.50,Teclado,2
19.90,"Mouse, inalámbrico",1
7.25,"Cable ""USB-C""",5
`
	leidas, err := leerCSV(strings.NewReader(entrada))
	if err != nil {
		log.Fatal(err)
	}
	total := 0.0
	for _, v := range leidas {
		subtotal := float64(v.Cantidad) * v.Precio
		total += subtotal
		fmt.Printf("  %-20s %2d x %6.2f = %7.2f\n", v.Producto, v.Cantidad, v.Precio, subtotal)
	}
	fmt.Printf("  %-20s %21.2f\n", "TOTAL", total)

	fmt.Println("\n===== CSV: errores =====")
	_, err = leerCSV(strings.NewReader("producto,cantidad,precio\nTeclado,dos,45.5\n"))
	fmt.Println("  ", err)
	// Por defecto todas las filas deben tener la misma cantidad de campos
	_, err = leerCSV(strings.NewReader("producto,cantidad,precio\nTeclado,2\n"))
	fmt.Println("  ", err)

	// Otros separadores: muchas planillas en español exportan con ';'
	fmt.Println("\n===== CSV con separador ';' =====")
	r := csv.NewReader(strings.NewReader("nombre;saldo\nAna;1.234,56\n"))
	r.Comma = ';'
	filas, _ := r.ReadAll() // ReadAll: cómodo para archivos chicos
	fmt.Printf("   %q\n", filas)
}

func main() {
	ejemploXML()
	ejemploCSV()
}
