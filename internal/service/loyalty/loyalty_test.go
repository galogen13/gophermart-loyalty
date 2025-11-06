package loyalty

import (
	"context"
	"errors"
	"testing"

	"github.com/galogen13/gophermart-loyalty/internal/auth"
	"github.com/galogen13/gophermart-loyalty/internal/config"
	"github.com/galogen13/gophermart-loyalty/internal/service/accrual"
	mock "github.com/galogen13/gophermart-loyalty/internal/service/loyalty/mocks"
	"github.com/galogen13/gophermart-loyalty/internal/service/market"
	"github.com/stretchr/testify/assert"

	"go.uber.org/mock/gomock"
)

func TestGophermartLoyaltyService_RegisterUser(t *testing.T) {

	ctrl := gomock.NewController(t)
	mockStorage := mock.NewMockStorage(ctrl)

	config := &config.GophermartConfig{}
	accrualService := &accrual.AccrualService{}
	authService := &auth.JWTAuthService{}

	testUserCred := market.UserCredentials{Login: "Test", Password: "test"}
	testUser := market.User{UserCredentials: testUserCred}

	type result struct {
		userIsNotNil bool
		wantErr      bool
		errorIs      error
	}

	tests := []struct {
		name     string
		setup    func()
		userCred market.UserCredentials
		result   result
	}{
		{
			name:     "Успешная регистрация",
			setup:    func() { mockStorage.EXPECT().AddUser(gomock.Any(), testUser).Return(&testUser, nil) },
			userCred: testUserCred,
			result: result{
				userIsNotNil: true,
				wantErr:      false,
				errorIs:      nil},
		},

		{
			name: "Логин уже используется",
			setup: func() {
				mockStorage.EXPECT().AddUser(gomock.Any(), testUser).Return(nil, market.ErrUserLoginAlreadyInUse)
			},
			userCred: testUserCred,
			result: result{
				userIsNotNil: false,
				wantErr:      true,
				errorIs:      market.ErrUserLoginAlreadyInUse},
		},

		{
			name: "Неизвестная ошибка",
			setup: func() {
				mockStorage.EXPECT().AddUser(gomock.Any(), testUser).Return(nil, errors.New("unexpected error"))
			},
			userCred: testUserCred,
			result: result{
				userIsNotNil: false,
				wantErr:      true,
				errorIs:      nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.setup != nil {
				tt.setup()
			}

			ls := NewGophermartLoyaltyService(config, mockStorage, accrualService, authService)
			gotUser, gotErr := ls.RegisterUser(t.Context(), tt.userCred)

			if tt.result.userIsNotNil {
				assert.NotNil(t, gotUser)
			} else {
				assert.Nil(t, gotUser)
			}

			if tt.result.wantErr {
				assert.Error(t, gotErr)
				if tt.result.errorIs != nil {
					assert.ErrorIs(t, gotErr, tt.result.errorIs)
				}
			} else {
				assert.NoError(t, gotErr)
			}

		})
	}
}

func TestGophermartLoyaltyService_LoginUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStorage := mock.NewMockStorage(ctrl)

	config := &config.GophermartConfig{}
	accrualService := &accrual.AccrualService{}
	authService := &auth.JWTAuthService{}

	testUserCred := market.UserCredentials{Login: "Test", Password: "test"}
	testUser := market.User{UserCredentials: testUserCred}

	type result struct {
		userIsNotNil bool
		wantErr      bool
		errorIs      error
	}

	tests := []struct {
		name     string
		setup    func()
		userCred market.UserCredentials
		result   result
	}{
		{
			name:     "Успешная авторизация",
			setup:    func() { mockStorage.EXPECT().GetUserByLogin(gomock.Any(), testUser).Return(&testUser, nil) },
			userCred: testUserCred,
			result: result{
				userIsNotNil: true,
				wantErr:      false,
				errorIs:      nil},
		},

		{
			name: "Пользователь не существует",
			setup: func() {
				mockStorage.EXPECT().GetUserByLogin(gomock.Any(), testUser).Return(nil, market.ErrUserNotExists)
			},
			userCred: testUserCred,
			result: result{
				userIsNotNil: false,
				wantErr:      true,
				errorIs:      market.ErrUserNotExists},
		},

		{
			name: "Неизвестная ошибка",
			setup: func() {
				mockStorage.EXPECT().GetUserByLogin(gomock.Any(), testUser).Return(nil, errors.New("unexpected error"))
			},
			userCred: testUserCred,
			result: result{
				userIsNotNil: false,
				wantErr:      true,
				errorIs:      nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			ls := NewGophermartLoyaltyService(config, mockStorage, accrualService, authService)
			gotUser, gotErr := ls.LoginUser(t.Context(), tt.userCred)

			if tt.result.userIsNotNil {
				assert.NotNil(t, gotUser)
			} else {
				assert.Nil(t, gotUser)
			}

			if tt.result.wantErr {
				assert.Error(t, gotErr)
				if tt.result.errorIs != nil {
					assert.ErrorIs(t, gotErr, tt.result.errorIs)
				}
			} else {
				assert.NoError(t, gotErr)
			}

		})
	}
}

func TestGophermartLoyaltyService_AddOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStorage := mock.NewMockStorage(ctrl)

	config := &config.GophermartConfig{}
	accrualService := &accrual.AccrualService{}
	authService := &auth.JWTAuthService{}

	var userID1 int64 = 1
	order := market.Order{Number: "34", UserID: &userID1, Status: market.OrderStatusNew}
	existedOrderByNumber := market.Order{ID: 1, Number: order.Number, UserID: order.UserID, Status: order.Status}

	var userID2 int64 = 2
	orderByOtherUser := market.Order{ID: 2, Number: order.Number, UserID: &userID2}

	badOrder1 := market.Order{ID: 2, Number: "11", UserID: &userID1} // плохой номер
	badOrder2 := market.Order{ID: 3, Number: "34"}                   // не указан id пользователя

	tests := []struct {
		name    string
		setup   func()
		order   market.Order
		wantErr bool
		errorIs error
	}{
		{
			name: "Успешное добавление",
			setup: func() {
				mockStorage.EXPECT().GetOrderByNumber(gomock.Any(), order).Return(nil, market.ErrOrderNotExists)
				mockStorage.EXPECT().AddOrder(gomock.Any(), order).Return(nil)
			},
			order:   order,
			wantErr: false,
			errorIs: nil,
		},

		{
			name:    "Некорректный номер заказа",
			order:   badOrder1,
			wantErr: true,
			errorIs: market.ErrOrderIncorrectNumber,
		},

		{
			name:    "Не указан id пользователя в заказе",
			order:   badOrder2,
			wantErr: true,
			errorIs: nil,
		},

		{
			name: "Заказ принадлежит другому пользоателю",
			setup: func() {
				mockStorage.EXPECT().GetOrderByNumber(gomock.Any(), order).Return(&orderByOtherUser, nil)
			},
			order:   order,
			wantErr: true,
			errorIs: market.ErrOrderBelongsToAnotherUser,
		},

		{
			name: "Заказ уже существует",
			setup: func() {
				mockStorage.EXPECT().GetOrderByNumber(gomock.Any(), order).Return(&existedOrderByNumber, nil)
			},
			order:   order,
			wantErr: true,
			errorIs: market.ErrOrderAlreadyExists,
		},

		{
			name: "Неизвестная ошибка",
			setup: func() {
				mockStorage.EXPECT().GetOrderByNumber(gomock.Any(), order).Return(nil, market.ErrOrderNotExists)
				mockStorage.EXPECT().AddOrder(gomock.Any(), order).Return(errors.New("unexpected error"))
			},
			order:   order,
			wantErr: true,
			errorIs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			ls := NewGophermartLoyaltyService(config, mockStorage, accrualService, authService)
			gotErr := ls.AddOrder(t.Context(), tt.order)

			if tt.wantErr {
				assert.Error(t, gotErr)
				if tt.errorIs != nil {
					assert.ErrorIs(t, gotErr, tt.errorIs)
				}
			} else {
				assert.NoError(t, gotErr)
			}
		})
	}
}

func TestGophermartLoyaltyService_GetUserOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStorage := mock.NewMockStorage(ctrl)

	config := &config.GophermartConfig{}
	accrualService := &accrual.AccrualService{}
	authService := &auth.JWTAuthService{}

	var id int64 = 1
	user := market.User{ID: &id}
	orders := []market.Order{{ID: 1, Status: market.OrderStatusNew, Accrual: 100, UserID: &id}}
	noOrders := []market.Order{}

	tests := []struct {
		name      string
		setup     func()
		ordersLen int
		wantErr   bool
		errorIs   error
	}{
		{
			name:      "Есть ордера",
			setup:     func() { mockStorage.EXPECT().GetOrdersByUserID(gomock.Any(), user).Return(orders, nil) },
			ordersLen: len(orders),
			wantErr:   false,
			errorIs:   nil,
		},

		{
			name: "Нет ордеров",
			setup: func() {
				mockStorage.EXPECT().GetOrdersByUserID(gomock.Any(), user).Return(noOrders, market.ErrNoOrders)
			},
			ordersLen: len(noOrders),
			wantErr:   true,
			errorIs:   market.ErrNoOrders,
		},

		{
			name: "Неизвестная ошибка",
			setup: func() {
				mockStorage.EXPECT().GetOrdersByUserID(gomock.Any(), user).Return(noOrders, errors.New("unexpected error"))
			},
			ordersLen: len(noOrders),
			wantErr:   true,
			errorIs:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			ls := NewGophermartLoyaltyService(config, mockStorage, accrualService, authService)
			gotOrders, gotErr := ls.GetUserOrders(t.Context(), user)

			assert.Equal(t, tt.ordersLen, len(gotOrders))

			if tt.wantErr {
				assert.Error(t, gotErr)
				if tt.errorIs != nil {
					assert.ErrorIs(t, gotErr, tt.errorIs)
				}
			} else {
				assert.NoError(t, gotErr)
			}

		})
	}

}

func TestGophermartLoyaltyService_GetUserBalance(t *testing.T) {

	ctrl := gomock.NewController(t)
	mockStorage := mock.NewMockStorage(ctrl)

	config := &config.GophermartConfig{}
	accrualService := &accrual.AccrualService{}
	authService := &auth.JWTAuthService{}

	var id int64 = 1
	user := market.User{ID: &id}
	balance := &market.Balance{Current: 100, Withdrawn: 50}
	noBalance := &market.Balance{Current: 0, Withdrawn: 0}

	tests := []struct {
		name        string
		setup       func()
		user        market.User
		wantBalance *market.Balance
		wantErr     bool
		errorIs     error
	}{
		{
			name: "Есть баланс",
			user: user,
			setup: func() {
				mockStorage.EXPECT().GetBalanceByUserID(gomock.Any(), user).Return(balance, nil)
			},
			wantBalance: balance,
			wantErr:     false,
			errorIs:     nil,
		},
		{
			name: "Нет баланса",
			user: user,
			setup: func() {
				mockStorage.EXPECT().GetBalanceByUserID(gomock.Any(), user).Return(noBalance, nil)
			},
			wantBalance: noBalance,
			wantErr:     false,
			errorIs:     nil,
		},
		{
			name: "Неожиданная ошибка",
			user: user,
			setup: func() {
				mockStorage.EXPECT().GetBalanceByUserID(gomock.Any(), user).Return(nil, errors.New("unexpected error"))
			},
			wantBalance: nil,
			wantErr:     true,
			errorIs:     nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.setup != nil {
				tt.setup()
			}

			ls := NewGophermartLoyaltyService(config, mockStorage, accrualService, authService)
			gotBalance, gotErr := ls.GetUserBalance(context.Background(), tt.user)

			assert.Equal(t, tt.wantBalance, gotBalance)

			if tt.wantErr {
				assert.Error(t, gotErr)
				if tt.errorIs != nil {
					assert.ErrorIs(t, gotErr, tt.errorIs)
				}
			} else {
				assert.NoError(t, gotErr)
			}

		})
	}
}

func TestGophermartLoyaltyService_GetUserWithdrawals(t *testing.T) {

	ctrl := gomock.NewController(t)
	mockStorage := mock.NewMockStorage(ctrl)

	config := &config.GophermartConfig{}
	accrualService := &accrual.AccrualService{}
	authService := &auth.JWTAuthService{}

	var id int64 = 1
	user := market.User{ID: &id}
	withdrawals := []market.Withdrawal{{OrderNumber: "34", UserID: &id, Sum: 50}}
	noWithdrawals := []market.Withdrawal{}

	tests := []struct {
		name            string
		setup           func()
		user            market.User
		wantWithdrawals []market.Withdrawal
		wantErr         bool
		errorIs         error
	}{
		{
			name: "Есть списания",
			user: user,
			setup: func() {
				mockStorage.EXPECT().GetWithdrawalsByUserID(gomock.Any(), user).Return(withdrawals, nil)
			},
			wantWithdrawals: withdrawals,
			wantErr:         false,
			errorIs:         nil,
		},
		{
			name: "Нет списаний",
			user: user,
			setup: func() {
				mockStorage.EXPECT().GetWithdrawalsByUserID(gomock.Any(), user).Return(noWithdrawals, nil)
			},
			wantWithdrawals: nil,
			wantErr:         true,
			errorIs:         market.ErrNoWithdrawals,
		},
		{
			name: "Неожиданная ошибка",
			user: user,
			setup: func() {
				mockStorage.EXPECT().GetWithdrawalsByUserID(gomock.Any(), user).Return(noWithdrawals, errors.New("unexpected error"))
			},
			wantWithdrawals: nil,
			wantErr:         true,
			errorIs:         nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.setup != nil {
				tt.setup()
			}

			ls := NewGophermartLoyaltyService(config, mockStorage, accrualService, authService)
			gotWithdrawals, gotErr := ls.GetUserWithdrawals(context.Background(), tt.user)

			assert.Equal(t, tt.wantWithdrawals, gotWithdrawals)

			if tt.wantErr {
				assert.Error(t, gotErr)
				if tt.errorIs != nil {
					assert.ErrorIs(t, gotErr, tt.errorIs)
				}
			} else {
				assert.NoError(t, gotErr)
			}

		})
	}

}

func TestGophermartLoyaltyService_ExecuteWithdrawal(t *testing.T) {

	ctrl := gomock.NewController(t)
	mockStorage := mock.NewMockStorage(ctrl)

	config := &config.GophermartConfig{}
	accrualService := &accrual.AccrualService{}
	authService := &auth.JWTAuthService{}

	var id int64 = 1
	withdrawal := market.Withdrawal{OrderNumber: "34", UserID: &id, Sum: 50}

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
		errorIs error
	}{
		{
			name: "Списание выполнено",
			setup: func() {
				mockStorage.EXPECT().AddWithdrawalWithBalanceCheck(gomock.Any(), withdrawal).Return(nil)
			},
			wantErr: false,
			errorIs: nil,
		},
		{
			name: "Недостаточно средств",
			setup: func() {
				mockStorage.EXPECT().AddWithdrawalWithBalanceCheck(gomock.Any(), withdrawal).Return(market.ErrWithdrawalInsufficientFunds)
			},
			wantErr: true,
			errorIs: market.ErrWithdrawalInsufficientFunds,
		},
		{
			name: "Списание уже выполнено",
			setup: func() {
				mockStorage.EXPECT().AddWithdrawalWithBalanceCheck(gomock.Any(), withdrawal).Return(market.ErrWithdrawalAlreadyExists)
			},
			wantErr: true,
			errorIs: market.ErrWithdrawalAlreadyExists,
		},
		{
			name: "Неожиданная ошибка",
			setup: func() {
				mockStorage.EXPECT().AddWithdrawalWithBalanceCheck(gomock.Any(), withdrawal).Return(errors.New("unexpected error"))
			},
			wantErr: true,
			errorIs: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.setup != nil {
				tt.setup()
			}

			ls := NewGophermartLoyaltyService(config, mockStorage, accrualService, authService)
			gotErr := ls.ExecuteWithdrawal(context.Background(), withdrawal)

			if tt.wantErr {
				assert.Error(t, gotErr)
				if tt.errorIs != nil {
					assert.ErrorIs(t, gotErr, tt.errorIs)
				}
			} else {
				assert.NoError(t, gotErr)
			}

		})
	}
}

func TestGophermartLoyaltyService_getUnprocessedOrders(t *testing.T) {

	ctrl := gomock.NewController(t)
	mockStorage := mock.NewMockStorage(ctrl)

	config := &config.GophermartConfig{}
	accrualService := &accrual.AccrualService{}
	authService := &auth.JWTAuthService{}

	var id int64 = 1
	orders := []market.Order{{ID: 1, Number: "34", Status: market.OrderStatusProcessing, UserID: &id}}

	tests := []struct {
		name       string
		setup      func()
		wantOrders []market.Order
		wantErr    bool
		errorIs    error
	}{
		{
			name: "Есть незавершенные ордера",
			setup: func() {
				mockStorage.EXPECT().GetOrdersByStatuses(gomock.Any(), gomock.Any()).Return(orders, nil)
			},
			wantOrders: orders,
			wantErr:    false,
			errorIs:    nil,
		},
		{
			name: "Нет незавершенных ордеров",
			setup: func() {
				mockStorage.EXPECT().GetOrdersByStatuses(gomock.Any(), gomock.Any()).Return([]market.Order{}, nil)
			},
			wantOrders: []market.Order{},
			wantErr:    false,
			errorIs:    nil,
		},
		{
			name: "Неожиданная ошибка",
			setup: func() {
				mockStorage.EXPECT().GetOrdersByStatuses(gomock.Any(), gomock.Any()).Return(nil, errors.New("unexpected error"))
			},
			wantErr: true,
			errorIs: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.setup != nil {
				tt.setup()
			}

			ls := NewGophermartLoyaltyService(config, mockStorage, accrualService, authService)
			got, gotErr := ls.getUnprocessedOrders(context.Background())

			assert.Equal(t, tt.wantOrders, got)

			if tt.wantErr {
				assert.Error(t, gotErr)
				if tt.errorIs != nil {
					assert.ErrorIs(t, gotErr, tt.errorIs)
				}
			} else {
				assert.NoError(t, gotErr)
			}

		})
	}
}

func TestGophermartLoyaltyService_updateOrderAccrual(t *testing.T) {

	ctrl := gomock.NewController(t)
	mockStorage := mock.NewMockStorage(ctrl)

	config := &config.GophermartConfig{}
	accrualService := &accrual.AccrualService{}
	authService := &auth.JWTAuthService{}

	accrual := market.OrderAccrual{Number: "34", Status: market.OrderStatusProcessing, Accrual: 50}

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
		errorIs error
	}{
		{
			name: "Есть незавершенные ордера",
			setup: func() {
				mockStorage.EXPECT().UpdateOrderAccrual(gomock.Any(), gomock.Any()).Return(nil)
			},
			wantErr: false,
			errorIs: nil,
		},
		{
			name: "Неожиданная ошибка",
			setup: func() {
				mockStorage.EXPECT().UpdateOrderAccrual(gomock.Any(), gomock.Any()).Return(errors.New("unexpected error"))
			},
			wantErr: true,
			errorIs: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.setup != nil {
				tt.setup()
			}

			ls := NewGophermartLoyaltyService(config, mockStorage, accrualService, authService)
			gotErr := ls.updateOrderAccrual(context.Background(), accrual)

			if tt.wantErr {
				assert.Error(t, gotErr)
				if tt.errorIs != nil {
					assert.ErrorIs(t, gotErr, tt.errorIs)
				}
			} else {
				assert.NoError(t, gotErr)
			}

		})
	}

}
