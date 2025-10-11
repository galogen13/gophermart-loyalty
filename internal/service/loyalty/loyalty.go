package loyalty

import (
	"context"
	"net/http"

	"github.com/galogen13/gophermart-loyalty/internal/config"
	"github.com/galogen13/gophermart-loyalty/internal/logger"
	"go.uber.org/zap"
)

type Storage interface {
	AddUser(ctx context.Context, login, pass string) error
}

type GophermartLoyaltyService struct {
	Storage Storage
	Config  *config.GophermartConfig
}

func NewGophermartLoyaltyService(config *config.GophermartConfig, storage Storage) *GophermartLoyaltyService {
	return &GophermartLoyaltyService{Config: config, Storage: storage}
}

func (ls *GophermartLoyaltyService) Start() error {

	r := loyaltyRouter(ls)
	logger.Log.Info("Running server",
		zap.String("address", ls.Config.RunAddress),
		zap.String("AccrualSystemAddress", ls.Config.AccrualSystemAddress),
	)
	return http.ListenAndServe(ls.Config.RunAddress, r)
}

func (ls *GophermartLoyaltyService) RegisterUser(ctx context.Context, login, pass string) error {
	return nil
}
