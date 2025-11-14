package connectionpool

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresPool implements ResourcePool
type PostgresPool struct {
	Pool *pgxpool.Pool
}

func (p *PostgresPool) close() {
	logger.V(1).Info("Closing PostgreSQL pool")
	p.Pool.Close()
}

// NewPostgresPool : create new pgxpool.Pool
func NewPostgresPool(ctx context.Context, connStr string, maxConns int32) (ResourcePool, error) {
	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		logger.Error(err, "Failed to parse PostgreSQL connection string")
		return nil, err
	}
	if maxConns > 0 {
		logger.Info("Initializing PostgreSQL pool with max connections", "maxConns", maxConns)
		cfg.MaxConns = maxConns
	} else {
		logger.Info("Initialized PostgreSQL pool with default connection settings")
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		logger.Error(err, "Failed to create PostgreSQL pool")
		return nil, err
	}
	logger.Info("PostgreSQL pool created", "maxConns", pool.Config().MaxConns)
	return &PostgresPool{Pool: pool}, nil
}
