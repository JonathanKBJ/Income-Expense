package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"expense-tracker/internal/models"
	"expense-tracker/internal/repository"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type AdminHandler struct {
	userRepo     *repository.UserRepository
	groupRepo    *repository.GroupRepository
	activityRepo *repository.ActivityLogRepository
}

func NewAdminHandler(
	userRepo *repository.UserRepository,
	groupRepo *repository.GroupRepository,
	activityRepo *repository.ActivityLogRepository,
) *AdminHandler {
	return &AdminHandler{
		userRepo:     userRepo,
		groupRepo:    groupRepo,
		activityRepo: activityRepo,
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
