package loyalty

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/galogen13/gophermart-loyalty/internal/auth"
	"github.com/galogen13/gophermart-loyalty/internal/config"
	"github.com/galogen13/gophermart-loyalty/internal/handlers"
	"github.com/galogen13/gophermart-loyalty/internal/logger"
	"github.com/galogen13/gophermart-loyalty/internal/service/market"

	"go.uber.org/zap"
)

type Storage interface {
	AddUser(ctx context.Context, user *market.User) error
	GetUser(ctx context.Context, user *market.User) error
	AddOrder(ctx context.Context, order *market.Order) error
	GetUserOrders(ctx context.Context, user *market.User) ([]*market.Order, error)
	GetOrderByNumber(ctx context.Context, order *market.Order) error
}

type GophermartLoyaltyService struct {
	Storage Storage
	Config  *config.GophermartConfig
	handlers.AuthService
}

func NewGophermartLoyaltyService(config *config.GophermartConfig, storage Storage) *GophermartLoyaltyService {
	return &GophermartLoyaltyService{
		Config:      config,
		Storage:     storage,
		AuthService: auth.NewJWTAuthService(config.JWTSecret)}
}

func (ls *GophermartLoyaltyService) Start() error {

	r := loyaltyRouter(ls)
	logger.Log.Info("Running server",
		zap.String("address", ls.Config.RunAddress),
		zap.String("AccrualSystemAddress", ls.Config.AccrualSystemAddress),
	)
	return http.ListenAndServe(ls.Config.RunAddress, r)
}

func (ls *GophermartLoyaltyService) RegisterUser(ctx context.Context, user *market.User) error {

	if err := ls.Storage.AddUser(ctx, user); err != nil {
		return fmt.Errorf("failed to register user: %w", err)
	}

	return nil
}

func (ls *GophermartLoyaltyService) LoginUser(ctx context.Context, user *market.User) error {

	if err := ls.Storage.GetUser(ctx, user); err != nil {
		return fmt.Errorf("failed to login user: %w", err)
	}

	return nil
}

func (ls *GophermartLoyaltyService) AddOrder(ctx context.Context, order *market.Order) error {

	existedOrder := &market.Order{Number: order.Number}
	err := ls.Storage.GetOrderByNumber(ctx, existedOrder)
	if err == nil {
		if *order.UserID == *existedOrder.UserID {
			return market.ErrOrderAlreadyExists
		} else {
			return market.ErrOrderBelongsToAnotherUser
		}
	} else {
		if !errors.Is(err, market.ErrOrderNotExists) {
			return fmt.Errorf("failed to get order: %w", err)
		}
	}

	order.Status = market.OrderStatusNew

	if err := ls.Storage.AddOrder(ctx, order); err != nil {
		return fmt.Errorf("failed to add order: %w", err)
	}

	return nil
}

func (ls *GophermartLoyaltyService) GetUserOrders(ctx context.Context, user *market.User) ([]*market.Order, error) {

	orders, err := ls.Storage.GetUserOrders(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}

	return orders, nil
}
