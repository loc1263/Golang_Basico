// Tema 31: Transacciones y prepared statements
// Una transacción agrupa varias sentencias para que se apliquen TODAS o
// NINGUNA (atomicidad). En database/sql se inicia con db.Begin(), que
// devuelve un *sql.Tx; al final se llama Commit() o Rollback().
// Un prepared statement (db.Prepare) compila el SQL una vez y lo reutiliza
// con distintos parámetros: más eficiente cuando se ejecuta muchas veces.
// Ejecutar con: go run ./31-transacciones-sql
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

var ErrSaldoInsuficiente = errors.New("saldo insuficiente")

type Cuenta struct {
	Titular string
	Saldo   int
}

func prepararBase(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE cuentas (
			id      INTEGER PRIMARY KEY,
			titular TEXT    NOT NULL,
			saldo   INTEGER NOT NULL CHECK (saldo >= 0)
		)`)
	return err
}

// insertarCuentas usa un prepared statement: el SQL se envía y compila una
// sola vez, y luego solo se mandan los parámetros en cada Exec.
func insertarCuentas(db *sql.DB, cuentas []Cuenta) error {
	stmt, err := db.Prepare("INSERT INTO cuentas (titular, saldo) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close() // un statement ocupa recursos: hay que cerrarlo

	for _, c := range cuentas {
		if _, err := stmt.Exec(c.Titular, c.Saldo); err != nil {
			return fmt.Errorf("insertar %s: %w", c.Titular, err)
		}
	}
	return nil
}

// transferir mueve dinero entre dos cuentas dentro de una transacción.
// Si algo falla a mitad de camino, Rollback deshace lo ya hecho.
func transferir(ctx context.Context, db *sql.DB, origen, destino, monto int) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	// Patrón idiomático: defer Rollback. Si ya se hizo Commit, Rollback no
	// hace nada (devuelve sql.ErrTxDone, que ignoramos). Así nunca queda una
	// transacción abierta aunque retornemos antes por un error.
	defer tx.Rollback()

	// Dentro de la transacción se usan los métodos de tx, NO los de db
	var saldo int
	err = tx.QueryRowContext(ctx, "SELECT saldo FROM cuentas WHERE id = ?", origen).Scan(&saldo)
	if err != nil {
		return fmt.Errorf("leer cuenta origen: %w", err)
	}
	if saldo < monto {
		return fmt.Errorf("transferir %d desde cuenta %d: %w", monto, origen, ErrSaldoInsuficiente)
	}

	if _, err = tx.ExecContext(ctx, "UPDATE cuentas SET saldo = saldo - ? WHERE id = ?", monto, origen); err != nil {
		return err
	}

	res, err := tx.ExecContext(ctx, "UPDATE cuentas SET saldo = saldo + ? WHERE id = ?", monto, destino)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// El débito ya se aplicó, pero como retornamos sin Commit,
		// el defer tx.Rollback() lo deshace.
		return fmt.Errorf("cuenta destino %d no existe", destino)
	}

	return tx.Commit()
}

func mostrarSaldos(db *sql.DB) {
	rows, err := db.Query("SELECT id, titular, saldo FROM cuentas ORDER BY id")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, saldo int
		var titular string
		if err := rows.Scan(&id, &titular, &saldo); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  cuenta %d %-6s saldo %5d\n", id, titular, saldo)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1) // ver tema 30: SQLite en memoria = una sola conexión

	if err := prepararBase(db); err != nil {
		log.Fatal(err)
	}

	// Los ids se asignan en orden: Ana=1, Bruno=2, Carla=3
	err = insertarCuentas(db, []Cuenta{{"Ana", 1000}, {"Bruno", 500}, {"Carla", 0}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Saldos iniciales:")
	mostrarSaldos(db)

	ctx := context.Background()

	fmt.Println("\n1) Transferencia válida: Ana -> Bruno, 300")
	if err := transferir(ctx, db, 1, 2, 300); err != nil {
		fmt.Println("  error:", err)
	}
	mostrarSaldos(db)

	fmt.Println("\n2) Saldo insuficiente: Carla -> Ana, 50")
	err = transferir(ctx, db, 3, 1, 50)
	if errors.Is(err, ErrSaldoInsuficiente) {
		fmt.Println("  rechazada:", err)
	}
	mostrarSaldos(db)

	fmt.Println("\n3) Destino inexistente: Bruno -> cuenta 99, 100 (se hace rollback)")
	if err := transferir(ctx, db, 2, 99, 100); err != nil {
		fmt.Println("  error:", err)
	}
	fmt.Println("  El saldo de Bruno no cambió gracias al rollback:")
	mostrarSaldos(db)

	// Prepared statement reutilizado muchas veces dentro de una transacción:
	// tx.Prepare crea un statement ligado a esa transacción.
	fmt.Println("\n4) Carga masiva con tx.Prepare (todo o nada)")
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	stmt, err := tx.Prepare("UPDATE cuentas SET saldo = saldo + ? WHERE id = ?")
	if err != nil {
		log.Fatal(err)
	}
	for id := 1; id <= 3; id++ {
		if _, err := stmt.Exec(10, id); err != nil {
			tx.Rollback()
			log.Fatal(err)
		}
	}
	stmt.Close()
	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("  Bonificación de 10 aplicada a todas las cuentas:")
	mostrarSaldos(db)

	// Después de Commit la transacción queda cerrada
	fmt.Println("\nRollback tras Commit devuelve:", tx.Rollback())
}
