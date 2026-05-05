package middleware

import (
	"context"
	"expense-tracker/internal/models"
	"expense-tracker/internal/service"
	"net/http"
	"strings"
)

type contextKey string

const (
	UserIDKey   contextKey = "userID"
	UserRoleKey contextKey = "userRole"
	GroupIDKey  contextKey = "groupID"
)

// AuthMiddleware validates JWT token and adds user info to context.
func AuthMiddleware(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "authorization header missing", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, "invalid authorization format", http.StatusUnauthorized)
				return
			}

			claims, err := authService.ValidateToken(parts[1])
			if err != nil {
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Use comma-ok pattern to avoid panic when claims are missing or wrong type
			sub, _ := claims["sub"].(string)
			role, _ := claims["role"].(string)
			groupId, _ := claims["groupId"].(string)

			if sub == "" || role == "" {
				http.Error(w, "invalid token claims", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, sub)
			ctx = context.WithValue(ctx, UserRoleKey, role)
			ctx = context.WithValue(ctx, GroupIDKey, groupId)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AdminOnly middleware restricts access to users with ADMIN role.
func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := r.Context().Value(UserRoleKey).(string)
		if !ok || role != string(models.RoleAdmin) {
			http.Error(w, "forbidden: admin access required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// GetUserFromContext helper functions
func GetUserID(ctx context.Context) string {
	if v := ctx.Value(UserIDKey); v != nil {
		return v.(string)
	}
	return ""
}

func GetGroupID(ctx context.Context) string {
	if v := ctx.Value(GroupIDKey); v != nil {
		return v.(string)
	}
	return ""
}
