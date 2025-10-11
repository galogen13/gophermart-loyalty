package main

import (
	"context"
	"log"

	"github.com/galogen13/gophermart-loyalty/internal/config"
	"github.com/galogen13/gophermart-loyalty/internal/logger"
	"github.com/galogen13/gophermart-loyalty/internal/repository/pgrepo"
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

	storage, err = pgrepo.NewPGRepo(context.Background(), config.DatabaseURI)
	if err != nil {
		return err
	}

	ls := loyalty.NewGophermartLoyaltyService(config, storage)

	if err := ls.Start(); err != nil {
		return err
	}

	return nil
}
