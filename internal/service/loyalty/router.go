package loyalty

import (
	"net/http"

	"github.com/galogen13/gophermart-loyalty/internal/handlers"
	"github.com/galogen13/gophermart-loyalty/internal/logger"
	"github.com/go-chi/chi/v5"
)

// const (
// 	reqContentTypeTextPlain  = "text/plain"
// 	respContentTypeTextPlain = "text/plain; charset=utf-8"
// )

func loyaltyRouter(ls handlers.LoyaltyService) *chi.Mux {
	r := chi.NewRouter()

	r.NotFound(logger.RequestLogger(notFoundHandler()))
	r.MethodNotAllowed(logger.RequestLogger(methodNotAllowedHandler()))

	r.Route("/api/user", func(r chi.Router) {

		r.Route("/register", func(r chi.Router) {
			r.Post("/", logger.RequestLogger(
				handlers.Empty(ls)))
		})

		r.Route("/login", func(r chi.Router) {
			r.Post("/", logger.RequestLogger(
				handlers.Empty(ls)))
		})

		r.Route("/orders", func(r chi.Router) {
			r.Post("/", logger.RequestLogger(
				handlers.Empty(ls)))
			r.Get("/", logger.RequestLogger(
				handlers.Empty(ls)))
		})

		r.Route("/balance", func(r chi.Router) {
			r.Get("/", logger.RequestLogger(
				handlers.Empty(ls)))
			r.Route("/withdraw", func(r chi.Router) {
				r.Post("/", logger.RequestLogger(
					handlers.Empty(ls)))
			})
		})

		r.Route("/withdrawals", func(r chi.Router) {
			r.Get("/", logger.RequestLogger(
				handlers.Empty(ls)))
		})

	})

	return r
}

func notFoundHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// w.Header().Set("Content-Type", respContentTypeTextPlain)
		w.WriteHeader(http.StatusNotFound)
	})
}

func methodNotAllowedHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// w.Header().Set("Content-Type", respContentTypeTextPlain)
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
}
