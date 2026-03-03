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
	logger.V(1).Info("Closing PostgreSQL pool", "server", p.Pool.Config().ConnConfig.Host)
	p.Pool.Close()
}

// NewPostgresPool : create new pgxpool.Pool
func NewPostgresPool(ctx context.Context, connStr string, maxConns int32) (ResourcePool, error) {
	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		logger.V(0).Error(err, "Failed to parse PostgreSQL connection string")
		return nil, err
	}
	if maxConns > 0 {
		logger.V(1).Info("Initializing PostgreSQL pool with max connections", "maxConns", maxConns)
		cfg.MaxConns = maxConns
	} else {
		logger.V(1).Info("Initialized PostgreSQL pool with default connection settings")
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		logger.V(0).Error(err, "Failed to create PostgreSQL pool")
		return nil, err
	}
	logger.V(1).Info("PostgreSQL pool created", "maxConns", pool.Config().MaxConns)
	return &PostgresPool{Pool: pool}, nil
}
