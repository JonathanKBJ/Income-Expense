package middleware

import (
	"expense-tracker/internal/models"
	"net/http"
)

// GroupRoleMiddleware restricts access based on group role hierarchy: OWNER > EDITOR > VIEWER.
func GroupRoleMiddleware(minRole models.GroupRole) func(http.Handler) http.Handler {
	roleLevel := map[models.GroupRole]int{
		models.RoleOwner:  3,
		models.RoleEditor: 2,
		models.RoleViewer: 1,
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := GetGroupRole(r.Context())
			if role == "" {
				http.Error(w, "group role missing from context", http.StatusInternalServerError)
				return
			}

			if roleLevel[models.GroupRole(role)] < roleLevel[minRole] {
				http.Error(w, "insufficient group permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
