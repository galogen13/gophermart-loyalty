package main

import (
	"context"
	"log"

	"github.com/galogen13/gophermart-loyalty/internal/auth"
	"github.com/galogen13/gophermart-loyalty/internal/config"
	"github.com/galogen13/gophermart-loyalty/internal/handlers"
	"github.com/galogen13/gophermart-loyalty/internal/logger"
	"github.com/galogen13/gophermart-loyalty/internal/repository/pgrepo"
	"github.com/galogen13/gophermart-loyalty/internal/service/accrual"
	"github.com/galogen13/gophermart-loyalty/internal/service/loyalty"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {

	config, err := config.GetGophermartConfig()
	if err != nil {
		return err
	}

	if err := logger.Initialize(); err != nil {
		return err
	}
	defer logger.Log.Sync()

	var storage loyalty.Storage

	ctx := context.Background()

	storage, err = pgrepo.NewPGRepo(ctx, config.DatabaseURI)
	if err != nil {
		return err
	}

	accrualService := accrual.NewAccrualService(config.AccrualSystemAddress, 5)

	authService := auth.NewJWTAuthService(config.JWTSecret)

	var ls handlers.LoyaltyService = loyalty.NewGophermartLoyaltyService(config, storage, accrualService, authService)

	if err := ls.Start(ctx); err != nil {
		return err
	}

	return nil
}
