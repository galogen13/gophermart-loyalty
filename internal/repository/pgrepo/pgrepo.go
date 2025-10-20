package pgrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/galogen13/gophermart-loyalty/internal/logger"
	"github.com/galogen13/gophermart-loyalty/internal/service/market"
	"github.com/galogen13/gophermart-loyalty/migrations"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
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

	if err := storage.runMigrations(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &storage, nil
}

func (repo *PGRepo) Close() error {
	repo.pool.Close()
	return nil
}

func (repo *PGRepo) AddUser(ctx context.Context, user *market.User) error {

	var id int64

	err := repo.pool.QueryRow(
		ctx,
		`INSERT INTO users(login, password) VALUES ($1, $2) 
		RETURNING id`,
		user.Login, user.Password).Scan(&id)

	if err != nil {
		if isDuplicateKeyError(err) {
			return market.ErrUserLoginAlreadyInUse
		}
		return fmt.Errorf("failed to execute insert user: %w", err)
	}

	user.ID = &id

	return nil
}

func (repo *PGRepo) GetUser(ctx context.Context, user *market.User) error {

	row := repo.pool.QueryRow(ctx, `SELECT id, login, password
		FROM users WHERE login=$1;`, user.Login)
	err := row.Scan(&user.ID, &user.Login, &user.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return market.ErrUserNotExists
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	return nil
}

func (repo *PGRepo) runMigrations() error {
	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	sqlDB := stdlib.OpenDBFromPool(repo.pool)
	defer sqlDB.Close()

	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create database driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		logger.Log.Info("migrations are already installed")
	} else {
		logger.Log.Info("migrations installed succesfully")
	}

	return nil
}

func isDuplicateKeyError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return true
		}
	}
	return false
}
