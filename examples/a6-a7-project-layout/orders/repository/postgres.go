package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/semirm-dev/goplusplus/examples/a6-a7-project-layout/orders"
)

// Implements orders.Repository.
type Postgres struct {
	db *sql.DB
}

func NewPostgres(db *sql.DB) *Postgres {
	return &Postgres{db: db}
}

func (p *Postgres) Save(ctx context.Context, o orders.Order) error {
	_, err := p.db.ExecContext(ctx, "INSERT INTO orders (id, total, code) VALUES ($1, $2, $3)", o.ID, o.Total, o.Code)
	return err
}

func (p *Postgres) Find(ctx context.Context, id string) (orders.Order, error) {
	var o orders.Order
	err := p.db.QueryRowContext(ctx, "SELECT id, total, code FROM orders WHERE id = $1", id).Scan(&o.ID, &o.Total, &o.Code)
	if errors.Is(err, sql.ErrNoRows) {
		return o, orders.ErrNotFound
	}
	return o, err
}
