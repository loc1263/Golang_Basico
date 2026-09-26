// Package sqlite es otro adaptador del MISMO puerto (RepositorioPedidos),
// esta vez con database/sql (temas 30-31). Los casos de uso no saben ni les
// importa cuál de los dos se usa: se elige en main.go.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	_ "modernc.org/sqlite"

	"golang-basico/50-arquitectura-limpia/internal/dominio"
)

type Repositorio struct{ db *sql.DB }

func NuevoRepositorio(dsn string) (*Repositorio, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS pedidos (
		id        TEXT PRIMARY KEY,
		cliente   TEXT NOT NULL,
		items     TEXT NOT NULL, -- JSON: simplificación para el ejemplo
		estado    TEXT NOT NULL,
		creado_en TEXT NOT NULL
	)`)
	return &Repositorio{db: db}, err
}

// Close libera la conexión. Se llama "Close" (y no "Cerrar") por
// convención: así el tipo cumple la interfaz estándar io.Closer.
func (r *Repositorio) Close() error { return r.db.Close() }

// Modelo de persistencia separado del de dominio: la forma en que se guarda
// (JSON, columnas, fechas como texto) es un detalle de este adaptador.
func (r *Repositorio) Guardar(ctx context.Context, p *dominio.Pedido) error {
	items, err := json.Marshal(p.Items)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO pedidos (id, cliente, items, estado, creado_en) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET estado = excluded.estado, items = excluded.items`,
		p.ID, p.Cliente, string(items), string(p.Estado), p.CreadoEn.Format(time.RFC3339))
	return err
}

type escaneable interface{ Scan(dest ...any) error }

func escanear(fila escaneable) (*dominio.Pedido, error) {
	var p dominio.Pedido
	var items, estado, creado string
	if err := fila.Scan(&p.ID, &p.Cliente, &items, &estado, &creado); err != nil {
		return nil, err
	}
	p.Estado = dominio.EstadoPedido(estado)
	p.CreadoEn, _ = time.Parse(time.RFC3339, creado)
	if err := json.Unmarshal([]byte(items), &p.Items); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repositorio) Buscar(ctx context.Context, id string) (*dominio.Pedido, error) {
	p, err := escanear(r.db.QueryRowContext(ctx,
		"SELECT id, cliente, items, estado, creado_en FROM pedidos WHERE id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		// Traducir el error técnico al error de dominio: las capas internas
		// no deben conocer sql.ErrNoRows
		return nil, dominio.ErrPedidoNoEncontrado
	}
	return p, err
}

func (r *Repositorio) Listar(ctx context.Context) ([]*dominio.Pedido, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, cliente, items, estado, creado_en FROM pedidos ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var lista []*dominio.Pedido
	for rows.Next() {
		p, err := escanear(rows)
		if err != nil {
			return nil, err
		}
		lista = append(lista, p)
	}
	return lista, rows.Err()
}
