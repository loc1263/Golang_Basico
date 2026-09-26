package main

import (
	"errors"
	"testing"
)

// Test simple: el nombre empieza con "Test" y recibe *testing.T
func TestSumar(t *testing.T) {
	resultado := Sumar(2, 3)
	esperado := 5
	if resultado != esperado {
		// t.Errorf marca el test como fallido, pero sigue ejecutándolo.
		// Convención del mensaje: "Funcion(args) = obtenido; se esperaba X"
		t.Errorf("Sumar(2, 3) = %d; se esperaba %d", resultado, esperado)
	}
}

// Test guiado por tabla (table-driven): la forma idiomática en Go de probar
// muchos casos con poco código. Agregar un caso nuevo es agregar una línea.
func TestEsPar(t *testing.T) {
	casos := []struct {
		nombre   string
		entrada  int
		esperado bool
	}{
		{"cero es par", 0, true},
		{"dos es par", 2, true},
		{"tres es impar", 3, false},
		{"número negativo par", -4, true},
		{"número negativo impar", -3, false},
	}

	for _, c := range casos {
		// t.Run crea un subtest con nombre propio: si falla, se ve cuál caso fue.
		// Se puede ejecutar uno solo: go test -run 'TestEsPar/tres' ./18-testing
		t.Run(c.nombre, func(t *testing.T) {
			resultado := EsPar(c.entrada)
			if resultado != c.esperado {
				t.Errorf("EsPar(%d) = %v; se esperaba %v", c.entrada, resultado, c.esperado)
			}
		})
	}
}

func TestDividir(t *testing.T) {
	t.Run("división válida", func(t *testing.T) {
		resultado, err := Dividir(10, 2)
		if err != nil {
			// t.Fatalf marca el fallo y DETIENE este test: sin un resultado
			// válido no tiene sentido seguir verificando
			t.Fatalf("no se esperaba error, se obtuvo: %v", err)
		}
		if resultado != 5 {
			t.Errorf("Dividir(10, 2) = %v; se esperaba 5", resultado)
		}
	})

	t.Run("división por cero devuelve ErrDivisionPorCero", func(t *testing.T) {
		_, err := Dividir(10, 0)
		// Verificar el error CONCRETO, no solo que haya "algún" error
		if !errors.Is(err, ErrDivisionPorCero) {
			t.Errorf("Dividir(10, 0) error = %v; se esperaba %v", err, ErrDivisionPorCero)
		}
	})
}
