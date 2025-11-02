package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/galogen13/gophermart-loyalty/internal/auth/password"
	"github.com/galogen13/gophermart-loyalty/internal/logger"
	"github.com/galogen13/gophermart-loyalty/internal/service/market"
	"go.uber.org/zap"
)

type LoyaltyService interface {
	RegisterUser(ctx context.Context, user *market.User) error
	LoginUser(ctx context.Context, user *market.User) error
	AddOrder(ctx context.Context, order *market.Order) error
	GetUserOrders(ctx context.Context, user *market.User) ([]market.Order, error)
	GetUserBalance(ctx context.Context, user *market.User) (*market.Balance, error)
	GetUserWithdrawals(ctx context.Context, user *market.User) ([]market.Withdrawal, error)
	ExecuteWithdrawal(ctx context.Context, user *market.User, withdrawal *market.Withdrawal) error
	AuthService
}

type AuthService interface {
	SetTokenInResponseCookie(w http.ResponseWriter, user *market.User) error
	ValidateTokenInRequest(r *http.Request) (context.Context, error)
	GetUserFromContext(ctx context.Context) (*market.User, error)
}

func RegisterUserHandler(ls LoyaltyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		user := &market.User{}
		if err := json.NewDecoder(r.Body).Decode(user); err != nil {
			logger.Log.Error("JSON decoding error", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		hashedPassword, err := password.HashPassword(user.Password)
		if err != nil {
			logger.Log.Error("Error processing password", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		user.Password = hashedPassword

		err = ls.RegisterUser(ctx, user)
		if err != nil {
			logger.Log.Info("Error register user", zap.Error(err))
			if errors.Is(err, market.ErrUserLoginAlreadyInUse) {
				w.WriteHeader(http.StatusConflict)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		err = ls.SetTokenInResponseCookie(w, user)
		if err != nil {
			logger.Log.Error("Error setting token in cookie", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, err = http.NoBody.WriteTo(w)
		if err != nil {
			logger.Log.Error("Error writing body", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	}
}

func LoginUserHandler(ls LoyaltyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		user := &market.User{}
		if err := json.NewDecoder(r.Body).Decode(user); err != nil {
			logger.Log.Error("JSON decoding error", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		incomingPass := user.Password

		err := ls.LoginUser(ctx, user)
		if err != nil {
			logger.Log.Info("Error login user", zap.Error(err))
			if errors.Is(err, market.ErrUserNotExists) {
				w.WriteHeader(http.StatusUnauthorized)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		if !password.CheckPassword(incomingPass, user.Password) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		err = ls.SetTokenInResponseCookie(w, user)
		if err != nil {
			logger.Log.Error("Error setting token in cookie", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, err = http.NoBody.WriteTo(w)
		if err != nil {
			logger.Log.Error("Error writing body", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	}
}

func AddOrderHandler(ls LoyaltyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		b, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Log.Error("Error reading body", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		ctx := r.Context()

		user, err := ls.GetUserFromContext(ctx)
		if err != nil {
			logger.Log.Error("Error getting ID from context", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		order := &market.Order{
			Number: string(b),
			UserID: user.ID}

		err = ls.AddOrder(ctx, order)
		if err != nil {
			if errors.Is(err, market.ErrOrderAlreadyExists) {
				logger.Log.Info("Error adding order", zap.Error(err))
				w.WriteHeader(http.StatusOK)
				return
			}

			if errors.Is(err, market.ErrOrderBelongsToAnotherUser) {
				logger.Log.Info("Error adding order", zap.Error(err))
				w.WriteHeader(http.StatusConflict)
				return
			}

			if errors.Is(err, market.ErrOrderIncorrectNumber) {
				logger.Log.Info("Error adding order", zap.Error(err))
				w.WriteHeader(http.StatusUnprocessableEntity)
				return
			}

			logger.Log.Error("Error adding order", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)

	}
}

func GetUserOrdersHandler(ls LoyaltyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		user, err := ls.GetUserFromContext(ctx)
		if err != nil {
			logger.Log.Error("Error getting ID from context", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		orders, err := ls.GetUserOrders(ctx, user)
		if err != nil {
			if errors.Is(err, market.ErrNoOrders) {
				logger.Log.Info("Error getting users orders", zap.Error(err))
				w.WriteHeader(http.StatusNoContent)
				return
			}
			logger.Log.Error("Error adding order", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(orders); err != nil {
			logger.Log.Error("Error encoding user orders", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	}
}

func GetBalanceHandler(ls LoyaltyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		user, err := ls.GetUserFromContext(ctx)
		if err != nil {
			logger.Log.Error("Error getting user from context", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		balance, err := ls.GetUserBalance(ctx, user)
		if err != nil {
			logger.Log.Error("Error getting user balance", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(balance); err != nil {
			logger.Log.Error("Error encoding users balance", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	}
}

func GetWithdrawalsHandler(ls LoyaltyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		user, err := ls.GetUserFromContext(ctx)
		if err != nil {
			logger.Log.Error("Error getting user from context", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		withdrawals, err := ls.GetUserWithdrawals(ctx, user)
		if err != nil {
			if errors.Is(err, market.ErrNoWithdrawals) {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			logger.Log.Error("Error getting users withdrawals", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(withdrawals); err != nil {
			logger.Log.Error("Error encoding users balance", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	}
}

func ExecuteWithdrawalHandler(ls LoyaltyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		user, err := ls.GetUserFromContext(ctx)
		if err != nil {
			logger.Log.Error("Error getting user from context", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		withdrawal := &market.Withdrawal{}
		if err := json.NewDecoder(r.Body).Decode(withdrawal); err != nil {
			logger.Log.Error("JSON decoding error", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		withdrawal.UserID = user.ID

		err = ls.ExecuteWithdrawal(ctx, user, withdrawal)
		if err != nil {
			if errors.Is(err, market.ErrOrderIncorrectNumber) {
				logger.Log.Info("Error executing withdrawals", zap.Error(err))
				w.WriteHeader(http.StatusUnprocessableEntity)
				return
			}

			if errors.Is(err, market.ErrWithdrawalInsufficientFunds) {
				logger.Log.Info("Error executing withdrawals", zap.Error(err))
				w.WriteHeader(http.StatusPaymentRequired)
				return
			}

			logger.Log.Error("Error executing withdrawals", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)

		_, err = http.NoBody.WriteTo(w)
		if err != nil {
			logger.Log.Error("Error writing body", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}
