// Tema 30: database/sql básico (CRUD con SQLite embebido)
// database/sql es la interfaz estándar de Go para bases de datos SQL. No trae
// ningún motor: se combina con un "driver" que se registra al importarlo.
// Aquí usamos modernc.org/sqlite, un SQLite escrito en Go puro (no requiere
// gcc ni un servidor externo), con una base de datos en memoria.
// Ejecutar con: go run ./30-database-sql
package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	// El import con "_" solo ejecuta el init() del paquete, que registra el
	// driver con el nombre "sqlite". No usamos ningún identificador del paquete.
	_ "modernc.org/sqlite"
)

type Producto struct {
	ID     int64
	Nombre string
	Precio float64
	Stock  int
}

// crearTabla ejecuta una sentencia DDL. Exec se usa para sentencias que no
// devuelven filas (CREATE, INSERT, UPDATE, DELETE).
func crearTabla(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE productos (
			id     INTEGER PRIMARY KEY AUTOINCREMENT,
			nombre TEXT    NOT NULL UNIQUE,
			precio REAL    NOT NULL,
			stock  INTEGER NOT NULL DEFAULT 0
		)`)
	return err
}

// Create: los "?" son placeholders. NUNCA concatenes valores en el SQL:
// el driver escapa los parámetros y así se evita la inyección SQL.
func insertarProducto(db *sql.DB, p Producto) (int64, error) {
	res, err := db.Exec(
		"INSERT INTO productos (nombre, precio, stock) VALUES (?, ?, ?)",
		p.Nombre, p.Precio, p.Stock,
	)
	if err != nil {
		return 0, fmt.Errorf("insertar %q: %w", p.Nombre, err)
	}
	return res.LastInsertId() // id generado por AUTOINCREMENT
}

// Read (una fila): QueryRow devuelve como máximo una fila. El error se
// entrega recién al llamar Scan; si no hubo filas es sql.ErrNoRows.
func buscarProducto(db *sql.DB, id int64) (Producto, error) {
	var p Producto
	err := db.QueryRow(
		"SELECT id, nombre, precio, stock FROM productos WHERE id = ?", id,
	).Scan(&p.ID, &p.Nombre, &p.Precio, &p.Stock)
	if err != nil {
		return Producto{}, fmt.Errorf("buscar id %d: %w", id, err)
	}
	return p, nil
}

// Read (varias filas): Query devuelve *sql.Rows, que hay que recorrer con
// Next(), leer con Scan() y cerrar SIEMPRE con Close() (usamos defer).
func listarProductos(db *sql.DB, precioMax float64) ([]Producto, error) {
	rows, err := db.Query(
		"SELECT id, nombre, precio, stock FROM productos WHERE precio <= ? ORDER BY precio",
		precioMax,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var productos []Producto
	for rows.Next() {
		var p Producto
		if err := rows.Scan(&p.ID, &p.Nombre, &p.Precio, &p.Stock); err != nil {
			return nil, err
		}
		productos = append(productos, p)
	}
	// rows.Err() informa errores ocurridos durante la iteración
	return productos, rows.Err()
}

// Update: RowsAffected indica cuántas filas cambiaron (0 = el id no existía)
func actualizarPrecio(db *sql.DB, id int64, nuevoPrecio float64) error {
	res, err := db.Exec("UPDATE productos SET precio = ? WHERE id = ?", nuevoPrecio, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("actualizar id %d: %w", id, sql.ErrNoRows)
	}
	return nil
}

// Delete
func eliminarProducto(db *sql.DB, id int64) error {
	_, err := db.Exec("DELETE FROM productos WHERE id = ?", id)
	return err
}

func main() {
	// sql.Open NO abre una conexión todavía: solo valida argumentos y prepara
	// un pool de conexiones. ":memory:" crea una base que vive mientras el
	// programa esté corriendo.
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Con SQLite en memoria cada conexión tendría su PROPIA base vacía, así
	// que limitamos el pool a una conexión. Con Postgres/MySQL no hace falta.
	db.SetMaxOpenConns(1)

	// Ping sí se conecta: es la forma de comprobar que todo funciona
	if err := db.Ping(); err != nil {
		log.Fatal("no se pudo conectar:", err)
	}

	if err := crearTabla(db); err != nil {
		log.Fatal(err)
	}

	// --- CREATE ---
	fmt.Println("== Insertar ==")
	iniciales := []Producto{
		{Nombre: "Teclado", Precio: 45.5, Stock: 10},
		{Nombre: "Mouse", Precio: 19.9, Stock: 25},
		{Nombre: "Monitor", Precio: 210, Stock: 3},
	}
	for _, p := range iniciales {
		id, err := insertarProducto(db, p)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("insertado %-8s con id %d\n", p.Nombre, id)
	}

	// La restricción UNIQUE hace fallar un nombre repetido
	if _, err := insertarProducto(db, Producto{Nombre: "Mouse", Precio: 1}); err != nil {
		fmt.Println("error esperado:", err)
	}

	// --- READ ---
	fmt.Println("\n== Buscar por id ==")
	p, err := buscarProducto(db, 2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v\n", p)

	// Distinguir "no existe" de otros errores con errors.Is
	_, err = buscarProducto(db, 99)
	if errors.Is(err, sql.ErrNoRows) {
		fmt.Println("no encontrado:", err)
	}

	fmt.Println("\n== Listar productos con precio <= 50 ==")
	baratos, err := listarProductos(db, 50)
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range baratos {
		fmt.Printf("  #%d %-8s $%7.2f  stock %d\n", p.ID, p.Nombre, p.Precio, p.Stock)
	}

	// --- UPDATE ---
	fmt.Println("\n== Actualizar ==")
	if err := actualizarPrecio(db, 1, 39.99); err != nil {
		log.Fatal(err)
	}
	p, _ = buscarProducto(db, 1)
	fmt.Printf("nuevo precio de %s: %.2f\n", p.Nombre, p.Precio)
	if err := actualizarPrecio(db, 42, 1); err != nil {
		fmt.Println("error esperado:", err)
	}

	// --- DELETE ---
	fmt.Println("\n== Eliminar ==")
	if err := eliminarProducto(db, 3); err != nil {
		log.Fatal(err)
	}
	todos, _ := listarProductos(db, 1e9)
	fmt.Println("productos restantes:", len(todos))

	// Valores NULL: un string de Go no puede ser nil, así que se usa
	// sql.NullString (o un *string) para columnas que admiten NULL.
	fmt.Println("\n== Valores NULL ==")
	var descripcion sql.NullString
	if err := db.QueryRow("SELECT NULL").Scan(&descripcion); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Valid=%v String=%q\n", descripcion.Valid, descripcion.String)
}
