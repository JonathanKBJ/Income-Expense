package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"expense-tracker/internal/models"
	"expense-tracker/internal/repository"
	"expense-tracker/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type AdminHandler struct {
	userRepo     *repository.UserRepository
	groupRepo    *repository.GroupRepository
	activityRepo *repository.ActivityLogRepository
	authService  *service.AuthService
}

func NewAdminHandler(
	userRepo *repository.UserRepository,
	groupRepo *repository.GroupRepository,
	activityRepo *repository.ActivityLogRepository,
	authService *service.AuthService,
) *AdminHandler {
	return &AdminHandler{
		userRepo:     userRepo,
		groupRepo:    groupRepo,
		activityRepo: activityRepo,
		authService:  authService,
	}
}

// ListUsers handles GET /api/admin/users
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.userRepo.ListUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list users")
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// UpdateUserStatus handles PATCH /api/admin/users/{id}/status
func (h *AdminHandler) UpdateUserStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req models.UpdateUserStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.userRepo.UpdateStatus(r.Context(), id, req.Status); err != nil {
		if err.Error() == "user not found" {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update user status")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "status updated"})
}

// ListGroups handles GET /api/admin/groups
func (h *AdminHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.groupRepo.ListGroups(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list groups")
		return
	}
	writeJSON(w, http.StatusOK, groups)
}

// AddMemberToGroup handles POST /api/admin/groups/{id}/members
func (h *AdminHandler) AddMemberToGroup(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "id")
	var req struct {
		UserID string `json:"userID"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.groupRepo.AddMember(r.Context(), groupID, req.UserID, models.RoleEditor); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add member to group")
		return
	}

	// Log MEMBER_JOINED activity
	h.logActivity(r.Context(), groupID, req.UserID, "MEMBER_JOINED", "group", groupID, "")

	writeJSON(w, http.StatusOK, map[string]string{"message": "member added to group"})
}

// GetGroupMembers handles GET /api/admin/groups/{id}/members
func (h *AdminHandler) GetGroupMembers(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "id")
	members, err := h.groupRepo.GetGroupMembers(r.Context(), groupID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch group members")
		return
	}
	writeJSON(w, http.StatusOK, members)
}

// RemoveMemberFromGroup handles DELETE /api/admin/groups/{id}/members/{userID}
func (h *AdminHandler) RemoveMemberFromGroup(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "id")
	userID := chi.URLParam(r, "userID")

	// Log MEMBER_LEFT before removal
	h.logActivity(r.Context(), groupID, userID, "MEMBER_LEFT", "group", groupID, "")

	if err := h.groupRepo.RemoveMember(r.Context(), groupID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove member from group")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "member removed from group"})
}

// ResetUserPassword handles PATCH /api/admin/users/{id}/reset-password
// Allows an admin to reset any user's password.
func (h *AdminHandler) ResetUserPassword(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req models.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Password == "" || len(req.Password) < 6 {
		writeError(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	// Fetch the user to get username for peppering
	user, err := h.userRepo.GetByID(r.Context(), id)
	if err != nil || user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	// Hash the new password with the same pepper+bcrypt scheme
	hashed, err := h.authService.HashPassword(req.Password, user.Username)
	if err != nil {
		log.Printf("ERROR: ResetUserPassword (hash): %v", err)
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	if err := h.userRepo.UpdatePassword(r.Context(), id, hashed); err != nil {
		log.Printf("ERROR: ResetUserPassword (update): %v", err)
		if err.Error() == "user not found" {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to reset password")
		return
	}

	writeJSON(w, http.StatusOK, models.ResetPasswordResponse{
		Message:  "password reset successfully",
		Username: user.Username,
	})
}

// logActivity creates an activity log entry only for multi-member groups.
func (h *AdminHandler) logActivity(ctx context.Context, groupID, userID, action, entityType, entityID, details string) {
	isMulti, err := h.groupRepo.GroupHasMultipleMembers(ctx, groupID)
	if err != nil || !isMulti {
		return
	}

	entry := &models.ActivityLogEntry{
		ID:         uuid.New().String(),
		GroupID:    groupID,
		UserID:     userID,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		Details:    details,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	if err := h.activityRepo.CreateEntry(ctx, entry); err != nil {
		log.Printf("WARN: failed to log activity: %v", err)
	}
}
