// Tema 42: ORMs y migraciones (GORM + golang-migrate)
// Un ORM (Object-Relational Mapper) mapea structs de Go a tablas y genera el
// SQL por nosotros. GORM (gorm.io/gorm) es el más usado en Go.
//   - Menos código repetitivo para CRUD, relaciones, paginación...
//   - Menos control y "magia" que oculta el SQL real; para consultas complejas
//     o de alto rendimiento muchos equipos prefieren database/sql (temas 30-31)
//     o generadores como sqlc.
//
// Este archivo: GORM con SQLite en memoria (driver Go puro github.com/glebarez/sqlite).
// La carpeta migrar/: migraciones versionadas con golang-migrate.
//
// Ejecutar con: go run ./42-orm-migraciones
// Migraciones:  go run ./42-orm-migraciones/migrar -accion=demo
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// gorm.Model agrega ID, CreatedAt, UpdatedAt y DeletedAt (soft delete).
// Por convención: struct Autor -> tabla "autors" (GORM pluraliza en inglés);
// con TableName() elegimos el nombre explícitamente.
type Autor struct {
	gorm.Model
	Nombre string  `gorm:"size:100;not null;uniqueIndex"`
	Libros []Libro // relación uno-a-muchos (has many): usa Libro.AutorID
}

func (Autor) TableName() string { return "autores" }

type Libro struct {
	ID      uint   `gorm:"primaryKey"`
	Titulo  string `gorm:"size:200;not null"`
	Anio    int    `gorm:"index"`
	Precio  float64
	AutorID uint  // clave foránea
	Autor   Autor // relación inversa (belongs to)
	// Hooks, validaciones y timestamps se pueden omitir si no hacen falta
}

// Hook: GORM llama a BeforeCreate antes de cada INSERT de un Libro
func (l *Libro) BeforeCreate(tx *gorm.DB) error {
	if l.Anio > time.Now().Year() {
		return fmt.Errorf("año %d en el futuro", l.Anio)
	}
	return nil
}

func titulo(s string) { fmt.Printf("\n=== %s ===\n", s) }

func main() {
	// logger.Info imprime cada SQL generado: ideal para aprender qué hace GORM.
	// Lo activamos solo en algunas secciones con db.Debug().
	sqlLogger := logger.New(log.New(os.Stdout, "", 0), logger.Config{
		LogLevel: logger.Silent, // db.Debug() lo sube a Info temporalmente
		Colorful: false,
	})
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{Logger: sqlLogger})
	if err != nil {
		log.Fatal(err)
	}
	// GORM usa por debajo un *sql.DB (tema 30). Igual que allí, SQLite en
	// memoria necesita una sola conexión para que todas vean la misma base.
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)

	// AutoMigrate crea/actualiza tablas según los structs. Cómodo para
	// prototipos y tests; en producción se prefieren migraciones versionadas
	// (ver migrar/), porque AutoMigrate nunca borra columnas ni permite revertir.
	titulo("AutoMigrate")
	if err := db.AutoMigrate(&Autor{}, &Libro{}); err != nil {
		log.Fatal(err)
	}
	tablas, _ := db.Migrator().GetTables()
	fmt.Println("tablas creadas:", tablas)

	// ---------- CREATE ----------
	titulo("Create (con asociaciones)")
	borges := Autor{Nombre: "Borges", Libros: []Libro{
		{Titulo: "Ficciones", Anio: 1944, Precio: 12.5},
		{Titulo: "El Aleph", Anio: 1949, Precio: 11},
	}}
	// Create inserta el autor y también sus libros (asociación)
	if err := db.Create(&borges).Error; err != nil {
		log.Fatal(err)
	}
	fmt.Printf("autor id=%d creado con %d libros\n", borges.ID, len(borges.Libros))

	cortazar := Autor{Nombre: "Cortázar"}
	db.Create(&cortazar)
	// Inserción masiva en un solo INSERT
	db.Create(&[]Libro{
		{Titulo: "Rayuela", Anio: 1963, Precio: 15, AutorID: cortazar.ID},
		{Titulo: "Bestiario", Anio: 1951, Precio: 9.9, AutorID: cortazar.ID},
	})

	// El hook BeforeCreate rechaza este libro
	err = db.Create(&Libro{Titulo: "Del futuro", Anio: 3000, AutorID: cortazar.ID}).Error
	fmt.Println("hook rechazó:", err)

	// ---------- READ ----------
	titulo("Consultas (con el SQL generado vía Debug)")
	var libro Libro
	db.Debug().First(&libro, "titulo = ?", "Rayuela") // First: ORDER BY id LIMIT 1
	fmt.Printf("-> %s (%d)\n", libro.Titulo, libro.Anio)

	var baratos []Libro
	db.Debug().Where("precio < ?", 12).Order("precio").Find(&baratos)
	for _, l := range baratos {
		fmt.Printf("-> %-10s $%.2f\n", l.Titulo, l.Precio)
	}

	// Not found: First devuelve gorm.ErrRecordNotFound (Find NO, devuelve vacío)
	err = db.First(&libro, 999).Error
	fmt.Println("buscar id 999 -> ErrRecordNotFound?", errors.Is(err, gorm.ErrRecordNotFound))

	titulo("Preload (cargar relaciones)")
	// Sin Preload, autor.Libros quedaría vacío. Preload hace una 2da consulta.
	var autores []Autor
	db.Preload("Libros").Order("nombre").Find(&autores)
	for _, a := range autores {
		fmt.Printf("%s:", a.Nombre)
		for _, l := range a.Libros {
			fmt.Printf(" %q", l.Titulo)
		}
		fmt.Println()
	}

	titulo("Joins y agregaciones a un struct propio")
	type Resumen struct {
		Nombre   string
		Cantidad int
		Promedio float64
	}
	var resumen []Resumen
	db.Model(&Libro{}).
		Select("autores.nombre AS nombre, COUNT(*) AS cantidad, AVG(libros.precio) AS promedio").
		Joins("JOIN autores ON autores.id = libros.autor_id").
		Group("autores.nombre").
		Order("nombre").
		Scan(&resumen)
	for _, r := range resumen {
		fmt.Printf("%-9s libros=%d precio promedio=%.2f\n", r.Nombre, r.Cantidad, r.Promedio)
	}

	// ---------- UPDATE ----------
	titulo("Update")
	// Update de un campo; Updates con map para incluir valores cero
	db.Model(&libro).Update("precio", 17.5)
	db.Model(&Libro{}).Where("anio < ?", 1950).Updates(map[string]any{"precio": gorm.Expr("precio * 0.9")})
	db.First(&libro, libro.ID)
	fmt.Printf("nuevo precio de %s: %.2f\n", libro.Titulo, libro.Precio)

	// ---------- TRANSACCIÓN ----------
	titulo("Transacción")
	// Si la función devuelve error, GORM hace rollback automáticamente
	err = db.Transaction(func(tx *gorm.DB) error {
		nuevo := Autor{Nombre: "Pizarnik"}
		if err := tx.Create(&nuevo).Error; err != nil {
			return err
		}
		return tx.Create(&Autor{Nombre: "Borges"}).Error // viola uniqueIndex
	})
	fmt.Println("transacción falló:", err != nil)
	var n int64
	db.Model(&Autor{}).Where("nombre = ?", "Pizarnik").Count(&n)
	fmt.Println("¿se guardó Pizarnik? (rollback esperado):", n == 1)

	// ---------- DELETE ----------
	titulo("Soft delete (gorm.Model)")
	db.Delete(&cortazar) // UPDATE autores SET deleted_at = ... (no borra la fila)
	var visibles, todos int64
	db.Model(&Autor{}).Count(&visibles)
	db.Unscoped().Model(&Autor{}).Count(&todos) // Unscoped incluye los borrados
	fmt.Printf("autores visibles=%d, incluyendo borrados=%d\n", visibles, todos)
	db.Unscoped().Delete(&cortazar) // borrado definitivo (DELETE real)

	// Raw SQL cuando el ORM no alcanza
	titulo("SQL crudo")
	var total float64
	db.Raw("SELECT SUM(precio) FROM libros").Scan(&total)
	fmt.Printf("valor total del catálogo: %.2f\n", total)
}
