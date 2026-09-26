// Tema 12: Manejo de errores
//
// Go no usa excepciones para los errores esperados (archivo inexistente,
// dato inválido...). Las funciones devuelven un valor de tipo "error" como
// último resultado, y quien llama debe revisarlo explícitamente.
// "error" es una interfaz con un solo método: Error() string.
//
// Qué aprenderás:
//   - Crear errores con errors.New y fmt.Errorf.
//   - Errores "centinela" y errores con tipo propio.
//   - Envolver errores con %w y examinarlos con errors.Is / errors.As / errors.AsType.
//   - Combinar varios errores con errors.Join.
//
// Ejecutar con: go run ./12-errores
package main

import (
	"errors"
	"fmt"
)

// fmt.Errorf crea un error con un mensaje formateado.
// Por convención, los mensajes de error van en minúscula y sin punto final.
func dividir(a, b float64) (float64, error) {
	if b == 0 {
		return 0, fmt.Errorf("división por cero: %v / %v", a, b)
	}
	return a / b, nil
}

// Error con tipo propio: un struct que implementa el método Error().
// Permite transportar datos adicionales (aquí, el campo que falló).
type ErrorValidacion struct {
	Campo   string
	Mensaje string
}

func (e *ErrorValidacion) Error() string {
	return fmt.Sprintf("validación fallida en %s: %s", e.Campo, e.Mensaje)
}

func validarEdad(edad int) error {
	if edad < 0 {
		return &ErrorValidacion{Campo: "edad", Mensaje: "no puede ser negativa"}
	}
	if edad > 130 {
		return &ErrorValidacion{Campo: "edad", Mensaje: "valor no realista"}
	}
	return nil // nil significa "sin error"
}

// Error "centinela": una variable de paquete con la que se compara.
// Por convención su nombre empieza con Err.
var ErrNoEncontrado = errors.New("elemento no encontrado")

func buscar(id int) (string, error) {
	datos := map[int]string{1: "manzana", 2: "banana"}
	valor, ok := datos[id]
	if !ok {
		return "", ErrNoEncontrado
	}
	return valor, nil
}

func main() {
	// Patrón básico: llamar, revisar el error, actuar
	resultado, err := dividir(10, 2)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Resultado:", resultado)
	}

	_, err = dividir(5, 0)
	if err != nil {
		fmt.Println("Error:", err)
	}

	// Error con tipo propio
	if err := validarEdad(-5); err != nil {
		fmt.Println("\nError de validación:", err)

		// errors.As busca en la cadena de errores uno del tipo indicado y,
		// si lo encuentra, lo guarda en la variable
		var errValidacion *ErrorValidacion
		if errors.As(err, &errValidacion) {
			fmt.Println("Campo con problema (errors.As):", errValidacion.Campo)
		}

		// Go 1.26+: errors.AsType hace lo mismo sin declarar la variable antes
		if ev, ok := errors.AsType[*ErrorValidacion](err); ok {
			fmt.Println("Campo con problema (errors.AsType):", ev.Campo)
		}
	}

	// errors.Is compara contra un error concreto (útil con centinelas)
	_, err = buscar(99)
	if errors.Is(err, ErrNoEncontrado) {
		fmt.Println("\nNo se encontró el elemento con id 99")
	}

	// Envolver un error: %w agrega contexto sin perder el error original.
	// Así el mensaje cuenta "dónde" falló y errors.Is sigue reconociéndolo.
	errorConContexto := fmt.Errorf("al procesar la búsqueda: %w", err)
	fmt.Println("Error con contexto:", errorConContexto)
	fmt.Println("¿Sigue siendo ErrNoEncontrado?", errors.Is(errorConContexto, ErrNoEncontrado))
	fmt.Println("Error original (errors.Unwrap):", errors.Unwrap(errorConContexto))

	// Con %v en lugar de %w el texto es el mismo, pero se pierde el vínculo
	sinVinculo := fmt.Errorf("al procesar la búsqueda: %v", err)
	fmt.Printf("Con %%v, ¿sigue siendo ErrNoEncontrado? %v\n", errors.Is(sinVinculo, ErrNoEncontrado))

	// errors.Join agrupa varios errores en uno (por ejemplo, al validar
	// varios campos). Devuelve nil si todos los errores son nil.
	conjunto := errors.Join(validarEdad(-1), validarEdad(200), validarEdad(30))
	fmt.Println("\nVarios errores juntos:\n" + conjunto.Error())
	fmt.Println("¿Contiene un ErrorValidacion?", errors.As(conjunto, new(*ErrorValidacion)))

	// Regla práctica: revisa SIEMPRE los errores. Ignorarlos con "_" solo es
	// aceptable cuando se sabe que la operación no puede fallar.
}
