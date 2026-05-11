package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"expense-tracker/internal/middleware"
	"expense-tracker/internal/models"
	"expense-tracker/internal/repository"

	"github.com/google/uuid"
)

// GroupHandler handles HTTP requests for group operations (Phase 1 & 2).
type GroupHandler struct {
	groupRepo    *repository.GroupRepository
	activityRepo *repository.ActivityLogRepository
	inviteRepo   *repository.GroupInviteRepository
}

// NewGroupHandler creates a new handler with the given repositories.
func NewGroupHandler(
	groupRepo *repository.GroupRepository,
	activityRepo *repository.ActivityLogRepository,
	inviteRepo *repository.GroupInviteRepository,
) *GroupHandler {
	return &GroupHandler{
		groupRepo:    groupRepo,
		activityRepo: activityRepo,
		inviteRepo:   inviteRepo,
	}
}

// GetMyGroup handles GET /api/me/group
func (h *GroupHandler) GetMyGroup(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	groupID := middleware.GetGroupID(r.Context())

	if groupID == "" {
		writeError(w, http.StatusBadRequest, "no group associated with this user")
		return
	}

	info, err := h.groupRepo.GetGroupInfo(r.Context(), groupID, userID)
	if err != nil {
		log.Printf("ERROR: GetMyGroup: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch group info")
		return
	}

	writeJSON(w, http.StatusOK, info)
}

// UpdateGroupName handles PATCH /api/me/group (OWNER only).
func (h *GroupHandler) UpdateGroupName(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	groupID := middleware.GetGroupID(r.Context())

	// Verify OWNER role
	role, err := h.groupRepo.GetMemberRole(r.Context(), groupID, userID)
	if err != nil || role != models.RoleOwner {
		writeError(w, http.StatusForbidden, "only the group owner can rename the group")
		return
	}

	var req models.UpdateGroupNameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if err := h.groupRepo.UpdateGroupName(r.Context(), groupID, req.Name); err != nil {
		log.Printf("ERROR: UpdateGroupName: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update group name")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "group name updated"})
}

// GetActivityFeed handles GET /api/me/activity?limit=10
func (h *GroupHandler) GetActivityFeed(w http.ResponseWriter, r *http.Request) {
	groupID := middleware.GetGroupID(r.Context())
	limit := 10

	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}

	entries, err := h.activityRepo.GetRecentByGroup(r.Context(), groupID, limit)
	if err != nil {
		log.Printf("ERROR: GetActivityFeed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch activity feed")
		return
	}

	writeJSON(w, http.StatusOK, entries)
}

// CreateInvite handles POST /api/me/group/invite (OWNER only).
func (h *GroupHandler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	groupID := middleware.GetGroupID(r.Context())

	role, err := h.groupRepo.GetMemberRole(r.Context(), groupID, userID)
	if err != nil || role != models.RoleOwner {
		writeError(w, http.StatusForbidden, "only the group owner can create invites")
		return
	}

	resp, err := h.inviteRepo.CreateInvite(r.Context(), groupID, userID)
	if err != nil {
		log.Printf("ERROR: CreateInvite: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create invite")
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// JoinGroup handles POST /api/me/group/join
func (h *GroupHandler) JoinGroup(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	currentGroupID := middleware.GetGroupID(r.Context())

	var req models.JoinGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	if req.InviteCode == "" {
		writeError(w, http.StatusBadRequest, "inviteCode is required")
		return
	}

	// Validate invite
	targetGroupID, err := h.inviteRepo.ValidateInvite(r.Context(), req.InviteCode)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Prevent joining own group
	if targetGroupID == currentGroupID {
		writeError(w, http.StatusBadRequest, "you are already a member of this group")
		return
	}

	// Add user to target group with EDITOR role
	if err := h.groupRepo.AddMember(r.Context(), targetGroupID, userID, models.RoleEditor); err != nil {
		log.Printf("ERROR: JoinGroup: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to join group")
		return
	}

	// Delete the invite (single-use)
	_ = h.inviteRepo.DeleteInvite(r.Context(), req.InviteCode)

	// Log MEMBER_JOINED activity
	h.logActivity(r.Context(), targetGroupID, userID, "MEMBER_JOINED", "group", targetGroupID, "")

	// User must re-login to get new JWT with updated groupId + groupRole
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "joined group successfully. please re-login to update your session.",
	})
}

// LeaveGroup handles POST /api/me/group/leave
func (h *GroupHandler) LeaveGroup(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	groupID := middleware.GetGroupID(r.Context())

	// Check if user is the only OWNER
	role, err := h.groupRepo.GetMemberRole(r.Context(), groupID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check member role")
		return
	}

	if role == models.RoleOwner {
		// Check if there are other OWNERs
		info, err := h.groupRepo.GetGroupInfo(r.Context(), groupID, userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to get group info")
			return
		}
		hasOtherOwner := false
		for _, m := range info.Members {
			if m.UserID != userID && m.Role == models.RoleOwner {
				hasOtherOwner = true
				break
			}
		}
		if !hasOtherOwner {
			writeError(w, http.StatusConflict,
				"you are the only owner. promote another member to owner before leaving, or delete the group")
			return
		}
	}

	// Log before removal (groupID still valid)
	h.logActivity(r.Context(), groupID, userID, "MEMBER_LEFT", "group", groupID, "")

	// Remove member (returns to personal group)
	if err := h.groupRepo.RemoveMember(r.Context(), groupID, userID); err != nil {
		log.Printf("ERROR: LeaveGroup: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to leave group")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "left group successfully. please re-login to update your session.",
	})
}

func (h *GroupHandler) logActivity(ctx context.Context, groupID, userID, action, entityType, entityID, details string) {
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
