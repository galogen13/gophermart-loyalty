package handlers

import (
	"context"
	"net/http"
)

type LoyaltyService interface {
	RegisterUser(ctx context.Context, login, pass string) error
}

func Empty(ls LoyaltyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.WriteHeader(http.StatusOK)

	}
}
