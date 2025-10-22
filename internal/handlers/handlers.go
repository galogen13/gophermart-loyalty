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
	GetUserOrders(ctx context.Context, user *market.User) ([]*market.Order, error)
	AuthService
}

type AuthService interface {
	SetTokenInResponseCookie(w http.ResponseWriter, user *market.User) error
	ValidateTokenInRequest(r *http.Request) (context.Context, error)
	GetUserIDFromContext(ctx context.Context) (int64, error)
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

		userID, err := ls.GetUserIDFromContext(ctx)
		if err != nil {
			logger.Log.Error("Error getting ID from context", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		order := &market.Order{
			Number: string(b),
			UserID: &userID}

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

		userID, err := ls.GetUserIDFromContext(ctx)
		if err != nil {
			logger.Log.Error("Error getting ID from context", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		user := &market.User{
			ID: &userID}

		orders, err := ls.GetUserOrders(ctx, user)
		if err != nil {
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

func Empty(ls LoyaltyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.WriteHeader(http.StatusOK)

	}
}
