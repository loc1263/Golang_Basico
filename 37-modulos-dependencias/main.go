// Tema 37: Módulos, versionado y dependencias externas
//
// Un MÓDULO es un conjunto de paquetes que se versionan juntos. Lo define el
// archivo go.mod en su carpeta raíz; este proyecto es el módulo "golang-basico".
//
// Qué aprenderás:
//   - Para qué sirven go.mod y go.sum.
//   - Los comandos para agregar, actualizar y quitar dependencias.
//   - Versionado semántico y cómo elige Go las versiones.
//   - Usar un paquete externo y resolver choques de nombres con alias.
//
// Archivos clave:
//
//	go.mod  nombre del módulo, versión mínima de Go, dependencias (require)
//	        y herramientas (tool)
//	go.sum  hashes criptográficos de cada dependencia descargada: garantiza
//	        que todos compilen exactamente el mismo código. Se sube a git
//	        junto con go.mod y nunca se edita a mano.
//
// Comandos esenciales (en la raíz del módulo):
//
//	go mod init github.com/usuario/proyecto   crear un módulo nuevo
//	go get github.com/google/uuid@v1.6.0      agregar o cambiar a una versión exacta
//	go get github.com/google/uuid@latest      última versión publicada
//	go get github.com/google/uuid@none        quitar la dependencia
//	go get -u ./...                           actualizar todas (menor/parche)
//	go get -u tool                            actualizar las herramientas del go.mod
//	go mod tidy                               agregar lo que falta y quitar lo que sobra
//	go list -m all                            listar todas las dependencias
//	go list -m -u all                         ver cuáles tienen versiones nuevas
//	go mod why github.com/google/uuid         explicar por qué se necesita
//	go mod vendor                             copiar las dependencias a ./vendor
//
// Versionado semántico (semver) vMAYOR.MENOR.PARCHE:
//
//	PARCHE  correcciones compatibles        (v1.6.0 -> v1.6.1)
//	MENOR   funciones nuevas compatibles    (v1.6.0 -> v1.7.0)
//	MAYOR   cambios incompatibles. Desde v2, la ruta de import cambia:
//	        github.com/go-playground/validator/v10  (el /v10 es obligatorio)
//
// Go usa "Minimal Version Selection": si dos dependencias piden versiones
// distintas de un mismo módulo, usa la MAYOR de las mínimas pedidas, no la
// última publicada. Así el build es reproducible: no cambia solo porque
// alguien publicó una versión nueva.
//
// La directiva "go" de go.mod indica la versión mínima de Go que necesita el
// módulo. Si una dependencia exige una versión más nueva, "go get" la sube.
//
// Ejecutar con: go run ./37-modulos-dependencias
package main

import (
	"fmt"
	"runtime/debug"
	"strings"
	"uuid" // paquete ESTÁNDAR uuid (Go 1.27+)

	// Dependencia externa: se descargó con "go get" y quedó registrada en
	// go.mod. Como su nombre de paquete también es "uuid", le damos un ALIAS
	// (guuid) para evitar el choque con el paquete estándar.
	guuid "github.com/google/uuid"

	// Paquete del propio módulo: ruta = nombre del módulo + carpeta
	"golang-basico/21-paquetes-propios/matematica"
)

func main() {
	// 1) Dependencia externa
	fmt.Println("1) Dependencia externa github.com/google/uuid (importada como guuid)")
	id := guuid.New() // UUID versión 4 (aleatorio)
	fmt.Println("   UUID nuevo:    ", id)
	fmt.Println("   versión:       ", id.Version())

	parseado, err := guuid.Parse("f47ac10b-58cc-4372-a567-0e02b2c3d479")
	fmt.Println("   UUID parseado: ", parseado, "error:", err)
	_, err = guuid.Parse("no-es-un-uuid")
	fmt.Println("   UUID inválido: ", err)

	// Desde Go 1.27 la librería estándar incluye su propio paquete uuid.
	// Regla práctica: si la librería estándar ya resuelve el problema,
	// prefiérela antes que agregar una dependencia (menos código de
	// terceros que mantener y auditar). google/uuid se mantiene aquí solo
	// como ejemplo de dependencia externa.
	fmt.Println("\n   Con el paquete estándar uuid (Go 1.27+):")
	fmt.Println("   uuid.New():    ", uuid.New())
	fmt.Println("   uuid.NewV7():  ", uuid.NewV7(), "(ordenable por fecha de creación)")

	// 2) Paquete propio del mismo módulo (tema 21)
	fmt.Println("\n2) Paquete interno del módulo")
	fmt.Println("   matematica.Sumar(2, 3) =", matematica.Sumar(2, 3))

	// 3) Información del módulo guardada dentro del binario.
	// El compilador registra qué módulos y versiones se usaron; se puede
	// consultar también con: go version -m ruta/al/binario
	fmt.Println("\n3) Información de build (runtime/debug.ReadBuildInfo)")
	info, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("   no disponible")
		return
	}
	fmt.Println("   módulo principal:", info.Main.Path)
	fmt.Println("   versión de Go:   ", info.GoVersion)
	fmt.Println("   dependencias enlazadas en este binario:")
	for _, dep := range info.Deps {
		fmt.Printf("     %-30s %s\n", dep.Path, dep.Version)
	}
	// Solo aparecen las dependencias que ESTE programa importa, no todas las
	// del go.mod: el compilador descarta lo que no se usa.

	fmt.Println("\n4) Algunos ajustes de compilación:")
	for _, s := range info.Settings {
		if strings.HasPrefix(s.Key, "GO") || s.Key == "CGO_ENABLED" {
			fmt.Printf("     %s=%s\n", s.Key, s.Value)
		}
	}
}
