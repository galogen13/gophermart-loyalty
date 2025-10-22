package config

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type GophermartConfig struct {
	RunAddress           string `mapstructure:"address"`
	DatabaseURI          string `mapstructure:"database"`
	AccrualSystemAddress string `mapstructure:"accrual"`
	JWTSecret            string `mapstructure:"secret"`
}

func GetGophermartConfig() (*GophermartConfig, error) {

	flags := map[string]struct {
		shorthand    string
		defaultValue any
		description  string
		envVar       string
	}{
		"address":  {"a", "localhost:8080", "Server address", "RUN_ADDRESS"},
		"database": {"d", "", "Database URI", "DATABASE_URI"},
		"accrual":  {"r", "", "Accrual system address", "ACCRUAL_SYSTEM_ADDRESS"},
		"secret":   {"s", "gophermart-secret-key", "Secret key for jwt", "SECRET_KEY"}, // значение по умолчанию указывать не стоило, но для автотестов пришлось
	}

	for name, config := range flags {
		switch v := config.defaultValue.(type) {
		case string:
			pflag.StringP(name, config.shorthand, v, config.description)
		case bool:
			pflag.BoolP(name, config.shorthand, v, config.description)
		case int:
			pflag.IntP(name, config.shorthand, v, config.description)
		}
	}

	pflag.Parse()

	v := viper.New()
	v.BindPFlags(pflag.CommandLine)
	v.AutomaticEnv()

	for flagName, config := range flags {
		v.BindEnv(flagName, config.envVar)
	}

	var cfg GophermartConfig

	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil

}
