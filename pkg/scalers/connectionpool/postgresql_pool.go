package connectionpool

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresPool implements ResourcePool
type PostgresPool struct {
	Pool *pgxpool.Pool
}

func (p *PostgresPool) Close() {
	p.Pool.Close()
}

// NewPostgresPool : create new pgxpool.Pool
func NewPostgresPool(ctx context.Context, connStr string, maxConns int32) (ResourcePool, error) {
	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}
	if maxConns > 0 {
		cfg.MaxConns = maxConns
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &PostgresPool{Pool: pool}, nil
}
