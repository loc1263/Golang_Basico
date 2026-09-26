package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContar(t *testing.T) {
	casos := []struct {
		nombre string
		texto  string
		want   Conteo
	}{
		{"vacío", "", Conteo{}},
		{"una línea", "hola mundo\n", Conteo{Lineas: 1, Palabras: 2, Caracteres: 11, Bytes: 11}},
		{"espacios extra", "  a   b  \n\n c", Conteo{Lineas: 2, Palabras: 3, Caracteres: 13, Bytes: 13}},
		{"acentos (runas vs bytes)", "año café\n", Conteo{Lineas: 1, Palabras: 2, Caracteres: 9, Bytes: 11}},
		{"sin salto final", "uno dos tres", Conteo{Lineas: 0, Palabras: 3, Caracteres: 12, Bytes: 12}},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			t.Parallel() // los subtests independientes pueden correr en paralelo
			got, err := Contar(strings.NewReader(c.texto))
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Errorf("Contar(%q) = %+v; se esperaba %+v", c.texto, got, c.want)
			}
		})
	}
}

// Test de la CLI completa: argumentos, archivos temporales y códigos de salida
func TestEjecutar(t *testing.T) {
	// t.TempDir crea un directorio que se borra solo al terminar el test
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	os.WriteFile(a, []byte("uno dos\n"), 0o644)
	os.WriteFile(b, []byte("tres\ncuatro\n"), 0o644)

	t.Run("archivos y total", func(t *testing.T) {
		var out, errOut bytes.Buffer
		codigo := ejecutar([]string{a, b}, nil, &out, &errOut)
		if codigo != 0 {
			t.Fatalf("código = %d, stderr = %s", codigo, errOut.String())
		}
		lineas := strings.Split(strings.TrimSpace(out.String()), "\n")
		ultima := strings.Fields(lineas[len(lineas)-1]) // "3 4 20 20 total"
		if ultima[0] != "3" || ultima[1] != "4" || ultima[4] != "total" {
			t.Errorf("línea total inesperada: %v\nsalida:\n%s", ultima, out.String())
		}
	})

	t.Run("stdin", func(t *testing.T) {
		var out bytes.Buffer
		ejecutar(nil, strings.NewReader("a b c"), &out, &out)
		if !strings.Contains(out.String(), "(stdin)") {
			t.Errorf("salida sin (stdin): %s", out.String())
		}
	})

	t.Run("archivo inexistente devuelve 1", func(t *testing.T) {
		var out, errOut bytes.Buffer
		if codigo := ejecutar([]string{filepath.Join(dir, "no-existe")}, nil, &out, &errOut); codigo != 1 {
			t.Errorf("código = %d; se esperaba 1", codigo)
		}
	})

	t.Run("flag desconocido devuelve 2", func(t *testing.T) {
		var out, errOut bytes.Buffer
		if codigo := ejecutar([]string{"-xyz"}, nil, &out, &errOut); codigo != 2 {
			t.Errorf("código = %d; se esperaba 2", codigo)
		}
	})

	t.Run("-version", func(t *testing.T) {
		var out bytes.Buffer
		ejecutar([]string{"-version"}, nil, &out, &out)
		if !strings.HasPrefix(out.String(), "contador dev") {
			t.Errorf("versión = %q", out.String())
		}
	})
}

// Los Example son tests (se verifica la salida del comentario "Output:") y
// además aparecen como ejemplos en la documentación generada (pkg.go.dev)
func ExampleContar() {
	c, _ := Contar(strings.NewReader("Go es simple\ny rápido\n"))
	fmt.Printf("%d líneas, %d palabras\n", c.Lineas, c.Palabras)
	// Output: 2 líneas, 5 palabras
}

// Fuzzing (Go 1.18+): Go genera entradas aleatorias buscando fallos.
// En CI corre solo con las semillas; para fuzzear de verdad:
//
//	go test -fuzz=FuzzContar -fuzztime=30s ./52-ci-cd
func FuzzContar(f *testing.F) {
	f.Add("hola mundo")
	f.Add("año\n\n  ñ")
	f.Add("\xff\xfe") // UTF-8 inválido
	f.Fuzz(func(t *testing.T, s string) {
		c, err := Contar(strings.NewReader(s))
		if err != nil {
			t.Fatal(err)
		}
		// Propiedades que SIEMPRE deben cumplirse, sea cual sea la entrada
		if c.Bytes != len(s) {
			t.Errorf("Bytes = %d; len = %d", c.Bytes, len(s))
		}
		if c.Palabras > c.Caracteres || c.Lineas > c.Caracteres {
			t.Errorf("conteo imposible: %+v", c)
		}
	})
}
