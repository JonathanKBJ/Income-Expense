package handlers

import (
	"encoding/json"
	"expense-tracker/internal/models"
	"expense-tracker/internal/repository"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type AdminHandler struct {
	userRepo  *repository.UserRepository
	groupRepo *repository.GroupRepository
}

func NewAdminHandler(userRepo *repository.UserRepository, groupRepo *repository.GroupRepository) *AdminHandler {
	return &AdminHandler{
		userRepo:  userRepo,
		groupRepo: groupRepo,
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

	if err := h.groupRepo.AddMember(r.Context(), groupID, req.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add member to group")
		return
	}

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
