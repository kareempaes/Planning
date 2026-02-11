package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey string

const userIDKey contextKey = "userID"

// AuthMiddleware returns chi-compatible middleware that validates JWT Bearer tokens.
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				writeJSON(w, http.StatusUnauthorized, ErrorBody{
					Error: ErrorDetail{Code: "unauthorized", Message: "missing or invalid authorization header"},
				})
				return
			}
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				writeJSON(w, http.StatusUnauthorized, ErrorBody{
					Error: ErrorDetail{Code: "unauthorized", Message: "invalid or expired token"},
				})
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				writeJSON(w, http.StatusUnauthorized, ErrorBody{
					Error: ErrorDetail{Code: "unauthorized", Message: "invalid token claims"},
				})
				return
			}

			sub, err := claims.GetSubject()
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, ErrorBody{
					Error: ErrorDetail{Code: "unauthorized", Message: "missing subject claim"},
				})
				return
			}

			userID, err := uuid.Parse(sub)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, ErrorBody{
					Error: ErrorDetail{Code: "unauthorized", Message: "invalid user ID in token"},
				})
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext extracts the authenticated user's UUID from the request context.
func UserIDFromContext(ctx context.Context) uuid.UUID {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	if !ok {
		panic("handler: UserIDFromContext called without AuthMiddleware")
	}
	return id
}
