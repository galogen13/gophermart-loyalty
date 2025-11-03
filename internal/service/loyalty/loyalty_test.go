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

	var userId1 int64 = 1
	order := market.Order{Number: "34", UserID: &userId1, Status: market.OrderStatusNew}
	existedOrderByNumber := market.Order{ID: 1, Number: order.Number, UserID: order.UserID}

	var userId2 int64 = 2
	orderByOtherUser := market.Order{ID: 2, Number: order.Number, UserID: &userId2}

	badOrder1 := market.Order{ID: 2, Number: "11", UserID: &userId1} // плохой номер
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

// func TestGophermartLoyaltyService_GetUserOrders(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	mockStorage := mock.NewMockStorage(ctrl)

// 	config := &config.GophermartConfig{}
// 	accrualService := &accrual.AccrualService{}
// 	authService := &auth.JWTAuthService{}

// 	var id int64 = 1
// 	user := &market.User{Login: "Test", Password: "test", ID: &id}
// 	orders := []market.Order{{ID: 1, Status: market.OrderStatusNew, Accrual: 100, UserID: &id}}
// 	noOrders := []market.Order{}

// 	tests := []struct {
// 		name      string
// 		setup     func()
// 		ordersLen int
// 		wantErr   bool
// 		errorIs   error
// 	}{
// 		{
// 			name:      "Есть ордера",
// 			setup:     func() { mockStorage.EXPECT().GetOrdersByUserID(gomock.Any(), user).Return(orders, nil) },
// 			ordersLen: len(orders),
// 			wantErr:   false,
// 			errorIs:   nil,
// 		},

// 		{
// 			name: "Нет ордеров",
// 			setup: func() {
// 				mockStorage.EXPECT().GetOrdersByUserID(gomock.Any(), user).Return(noOrders, market.ErrNoOrders)
// 			},
// 			ordersLen: len(noOrders),
// 			wantErr:   true,
// 			errorIs:   market.ErrNoOrders,
// 		},

// 		{
// 			name: "Неизвестная ошибка",
// 			setup: func() {
// 				mockStorage.EXPECT().GetOrdersByUserID(gomock.Any(), user).Return(noOrders, errors.New("unexpected error"))
// 			},
// 			ordersLen: len(noOrders),
// 			wantErr:   true,
// 			errorIs:   nil,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			if tt.setup != nil {
// 				tt.setup()
// 			}

// 			ls := NewGophermartLoyaltyService(config, mockStorage, accrualService, authService)
// 			gotOrders, gotErr := ls.GetUserOrders(t.Context(), user)

// 			assert.Equal(t, tt.ordersLen, len(gotOrders))

// 			if tt.wantErr {
// 				assert.Error(t, gotErr)
// 				if tt.errorIs != nil {
// 					assert.ErrorIs(t, gotErr, tt.errorIs)
// 				}
// 			} else {
// 				assert.NoError(t, gotErr)
// 			}

// 		})
// 	}

// }
