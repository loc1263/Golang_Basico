// Tema 13: defer, panic y recover
//
// Qué aprenderás:
//   - defer: dejar programada una llamada para cuando la función termine.
//   - panic: detener la ejecución ante un error grave e inesperado.
//   - recover: capturar un panic dentro de un defer.
//
// Importante: panic NO reemplaza al manejo de errores del tema 12. Se usa
// para situaciones que "no deberían ocurrir" (errores de programación). Los
// errores esperados se devuelven como valores de tipo error.
//
// Ejecutar con: go run ./13-defer-panic-recover
package main

import "fmt"

// defer pospone una llamada hasta que la función actual termina, sin
// importar cómo termine (return normal o panic). Se usa sobre todo para
// liberar recursos justo después de obtenerlos:
//
//	f, err := os.Open("datos.txt")
//	if err != nil { return err }
//	defer f.Close() // se cerrará al salir, pase lo que pase
func ejemploDefer() {
	fmt.Println("1. Inicio de la función")
	defer fmt.Println("3. Esto se ejecuta al final (defer)")
	fmt.Println("2. Antes de terminar la función")
}

// Varios defer se apilan y se ejecutan en orden inverso (LIFO: el último
// registrado es el primero en ejecutarse)
func ejemploDeferMultiple() {
	for i := 1; i <= 3; i++ {
		defer fmt.Println("defer registrado con i =", i)
	}
	fmt.Println("Función terminando; ahora se ejecutan los defer en orden inverso")
}

// Los argumentos de una llamada diferida se evalúan en el momento del
// defer, no al final
func ejemploEvaluacion() {
	x := 1
	defer fmt.Println("x al registrar el defer:", x) // imprime 1
	x = 2
	fmt.Println("x al final de la función:", x)
}

// recover solo funciona dentro de una función diferida. Detiene el panic y
// devuelve el valor que se pasó a panic (o nil si no hubo panic).
// Aquí convertimos un panic en un error normal usando retornos con nombre:
// la función diferida puede modificar "err" antes de que la función retorne.
func operacionSegura(a, b int) (resultado int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("se recuperó de un panic: %v", r)
		}
	}()

	resultado = a / b // si b es 0, el runtime de Go provoca un panic
	return resultado, nil
}

// panic también se puede lanzar explícitamente
func obtenerElemento(lista []string, i int) string {
	if i < 0 || i >= len(lista) {
		panic(fmt.Sprintf("índice %d fuera de rango (longitud %d)", i, len(lista)))
	}
	return lista[i]
}

func main() {
	ejemploDefer()

	fmt.Println()
	ejemploDeferMultiple()

	fmt.Println()
	ejemploEvaluacion()

	fmt.Println("\nLlamando a operacionSegura(10, 2):")
	res, err := operacionSegura(10, 2)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Resultado:", res)
	}

	fmt.Println("\nLlamando a operacionSegura(10, 0):")
	res, err = operacionSegura(10, 0)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Resultado:", res)
	}

	// Recuperar un panic lanzado explícitamente
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("\nPanic recuperado:", r)
			}
		}()
		fmt.Println(obtenerElemento([]string{"a", "b"}, 5))
	}()

	fmt.Println("\nEl programa sigue ejecutándose normalmente después de los panics recuperados")
	// Sin recover, un panic termina el programa e imprime el "stack trace"
	// (la lista de funciones que se estaban ejecutando).
}
