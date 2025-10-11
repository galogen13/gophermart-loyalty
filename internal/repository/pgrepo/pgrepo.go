package pgrepo

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PGRepo struct {
	pool *pgxpool.Pool
}

func NewPGRepo(ctx context.Context, ps string) (*PGRepo, error) {

	config, err := pgxpool.ParseConfig(ps)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	config.MaxConns = 5
	config.MinConns = 1
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	storage := PGRepo{pool: pool}

	// if err := storage.runMigrations(); err != nil {
	// 	return nil, fmt.Errorf("failed to run migrations: %w", err)
	// }

	return &storage, nil
}

func (storage *PGRepo) AddUser(ctx context.Context, login, pass string) error {
	return nil
}
