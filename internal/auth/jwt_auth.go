package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/galogen13/gophermart-loyalty/internal/handlers"
	"github.com/galogen13/gophermart-loyalty/internal/logger"
	"github.com/galogen13/gophermart-loyalty/internal/service/market"
	"github.com/golang-jwt/jwt/v5"

	"go.uber.org/zap"
)

const (
	cookieTokenName            = "jwt_token"
	userClaimsKey   contextKey = "user_claims"
)

var (
	ErrAuthorizationTokenRequired error = errors.New("authorization token required")
	ErrInvalidToken               error = errors.New("invalid token")
)

type contextKey string

type JWTAuthService struct {
	jwtSecret string
}

func NewJWTAuthService(secret string) *JWTAuthService {
	return &JWTAuthService{jwtSecret: secret}
}

func (s *JWTAuthService) SetTokenInResponseCookie(w http.ResponseWriter, user *market.User) error {
	token, err := s.generateToken(user)
	if err != nil {
		return fmt.Errorf("error generating token: %w", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieTokenName,
		Value:    token,
		Path:     "/",
		MaxAge:   24 * 3600,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	})

	return nil
}

func (s *JWTAuthService) generateToken(user *market.User) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)

	claims := &Claims{
		UserID:    *user.ID,
		UserLogin: user.Login,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   strconv.FormatInt(*user.ID, 10),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

type Claims struct {
	UserID    int64  //`json:"user_id"`
	UserLogin string //`json:"username"`
	jwt.RegisteredClaims
}

func (s *JWTAuthService) CheckTokenInRequest(r *http.Request) (context.Context, error) {
	tokenString := s.extractTokenFromRequest(r)

	if tokenString == "" {
		return nil, ErrAuthorizationTokenRequired
	}

	claims, err := s.validateToken(tokenString)
	if err != nil {
		return nil, ErrInvalidToken
	}

	ctx := context.WithValue(r.Context(), userClaimsKey, claims)

	return ctx, nil
}

func (s *JWTAuthService) extractTokenFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(cookieTokenName)
	if err == nil {
		return cookie.Value
	}

	return ""
}

func (s *JWTAuthService) validateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: unexpected signing method: %v", ErrInvalidToken, token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func RequireAuth(ls handlers.LoyaltyService, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, err := ls.CheckTokenInRequest(r)
		if err != nil {
			if errors.Is(err, ErrAuthorizationTokenRequired) || errors.Is(err, ErrInvalidToken) {
				logger.Log.Info("Error validating token", zap.Error(err))
				w.WriteHeader(http.StatusUnauthorized)
			} else {
				logger.Log.Error("Unexpected error", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
