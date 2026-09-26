// Tema 42 (parte 2): Migraciones versionadas con golang-migrate
// Una migración es un cambio de esquema versionado: cada versión tiene un
// archivo .up.sql (aplicar) y uno .down.sql (revertir). La herramienta guarda
// en la tabla schema_migrations qué versión tiene la base, y aplica solo las
// que faltan. Así todos los entornos (dev, CI, producción) evolucionan igual.
//
// Los .sql se EMBEBEN en el binario con go:embed, así el ejecutable no
// depende de archivos sueltos al desplegarlo.
//
// Uso (desde la raíz del proyecto; la base es un archivo local):
//
//	go run ./42-orm-migraciones/migrar                 aplica todas (up)
//	go run ./42-orm-migraciones/migrar -accion=down -pasos=1   revierte 1
//	go run ./42-orm-migraciones/migrar -accion=version
//	go run ./42-orm-migraciones/migrar -accion=demo    recorrido completo
//
// Por qué es un programa aparte del ejemplo GORM (main.go de la carpeta
// padre): en producción las migraciones suelen correr como un paso separado
// (en CI/CD o un job previo) ANTES de levantar la aplicación.
// También existe la CLI oficial: github.com/golang-migrate/migrate/v4/cmd/migrate
package main

import (
	"embed"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite" // registra el esquema sqlite:// (modernc, Go puro)
	"github.com/golang-migrate/migrate/v4/source/iofs"       // fuente: un fs.FS (nuestro embed)
)

//go:embed migraciones/*.sql
var archivosMigracion embed.FS

func nuevoMigrador(rutaDB string) (*migrate.Migrate, error) {
	fuente, err := iofs.New(archivosMigracion, "migraciones")
	if err != nil {
		return nil, err
	}
	// La URL indica driver y ubicación de la base: sqlite://ruta/al/archivo.db
	return migrate.NewWithSourceInstance("iofs", fuente, "sqlite://"+rutaDB)
}

func mostrarVersion(m *migrate.Migrate) {
	v, sucia, err := m.Version()
	switch {
	case errors.Is(err, migrate.ErrNilVersion):
		fmt.Println("   versión actual: ninguna (base vacía)")
	case err != nil:
		log.Fatal(err)
	default:
		// "sucia" = una migración falló a mitad: requiere intervención manual
		// (corregir y luego m.Force(version))
		fmt.Printf("   versión actual: %d (sucia=%v)\n", v, sucia)
	}
}

// ignorarSinCambios: Up/Down devuelven ErrNoChange si no hay nada que hacer,
// lo cual no es un error real.
func ignorarSinCambios(err error) error {
	if errors.Is(err, migrate.ErrNoChange) {
		fmt.Println("   (sin cambios: ya estaba en esa versión)")
		return nil
	}
	return err
}

func demo(rutaDB string) {
	os.Remove(rutaDB) // empezar desde cero
	m, err := nuevoMigrador(rutaDB)
	if err != nil {
		log.Fatal(err)
	}
	defer m.Close()

	fmt.Println("1) Estado inicial")
	mostrarVersion(m)

	fmt.Println("\n2) Up: aplicar todas las migraciones pendientes")
	if err := ignorarSinCambios(m.Up()); err != nil {
		log.Fatal(err)
	}
	mostrarVersion(m)

	fmt.Println("\n3) Up otra vez (idempotente)")
	if err := ignorarSinCambios(m.Up()); err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n4) Steps(-1): revertir la última (quita la columna isbn)")
	if err := m.Steps(-1); err != nil {
		log.Fatal(err)
	}
	mostrarVersion(m)

	fmt.Println("\n5) Migrate(1): ir exactamente a la versión 1")
	if err := m.Migrate(1); err != nil {
		log.Fatal(err)
	}
	mostrarVersion(m)

	fmt.Println("\n6) Up: volver a la última")
	if err := m.Up(); err != nil {
		log.Fatal(err)
	}
	mostrarVersion(m)
	fmt.Println("\nBase de datos resultante:", rutaDB)
}

func main() {
	accion := flag.String("accion", "up", "up | down | version | demo")
	pasos := flag.Int("pasos", 0, "con down: cuántas migraciones revertir (0 = todas)")
	rutaDB := flag.String("db", "42-orm-migraciones/biblioteca.db", "archivo SQLite")
	flag.Parse()

	if *accion == "demo" {
		demo(*rutaDB)
		return
	}

	m, err := nuevoMigrador(*rutaDB)
	if err != nil {
		log.Fatal(err)
	}
	defer m.Close()

	switch *accion {
	case "up":
		err = ignorarSinCambios(m.Up())
	case "down":
		if *pasos > 0 {
			err = m.Steps(-*pasos)
		} else {
			err = ignorarSinCambios(m.Down())
		}
	case "version":
	default:
		log.Fatalf("acción desconocida %q", *accion)
	}
	if err != nil {
		log.Fatal(err)
	}
	mostrarVersion(m)
}
