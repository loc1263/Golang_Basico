// Tema 24: Paquete time
//
// Qué aprenderás:
//   - Obtener la hora actual, formatear y parsear fechas.
//   - El "layout" de referencia de Go (en lugar de códigos como YYYY-MM-DD).
//   - Zonas horarias, duraciones, aritmética y comparación de fechas.
//   - Medir cuánto tarda una operación.
//
// Ejecutar con: go run ./24-time
package main

import (
	"fmt"
	"time"
)

func main() {
	ahora := time.Now()
	fmt.Println("Ahora:", ahora)

	// Formato: Go no usa "YYYY-MM-DD". Se escribe CÓMO se vería una fecha de
	// referencia fija con el formato deseado. La fecha de referencia es:
	//     Mon Jan 2 15:04:05 MST 2006
	// Truco para recordarla: 01/02 03:04:05PM '06 -0700 (1, 2, 3, 4, 5, 6, 7).
	//     2006 = año, 01 = mes, 02 = día, 15 = hora (24h), 04 = minutos, 05 = segundos
	fmt.Println("\nFormatos comunes:")
	fmt.Println("RFC3339:        ", ahora.Format(time.RFC3339))  // estándar para APIs y JSON
	fmt.Println("time.DateOnly:  ", ahora.Format(time.DateOnly)) // "2006-01-02"
	fmt.Println("time.DateTime:  ", ahora.Format(time.DateTime)) // "2006-01-02 15:04:05"
	fmt.Println("Día/mes/año:    ", ahora.Format("02/01/2006 15:04:05"))
	fmt.Println("Solo hora:      ", ahora.Format("15:04"))

	// Parsear: string -> time.Time, usando el mismo tipo de layout.
	// time.Parse interpreta la fecha en UTC si el texto no indica una zona.
	fecha, err := time.Parse(time.DateTime, "2025-12-25 10:30:00")
	if err != nil {
		fmt.Println("Error al parsear fecha:", err)
		return
	}
	fmt.Println("\nFecha parseada (UTC):", fecha)

	// time.ParseInLocation la interpreta en una zona horaria concreta.
	// LoadLocation usa la base de datos de zonas horarias del sistema (o la
	// que trae la instalación de Go). Para un binario que se ejecutará donde
	// no existe esa base (p. ej. un contenedor mínimo), se puede embeber con
	//     import _ "time/tzdata"
	if bogota, err := time.LoadLocation("America/Bogota"); err == nil {
		enBogota, _ := time.ParseInLocation(time.DateTime, "2025-12-25 10:30:00", bogota)
		fmt.Println("Misma hora local en Bogotá:", enBogota)
		fmt.Println("Esa hora de Bogotá en UTC:", enBogota.UTC())
	}

	// Un texto que no respeta el formato produce un error descriptivo
	_, err = time.Parse(time.DateOnly, "25/12/2025")
	fmt.Println("Parsear con formato incorrecto:", err)

	// Componentes individuales
	fmt.Println("\nComponentes de la fecha parseada:")
	fmt.Println("Año:", fecha.Year())
	fmt.Printf("Mes: %s (número %d)\n", fecha.Month(), fecha.Month())
	fmt.Println("Día:", fecha.Day())
	fmt.Println("Día de la semana:", fecha.Weekday()) // los nombres vienen en inglés

	// time.Duration representa un intervalo (internamente, en nanosegundos)
	duracion := 90 * time.Minute
	fmt.Println("\nDuración:", duracion)
	fmt.Println("En horas:", duracion.Hours())
	fmt.Println("En segundos:", duracion.Seconds())
	if d, err := time.ParseDuration("1h15m30s"); err == nil {
		fmt.Println("Duración parseada de \"1h15m30s\":", d, "=", d.Minutes(), "minutos")
	}

	// Sumar y restar
	enUnaSemana := ahora.Add(7 * 24 * time.Hour)
	fmt.Println("\nDentro de una semana:", enUnaSemana.Format(time.DateOnly))
	haceUnaHora := ahora.Add(-time.Hour)
	fmt.Println("Hace una hora:", haceUnaHora.Format(time.TimeOnly))
	// AddDate suma años, meses y días respetando el calendario
	fmt.Println("Dentro de 1 mes:", ahora.AddDate(0, 1, 0).Format(time.DateOnly))

	// Diferencia entre dos fechas: Sub devuelve una Duration
	fmt.Println("\nDiferencia entre ahora y dentro de una semana:", enUnaSemana.Sub(ahora))

	// Comparar: usa Before, After y Equal (no ==, que también compara la zona horaria)
	fmt.Println("\n¿enUnaSemana es después de ahora?", enUnaSemana.After(ahora))
	fmt.Println("¿haceUnaHora es antes de ahora?", haceUnaHora.Before(ahora))

	// Medir el tiempo de una operación con time.Since
	inicio := time.Now()
	suma := 0
	for i := range 1_000_000 { // "_" en números: separador visual, no cambia el valor
		suma += i
	}
	fmt.Println("\nSuma calculada:", suma, "en", time.Since(inicio))

	// time.Sleep pausa la goroutine actual
	fmt.Println("\nEsperando 100 milisegundos...")
	time.Sleep(100 * time.Millisecond)
	fmt.Println("Listo.")
}
