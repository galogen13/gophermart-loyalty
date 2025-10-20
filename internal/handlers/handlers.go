package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/galogen13/gophermart-loyalty/internal/auth/password"
	"github.com/galogen13/gophermart-loyalty/internal/logger"
	"github.com/galogen13/gophermart-loyalty/internal/service/market"
	"go.uber.org/zap"
)

type LoyaltyService interface {
	RegisterUser(ctx context.Context, user *market.User) error
	LoginUser(ctx context.Context, user *market.User) error
	AuthService
}

type AuthService interface {
	SetTokenInResponseCookie(w http.ResponseWriter, user *market.User) error
	ValidateTokenInRequest(r *http.Request) (context.Context, error)
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

		// hashedPassword, err := password.HashPassword(user.Password)
		// if err != nil {
		// 	logger.Log.Error("Error processing password", zap.Error(err))
		// 	w.WriteHeader(http.StatusInternalServerError)
		// 	return
		// }
		//user.Password = hashedPassword

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

func Empty(ls LoyaltyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.WriteHeader(http.StatusOK)

	}
}
