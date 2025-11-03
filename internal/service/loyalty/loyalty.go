package loyalty

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/galogen13/gophermart-loyalty/internal/config"
	"github.com/galogen13/gophermart-loyalty/internal/handlers"
	"github.com/galogen13/gophermart-loyalty/internal/logger"
	"github.com/galogen13/gophermart-loyalty/internal/service/accrual"
	"github.com/galogen13/gophermart-loyalty/internal/service/market"

	"go.uber.org/zap"
)

//go:generate mockgen -destination=mocks/storage_mock.go . Storage
type Storage interface {
	AddUser(ctx context.Context, user market.User) (*market.User, error)
	GetUserByLogin(ctx context.Context, user market.User) (*market.User, error)
	AddOrder(ctx context.Context, order market.Order) error
	GetOrdersByUserID(ctx context.Context, user market.User) ([]market.Order, error)
	GetOrderByNumber(ctx context.Context, order market.Order) (*market.Order, error)
	GetBalanceByUserID(ctx context.Context, user market.User) (*market.Balance, error)
	GetWithdrawalsByUserID(ctx context.Context, user market.User) ([]market.Withdrawal, error)
	GetOrdersByStatuses(ctx context.Context, statuses []market.OrderStatus) ([]market.Order, error)
	UpdateOrderAccrual(ctx context.Context, orderAccrual market.OrderAccrual) error
	AddWithdrawalWithBalanceCheck(ctx context.Context, withdrawal market.Withdrawal) error
}

type GophermartLoyaltyService struct {
	Storage        Storage
	Config         *config.GophermartConfig
	AccrualService *accrual.AccrualService
	handlers.AuthService
}

func NewGophermartLoyaltyService(config *config.GophermartConfig, storage Storage, accrualService *accrual.AccrualService, authService handlers.AuthService) *GophermartLoyaltyService {
	return &GophermartLoyaltyService{
		Config:         config,
		Storage:        storage,
		AccrualService: accrualService,
		AuthService:    authService}
}

func (ls *GophermartLoyaltyService) Start(ctx context.Context) error {

	go ls.accrualsGetter(ctx)

	r := loyaltyRouter(ls)
	logger.Log.Info("Running server",
		zap.String("address", ls.Config.RunAddress),
		zap.String("AccrualSystemAddress", ls.Config.AccrualSystemAddress),
	)
	return http.ListenAndServe(ls.Config.RunAddress, r)
}

func (ls *GophermartLoyaltyService) accrualsGetter(ctx context.Context) {

	ls.AccrualService.Start(ctx)

	tickerPoll := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-tickerPoll.C:
			orders, err := ls.getUnprocessedOrders(ctx)
			if err != nil {
				logger.Log.Error("error getting unprocessed orders")
				continue
			}
			for _, order := range orders {
				go ls.AccrualService.AddJob(accrual.Job{OrderNumber: order.Number})
			}

		case result := <-ls.AccrualService.GetResults():
			go ls.updateOrderAccrual(ctx, result)
		case <-ctx.Done():
			return
		}
	}
}

func (ls *GophermartLoyaltyService) RegisterUser(ctx context.Context, userCred market.UserCredentials) (*market.User, error) {

	user := market.User{UserCredentials: userCred}
	newUser, err := ls.Storage.AddUser(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to register user: %w", err)
	}

	return newUser, nil
}

func (ls *GophermartLoyaltyService) LoginUser(ctx context.Context, userCred market.UserCredentials) (*market.User, error) {

	user := market.User{UserCredentials: userCred}
	existedUser, err := ls.Storage.GetUserByLogin(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to login user: %w", err)
	}

	return existedUser, nil
}

func (ls *GophermartLoyaltyService) AddOrder(ctx context.Context, order market.Order) error {

	err := market.OrderNumberLuhnCheck(order.Number)
	if err != nil {
		return fmt.Errorf("failed to add order: %w", err)
	}

	if order.UserID == nil {
		return errors.New("user ID expected")
	}

	existedOrder, err := ls.Storage.GetOrderByNumber(ctx, order)
	if err == nil {
		if *order.UserID == *existedOrder.UserID {
			return market.ErrOrderAlreadyExists
		} else {
			return market.ErrOrderBelongsToAnotherUser
		}
	} else {
		if !errors.Is(err, market.ErrOrderNotExists) {
			return fmt.Errorf("failed to add order: %w", err)
		}
	}

	order.Status = market.OrderStatusNew

	if err := ls.Storage.AddOrder(ctx, order); err != nil {
		return fmt.Errorf("failed to add order: %w", err)
	}

	return nil
}

func (ls *GophermartLoyaltyService) GetUserOrders(ctx context.Context, user market.User) ([]market.Order, error) {

	orders, err := ls.Storage.GetOrdersByUserID(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}

	if len(orders) == 0 {
		return []market.Order{}, market.ErrNoOrders
	}

	return orders, nil
}

func (ls *GophermartLoyaltyService) GetUserBalance(ctx context.Context, user market.User) (*market.Balance, error) {

	balance, err := ls.Storage.GetBalanceByUserID(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to get user balance: %w", err)
	}

	return balance, nil
}

func (ls *GophermartLoyaltyService) GetUserWithdrawals(ctx context.Context, user market.User) ([]market.Withdrawal, error) {

	withdrawals, err := ls.Storage.GetWithdrawalsByUserID(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to get user withdrawals: %w", err)
	}

	if len(withdrawals) == 0 {
		return nil, market.ErrNoWithdrawals
	}

	return withdrawals, nil
}

func (ls *GophermartLoyaltyService) ExecuteWithdrawal(ctx context.Context, withdrawal market.Withdrawal) error {

	err := market.OrderNumberLuhnCheck(withdrawal.OrderNumber)
	if err != nil {
		return fmt.Errorf("failed to execute withdrawal: %w", err)
	}

	err = ls.Storage.AddWithdrawalWithBalanceCheck(ctx, withdrawal)
	if err != nil {
		return fmt.Errorf("failed to add withdrawal: %w", err)
	}

	return nil
}

func (ls *GophermartLoyaltyService) getUnprocessedOrders(ctx context.Context) ([]market.Order, error) {

	nonFinalStatuses := market.NonFinalStatuses()

	orders, err := ls.Storage.GetOrdersByStatuses(ctx, nonFinalStatuses)
	if err != nil {
		return nil, fmt.Errorf("failed to get unprocessed orders: %w", err)
	}

	return orders, nil
}

func (ls *GophermartLoyaltyService) updateOrderAccrual(ctx context.Context, orderAccrual market.OrderAccrual) error {

	err := ls.Storage.UpdateOrderAccrual(ctx, orderAccrual)
	if err != nil {
		return fmt.Errorf("failed to update order accrual: %w", err)
	}

	return nil
}
