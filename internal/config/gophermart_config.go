package config

import (
	"flag"

	"github.com/caarlos0/env/v6"
)

type GophermartConfig struct {
	RunAddress           string `env:"RUN_ADDRESS"`
	DatabaseURI          string `env:"DATABASE_URI"`
	AccrualSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
}

func GetGophermartConfig() (*GophermartConfig, error) {

	var cfg GophermartConfig

	err := env.Parse(&cfg)
	if err != nil {
		return nil, err
	}

	runAddressFlag := flag.String("a", "localhost:8080", "host address")
	databaseURIFlag := flag.String("d", "", "database URI")
	accrualSystemAddressFlag := flag.String("r", "", "accrual system address")

	flag.Parse()

	if cfg.RunAddress == "" {
		cfg.RunAddress = *runAddressFlag
	}

	if cfg.DatabaseURI == "" {
		cfg.DatabaseURI = *databaseURIFlag
	}

	if cfg.AccrualSystemAddress == "" {
		cfg.AccrualSystemAddress = *accrualSystemAddressFlag
	}

	return &cfg, nil
}
