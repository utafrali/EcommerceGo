package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type contextKeyType string

const (
	userIDKey contextKeyType = "user_id"
	roleKey   contextKeyType = "role"
)

// Claims represents the JWT claims extracted by the auth middleware.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

// TokenValidator is a function that validates a JWT token and returns claims.
// This allows the gateway/service to inject its own validation logic.
type TokenValidator func(token string) (*Claims, error)

// Auth middleware validates JWT tokens and injects user claims into context.
func Auth(validate TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeAuthError(w, "missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				writeAuthError(w, "invalid authorization header format")
				return
			}

			claims, err := validate(parts[1])
			if err != nil {
				writeAuthError(w, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			ctx = context.WithValue(ctx, roleKey, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole middleware checks that the authenticated user has the required role.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	roleSet := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		roleSet[r] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := RoleFromContext(r.Context())
			if _, ok := roleSet[role]; !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"code":    "FORBIDDEN",
					"message": "insufficient permissions",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// UserIDFromContext extracts the user ID from the request context.
func UserIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(userIDKey).(string); ok {
		return id
	}
	return ""
}

// RoleFromContext extracts the user role from the request context.
func RoleFromContext(ctx context.Context) string {
	if role, ok := ctx.Value(roleKey).(string); ok {
		return role
	}
	return ""
}

func writeAuthError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":    "UNAUTHORIZED",
		"message": message,
	})
}
