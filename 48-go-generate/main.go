// Tema 48: go generate y generación de código
// Go no tiene macros ni metaprogramación en tiempo de compilación. Para
// evitar escribir código repetitivo a mano se GENERA código Go normal, que
// se guarda en el repositorio junto al resto.
//
// `go generate` busca comentarios con la forma exacta
//
//	//go:generate comando argumentos...
//
// (sin espacio entre // y go) y ejecuta esos comandos en el directorio del
// paquete. NO se ejecuta automáticamente con go build/run/test: es un paso
// manual que corre el desarrollador cuando cambia la fuente.
//
// Usos habituales: stringer (este ejemplo), mocks (mockgen, moq), código de
// protobuf/gRPC (tema 43), SQL tipado (sqlc), embebido de assets, enums.
//
// Comandos (desde la raíz del proyecto):
//
//	go generate ./48-go-generate        regenera estado_string.go y paises_gen.go
//	go generate -x ./48-go-generate     muestra cada comando ejecutado
//	go run ./48-go-generate
//
// stringer se declara como herramienta del módulo (Go 1.24+):
//
//	go get -tool golang.org/x/tools/cmd/stringer
//
// Eso agrega una línea `tool` en go.mod y permite ejecutarla con
// `go tool stringer`, con la versión fijada en go.mod (sin instalar nada global).
package main

import (
	"fmt"
)

// Las directivas se ejecutan en el orden en que aparecen en el archivo.
// Va primero el generador propio: stringer analiza los tipos del paquete, y
// el paquete no compila hasta que exista paises_gen.go (Tarea usa CodigoPais).
//
// gen/ es un programa Go que lee paises.csv y escribe paises_gen.go.
// Si se agrega una fila al CSV, basta correr go generate.
//
//go:generate go run ./gen -entrada=paises.csv -salida=paises_gen.go

// ---------- 1) stringer: método String() para enums ----------

// La directiva genera estado_string.go con func (Estado) String() string.
// Sin esto habría que mantener un switch a mano cada vez que se agrega un valor.
//
//go:generate go tool stringer -type=Estado
type Estado int

const (
	Pendiente Estado = iota
	EnProceso
	Completado
	Cancelado
)

// -linecomment usa el comentario de cada línea como texto en vez del nombre
//
//go:generate go tool stringer -type=Prioridad -linecomment
type Prioridad int

const (
	Baja    Prioridad = iota + 1 // baja
	Media                        // media
	Alta                         // ¡alta!
	Critica                      // CRÍTICA
)

type Tarea struct {
	Titulo    string
	Estado    Estado
	Prioridad Prioridad
	Pais      CodigoPais // tipo definido en el código generado
}

func main() {
	fmt.Println("1) Enums con stringer")
	// fmt usa String() automáticamente (interfaz fmt.Stringer, tema 27)
	for e := Pendiente; e <= Cancelado; e++ {
		fmt.Printf("   Estado(%d) = %v\n", int(e), e)
	}
	// Valores fuera de rango no rompen: muestran "Estado(99)"
	fmt.Println("   valor inválido:", Estado(99))

	fmt.Println("\n   Prioridades con -linecomment:")
	for p := Baja; p <= Critica; p++ {
		fmt.Printf("   %d -> %s\n", p, p)
	}

	fmt.Println("\n2) Código generado desde paises.csv")
	for _, c := range TodosLosPaises() {
		info, _ := c.Info()
		fmt.Printf("   %s  %-10s %s  %s\n", c, info.Nombre, info.Moneda, info.Prefijo)
	}
	if _, ok := CodigoPais("BR").Info(); !ok {
		fmt.Println("   BR no está en el CSV: agregarlo y ejecutar go generate")
	}

	fmt.Println("\n3) Todo junto")
	t := Tarea{Titulo: "Traducir docs", Estado: EnProceso, Prioridad: Alta, Pais: PaisMX}
	info, _ := t.Pais.Info()
	fmt.Printf("   %+v (país: %s)\n", t, info.Nombre)
}
