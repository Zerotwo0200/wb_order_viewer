package repo

import (
	"context"

	"github.com/example/wb-order-service/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresOrderRepo struct {
	Pool *pgxpool.Pool
}

func NewPostgresOrderRepo(pool *pgxpool.Pool) *PostgresOrderRepo {
	return &PostgresOrderRepo{Pool: pool}
}

func (r *PostgresOrderRepo) Upsert(ctx context.Context, id string, raw []byte) error {
	_, err := r.Pool.Exec(ctx, `INSERT INTO orders(order_uid, payload) VALUES($1, $2)
        ON CONFLICT (order_uid) DO UPDATE SET payload = EXCLUDED.payload`, id, raw)
	return err
}

func (r *PostgresOrderRepo) LoadAll(ctx context.Context, fn func(id string, raw []byte) error) error {
	rows, err := r.Pool.Query(ctx, `SELECT order_uid, payload FROM orders`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var raw []byte
		if err := rows.Scan(&id, &raw); err != nil {
			return err
		}
		if err := fn(id, raw); err != nil {
			return err
		}
	}
	return rows.Err()
}

var _ domain.OrderRepository = (*PostgresOrderRepo)(nil)

// EnsureSchema — создать необходимые таблицы, если отсутствуют.
func EnsureSchema(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS orders (
  order_uid text PRIMARY KEY,
  payload jsonb NOT NULL
);`)
	return err
}
