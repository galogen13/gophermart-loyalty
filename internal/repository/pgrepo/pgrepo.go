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

func (repo *PGRepo) AddUser(ctx context.Context, user market.User) (*market.User, error) {

	var id int64

	err := repo.pool.QueryRow(
		ctx,
		`INSERT INTO users(login, password) VALUES ($1, $2) 
		RETURNING id`,
		user.Login, user.Password).Scan(&id)

	if err != nil {
		if isDuplicateKeyError(err) {
			return nil, market.ErrUserLoginAlreadyInUse
		}
		return nil, fmt.Errorf("failed to execute insert user: %w", err)
	}

	user.ID = &id

	return &user, nil
}

func (repo *PGRepo) GetUserByLogin(ctx context.Context, user market.User) (*market.User, error) {

	row := repo.pool.QueryRow(ctx, `SELECT id, login, password
		FROM users WHERE login=$1;`, user.Login)
	err := row.Scan(&user.ID, &user.Login, &user.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, market.ErrUserNotExists
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (repo *PGRepo) AddOrder(ctx context.Context, order market.Order) error {

	_, err := repo.pool.Exec(
		ctx,
		`INSERT INTO orders("number", status, user_id)
		VALUES ($1, $2, $3);`,
		order.Number, order.Status, order.UserID)

	if err != nil {
		if isDuplicateKeyError(err) {
			return market.ErrOrderAlreadyExists
		}
		return fmt.Errorf("failed to execute add order: %w", err)
	}

	return nil
}

func (repo *PGRepo) GetOrdersByUserID(ctx context.Context, user market.User) ([]market.Order, error) {

	result := []market.Order{}
	rows, err := repo.pool.Query(ctx, `SELECT id, "number", status, accrual, uploaded_at, user_id
	FROM orders WHERE user_id = $1
	ORDER BY uploaded_at DESC;`, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			qOrder market.Order
		)

		err = rows.Scan(
			&qOrder.ID,
			&qOrder.Number,
			&qOrder.Status,
			&qOrder.Accrual,
			&qOrder.UploadedAt,
			&qOrder.UserID)

		if err != nil {
			return nil, fmt.Errorf("failed to scan query result GetOrders: %w", err)
		}

		result = append(result, qOrder)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (repo *PGRepo) GetOrderByNumber(ctx context.Context, order market.Order) (*market.Order, error) {

	row := repo.pool.QueryRow(
		ctx,
		`SELECT id, status, accrual, uploaded_at, user_id
		FROM orders WHERE number = $1;`,
		order.Number)

	err := row.Scan(&order.ID, &order.Status, &order.Accrual, &order.UploadedAt, &order.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, market.ErrOrderNotExists
		}
		return nil, fmt.Errorf("failed to get order by number: %w", err)
	}

	return &order, nil

}

func (repo *PGRepo) GetBalanceByUserID(ctx context.Context, user market.User) (*market.Balance, error) {

	row := repo.pool.QueryRow(
		ctx,
		`SELECT COALESCE(SUM(t.current), 0) AS current, COALESCE(SUM(t.withdrawn), 0) AS withdrawn
			FROM(SELECT accrual AS current, 0 AS withdrawn
				FROM orders 
				WHERE user_id = $1 AND status = $2 
				UNION ALL 
				SELECT -sum, sum
				FROM withdrawals 
				WHERE user_id = $1) AS t;`,
		user.ID, market.OrderStatusProcessed)

	var balance market.Balance

	err := row.Scan(&balance.Current, &balance.Withdrawn)
	if err != nil {
		return nil, fmt.Errorf("failed to get user balance: %w", err)
	}

	return &balance, nil

}

func (repo *PGRepo) GetWithdrawalsByUserID(ctx context.Context, user market.User) ([]market.Withdrawal, error) {

	result := []market.Withdrawal{}
	rows, err := repo.pool.Query(ctx, `SELECT order_number, sum, processed_at, user_id
		FROM withdrawals 
		WHERE user_id = $1
		ORDER BY processed_at DESC;`,
		user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get withdrawals: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var withdrawal market.Withdrawal

		err = rows.Scan(
			&withdrawal.OrderNumber,
			&withdrawal.Sum,
			&withdrawal.ProcessedAt,
			&withdrawal.UserID)

		if err != nil {
			return nil, fmt.Errorf("failed to scan query result GetWithdrawalsByUserID: %w", err)
		}

		result = append(result, withdrawal)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return result, nil

}

func (repo *PGRepo) GetOrdersByStatuses(ctx context.Context, statuses []market.OrderStatus) ([]market.Order, error) {
	result := []market.Order{}
	rows, err := repo.pool.Query(ctx, `SELECT id, "number", status, accrual, uploaded_at, user_id
		FROM orders WHERE status = ANY($1);`,
		statuses)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders by statuses: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var order market.Order

		err = rows.Scan(
			&order.ID,
			&order.Number,
			&order.Status,
			&order.Accrual,
			&order.UploadedAt,
			&order.UserID,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan query result GetOrdersByStatuses: %w", err)
		}

		result = append(result, order)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (repo *PGRepo) UpdateOrderAccrual(ctx context.Context, orderAccrual market.OrderAccrual) error {

	_, err := repo.pool.Exec(ctx,
		`UPDATE orders
		SET status=$1, accrual=$2
		WHERE number = $3;`,
		orderAccrual.Status, orderAccrual.Accrual, orderAccrual.Number,
	)

	if err != nil {
		return fmt.Errorf("failed to execute UpdateOrderAccrual: %w", err)
	}

	return nil
}

func (repo *PGRepo) AddWithdrawalWithBalanceCheck(ctx context.Context, withdrawal market.Withdrawal) error {

	result, err := repo.pool.Exec(
		ctx,
		`INSERT INTO withdrawals (order_number, sum, user_id)
        SELECT $1, $2, $3
        WHERE (
            SELECT COALESCE(SUM(t.current), 0) AS current
			FROM(SELECT accrual AS current
				FROM orders 
				WHERE user_id = $3 AND status = $4 
				UNION ALL 
				SELECT -sum
				FROM withdrawals 
				WHERE user_id = $3) AS t
        ) >= $2`,
		withdrawal.OrderNumber, withdrawal.Sum, withdrawal.UserID, market.OrderStatusProcessed)

	if err != nil {
		if isDuplicateKeyError(err) {
			return market.ErrWithdrawalAlreadyExists
		}
		return fmt.Errorf("failed to execute add withdrawal: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return market.ErrWithdrawalInsufficientFunds
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
	if err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("failed to apply migrations: %w", err)
		}
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
