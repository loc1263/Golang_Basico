// Tema 52: CI/CD para proyectos Go (GitHub Actions, golangci-lint, releases)
// Integración continua (CI): en cada push/PR se ejecutan automáticamente
// formato, análisis estático, tests (con -race) y build, para detectar
// problemas antes de mezclar código. Entrega continua (CD): al crear un tag
// de versión se compilan binarios para varias plataformas y se publican.
//
// Este tema es una pequeña CLI ("contador", como wc) con todo el andamiaje:
//
//	.github/workflows/ci.yml       pipeline de CI (lint, test, build multiplataforma)
//	.github/workflows/release.yml  release automático con GoReleaser al pushear un tag
//	.golangci.yml                  configuración del linter (formato v2)
//	.goreleaser.yaml               cómo compilar y empaquetar los binarios
//	Makefile                       los mismos pasos para ejecutarlos localmente
//
// IMPORTANTE: GitHub solo lee workflows desde .github/workflows en la RAÍZ
// del repositorio. Aquí están dentro de la carpeta del tema como ejemplo;
// para usarlos, cópialos a la raíz.
//
// Pasos del CI, ejecutables a mano desde la raíz del proyecto:
//
//	gofmt -l .                          ¿hay archivos sin formatear?
//	go vet ./...                        análisis estático incluido en Go
//	go test -race -cover ./52-ci-cd     tests con detector de carreras y cobertura
//	go run golang.org/x/vuln/cmd/govulncheck@latest ./...   vulnerabilidades conocidas
//	golangci-lint run ./52-ci-cd/...    decenas de linters juntos (https://golangci-lint.run)
//
// Compilación cruzada: Go compila para otro SO/arquitectura sin herramientas extra
//
//	GOOS=linux   GOARCH=arm64 go build -o contador-linux-arm64 ./52-ci-cd
//	GOOS=darwin  GOARCH=arm64 go build -o contador-mac ./52-ci-cd
//	GOOS=windows GOARCH=amd64 go build -o contador.exe ./52-ci-cd
//	(PowerShell: $env:GOOS="linux"; $env:GOARCH="arm64"; go build ...)
//	go tool dist list                   lista todas las plataformas soportadas
//
// Uso de la CLI:
//
//	echo "hola mundo" | go run ./52-ci-cd
//	go run ./52-ci-cd README.md 52-ci-cd/main.go
//	go run ./52-ci-cd -version
//	go build -ldflags "-X main.version=1.2.3" -o contador.exe ./52-ci-cd && ./contador.exe -version
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"unicode"
)

// Variables inyectadas en tiempo de compilación por GoReleaser / el Makefile:
//
//	-ldflags "-X main.version=1.2.3 -X main.commit=abc123 -X main.fecha=2026-01-01"
var (
	version = "dev"
	commit  = ""
	fecha   = ""
)

// Conteo es el resultado de analizar un texto
type Conteo struct {
	Lineas, Palabras, Caracteres, Bytes int
}

func (c *Conteo) Sumar(o Conteo) {
	c.Lineas += o.Lineas
	c.Palabras += o.Palabras
	c.Caracteres += o.Caracteres
	c.Bytes += o.Bytes
}

// Contar lee todo el Reader. Recibe un io.Reader (tema 27) para poder
// testearlo con strings.NewReader sin tocar archivos.
func Contar(r io.Reader) (Conteo, error) {
	var c Conteo
	br := bufio.NewReader(r)
	enPalabra := false
	for {
		ru, tam, err := br.ReadRune()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return c, err
		}
		c.Bytes += tam
		// Un byte UTF-8 inválido llega como utf8.RuneError de tamaño 1:
		// lo contamos igual como carácter (el fuzzing de main_test.go
		// descubre el caso si no se cuenta)
		c.Caracteres++
		if ru == '\n' {
			c.Lineas++
		}
		if unicode.IsSpace(ru) {
			enPalabra = false
		} else if !enPalabra {
			enPalabra = true
			c.Palabras++
		}
	}
	return c, nil
}

// infoVersion combina lo inyectado con -ldflags y lo que Go guarda
// automáticamente del control de versiones (si se compiló dentro de un repo git)
func infoVersion() string {
	c, f := commit, fecha
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			switch {
			case s.Key == "vcs.revision" && c == "":
				c = s.Value
			case s.Key == "vcs.time" && f == "":
				f = s.Value
			}
		}
	}
	if len(c) > 7 {
		c = c[:7]
	}
	if c == "" {
		c = "desconocido"
	}
	if f == "" {
		f = "desconocida"
	}
	return fmt.Sprintf("contador %s (commit %s, compilado %s)", version, c, f)
}

func imprimir(w io.Writer, c Conteo, nombre string) {
	fmt.Fprintf(w, "%8d %8d %8d %8d %s\n", c.Lineas, c.Palabras, c.Caracteres, c.Bytes, nombre)
}

// ejecutar contiene la lógica del programa y devuelve el código de salida.
// Separarla de main() permite testear la CLI completa (ver main_test.go).
func ejecutar(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("contador", flag.ContinueOnError)
	fs.SetOutput(stderr)
	verVersion := fs.Bool("version", false, "mostrar la versión y salir")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *verVersion {
		fmt.Fprintln(stdout, infoVersion())
		return 0
	}

	fmt.Fprintf(stdout, "%8s %8s %8s %8s\n", "líneas", "palabras", "chars", "bytes")
	if fs.NArg() == 0 {
		c, err := Contar(stdin)
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		imprimir(stdout, c, "(stdin)")
		return 0
	}

	var total Conteo
	codigo := 0
	for _, ruta := range fs.Args() {
		f, err := os.Open(ruta)
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			codigo = 1
			continue
		}
		c, err := Contar(f)
		f.Close()
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			codigo = 1
			continue
		}
		imprimir(stdout, c, ruta)
		total.Sumar(c)
	}
	if fs.NArg() > 1 {
		imprimir(stdout, total, "total")
	}
	return codigo
}

func main() {
	os.Exit(ejecutar(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
