package loyalty

import (
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

	user := &market.User{Login: "Test", Password: "test"}

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
		errorIs error
	}{
		{
			name:    "Успешная регистрация",
			setup:   func() { mockStorage.EXPECT().AddUser(gomock.Any(), user).Return(nil) },
			wantErr: false,
			errorIs: nil,
		},

		{
			name:    "Логин уже используется",
			setup:   func() { mockStorage.EXPECT().AddUser(gomock.Any(), user).Return(market.ErrUserLoginAlreadyInUse) },
			wantErr: true,
			errorIs: market.ErrUserLoginAlreadyInUse,
		},

		{
			name:    "Неизвестная ошибка",
			setup:   func() { mockStorage.EXPECT().AddUser(gomock.Any(), user).Return(errors.New("unexpected error")) },
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
			gotErr := ls.RegisterUser(t.Context(), user)

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

func TestGophermartLoyaltyService_LoginUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStorage := mock.NewMockStorage(ctrl)

	config := &config.GophermartConfig{}
	accrualService := &accrual.AccrualService{}
	authService := &auth.JWTAuthService{}

	user := &market.User{Login: "Test", Password: "test"}

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
		errorIs error
	}{
		{
			name:    "Успешная авторизация",
			setup:   func() { mockStorage.EXPECT().GetUserByLogin(gomock.Any(), user).Return(nil) },
			wantErr: false,
			errorIs: nil,
		},

		{
			name: "Пользователь не существует",
			setup: func() {
				mockStorage.EXPECT().GetUserByLogin(gomock.Any(), user).Return(market.ErrUserNotExists)
			},
			wantErr: true,
			errorIs: market.ErrUserNotExists,
		},

		{
			name:    "Неизвестная ошибка",
			setup:   func() { mockStorage.EXPECT().GetUserByLogin(gomock.Any(), user).Return(errors.New("unexpected error")) },
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
			gotErr := ls.LoginUser(t.Context(), user)

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

func TestGophermartLoyaltyService_AddOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStorage := mock.NewMockStorage(ctrl)

	config := &config.GophermartConfig{}
	accrualService := &accrual.AccrualService{}
	authService := &auth.JWTAuthService{}

	var userId int64 = 1
	existedGoodOrder := market.Order{Number: "34"}
	goodOrder := market.Order{ID: 1, Number: existedGoodOrder.Number, UserID: &userId}

	badOrder1 := market.Order{ID: 2, Number: "11", UserID: &userId} // плохой номер
	badOrder2 := market.Order{ID: 3, Number: "34"}                  // не указан id пользователя

	tests := []struct {
		name    string
		setup   func()
		order   *market.Order
		wantErr bool
		errorIs error
	}{
		{
			name: "Успешное добавление",
			setup: func() {
				mockStorage.EXPECT().GetOrderByNumber(gomock.Any(), &existedGoodOrder).Return(market.ErrOrderNotExists)
				mockStorage.EXPECT().AddOrder(gomock.Any(), &goodOrder).Return(nil)
			},
			order:   &goodOrder,
			wantErr: false,
			errorIs: nil,
		},

		{
			name:    "Некорректный номер заказа",
			order:   &badOrder1,
			wantErr: true,
			errorIs: market.ErrOrderIncorrectNumber,
		},

		{
			name:    "Не указан id пользователя в заказе",
			order:   &badOrder2,
			wantErr: true,
			errorIs: nil,
		},

		{
			name: "Заказ принадлежит другому пользоателю",
			setup: func() {
				mockStorage.EXPECT().GetOrderByNumber(gomock.Any(), &existedGoodOrder).Return(nil)
			},
			order:   &goodOrder,
			wantErr: true,
			errorIs: market.ErrOrderBelongsToAnotherUser,
		},

		{
			name: "Заказ уже существует",
			setup: func() {
				mockStorage.EXPECT().GetOrderByNumber(gomock.Any(), &existedGoodOrder).Return(nil)
			},
			order:   &goodOrder,
			wantErr: true,
			errorIs: market.ErrOrderAlreadyExists,
		},

		{
			name: "Неизвестная ошибка",
			setup: func() {
				mockStorage.EXPECT().AddOrder(gomock.Any(), &existedGoodOrder).Return(errors.New("unexpected error"))
			},
			order:   &goodOrder,
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
	user := &market.User{Login: "Test", Password: "test", ID: &id}
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
