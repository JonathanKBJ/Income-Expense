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
	"expense-tracker/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// GroupHandler handles HTTP requests for group operations (Phase 1 & 2).
type GroupHandler struct {
	groupRepo    *repository.GroupRepository
	activityRepo *repository.ActivityLogRepository
	inviteRepo   *repository.GroupInviteRepository
	userRepo     *repository.UserRepository
	authService  *service.AuthService
}

// NewGroupHandler creates a new handler with the given repositories.
func NewGroupHandler(
	groupRepo *repository.GroupRepository,
	activityRepo *repository.ActivityLogRepository,
	inviteRepo *repository.GroupInviteRepository,
	userRepo *repository.UserRepository,
	authService *service.AuthService,
) *GroupHandler {
	return &GroupHandler{
		groupRepo:    groupRepo,
		activityRepo: activityRepo,
		inviteRepo:   inviteRepo,
		userRepo:     userRepo,
		authService:  authService,
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

// ListMyGroups handles GET /api/me/groups
func (h *GroupHandler) ListMyGroups(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	groups, err := h.groupRepo.ListUserGroups(r.Context(), userID)
	if err != nil {
		log.Printf("ERROR: ListMyGroups: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch user groups")
		return
	}

	writeJSON(w, http.StatusOK, groups)
}

// CreateMyGroup handles POST /api/me/groups
func (h *GroupHandler) CreateMyGroup(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req models.CreateUserGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	group, err := h.groupRepo.CreateUserGroup(r.Context(), userID, req.Name)
	if err != nil {
		log.Printf("ERROR: CreateMyGroup: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create group")
		return
	}

	// Log GROUP_CREATED activity
	h.logActivity(r.Context(), group.ID, userID, "GROUP_CREATED", "group", group.ID, "")

	writeJSON(w, http.StatusCreated, group)
}

// SwitchGroup handles POST /api/me/switch-group
func (h *GroupHandler) SwitchGroup(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req models.SwitchGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	if req.GroupID == "" {
		writeError(w, http.StatusBadRequest, "groupId is required")
		return
	}

	// Verify user is a member of the target group
	role, err := h.groupRepo.GetMemberRole(r.Context(), req.GroupID, userID)
	if err != nil {
		writeError(w, http.StatusForbidden, "you are not a member of this group")
		return
	}

	// Look up full user and group info
	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil || user == nil {
		log.Printf("ERROR: SwitchGroup: user not found: %v", err)
		writeError(w, http.StatusInternalServerError, "user not found")
		return
	}

	groupInfo, err := h.groupRepo.GetGroupInfo(r.Context(), req.GroupID, userID)
	if err != nil {
		log.Printf("ERROR: SwitchGroup: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch group info")
		return
	}

	// Generate new JWT with the target group
	token, err := h.authService.GenerateTokenForGroup(user, req.GroupID)
	if err != nil {
		log.Printf("ERROR: SwitchGroup: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, http.StatusOK, models.SwitchGroupResponse{
		Token:     token,
		GroupID:   req.GroupID,
		GroupName: groupInfo.Name,
		GroupRole: string(role),
	})
}

// DeleteMyGroup handles DELETE /api/me/groups/{id}
func (h *GroupHandler) DeleteMyGroup(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	groupID := chi.URLParam(r, "id")

	if groupID == "" {
		writeError(w, http.StatusBadRequest, "group id is required")
		return
	}

	if err := h.groupRepo.DeleteGroup(r.Context(), groupID, userID); err != nil {
		log.Printf("ERROR: DeleteMyGroup: %v", err)
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "group deleted"})
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

	// Auto-switch to the new group: return a fresh JWT
	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil || user == nil {
		log.Printf("ERROR: JoinGroup: user not found: %v", err)
		writeJSON(w, http.StatusOK, map[string]string{
			"message": "joined group successfully. please re-login to update your session.",
		})
		return
	}

	groupInfo, err := h.groupRepo.GetGroupInfo(r.Context(), targetGroupID, userID)
	if err != nil {
		log.Printf("ERROR: JoinGroup: %v", err)
		writeJSON(w, http.StatusOK, map[string]string{
			"message": "joined group successfully. please re-login to update your session.",
		})
		return
	}

	token, err := h.authService.GenerateTokenForGroup(user, targetGroupID)
	if err != nil {
		log.Printf("ERROR: JoinGroup token generation: %v", err)
		writeJSON(w, http.StatusOK, map[string]string{
			"message": "joined group successfully. please re-login to update your session.",
		})
		return
	}

	writeJSON(w, http.StatusOK, models.SwitchGroupResponse{
		Token:     token,
		GroupID:   targetGroupID,
		GroupName: groupInfo.Name,
		GroupRole: string(models.RoleEditor),
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

	// Remove the user from this group (no personal group restoration — user keeps other groups)
	ctx := r.Context()
		if err := h.groupRepo.DeleteMember(ctx, groupID, userID); err != nil {
		log.Printf("ERROR: LeaveGroup: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to leave group")
		return
	}

	// Auto-switch to the first remaining group and return new JWT
	groups, err := h.groupRepo.ListUserGroups(ctx, userID)
	if err != nil || len(groups) == 0 {
		writeJSON(w, http.StatusOK, map[string]string{
			"message": "left group successfully. no remaining groups. please re-login.",
		})
		return
	}

	nextGroup := groups[0]
	user, userErr := h.userRepo.GetByID(ctx, userID)
	if userErr != nil || user == nil {
		log.Printf("ERROR: LeaveGroup: user not found: %v", userErr)
		writeJSON(w, http.StatusOK, map[string]string{
			"message": "left group successfully. please re-login to update your session.",
		})
		return
	}
	token, tokenErr := h.authService.GenerateTokenForGroup(user, nextGroup.ID)
	if tokenErr != nil {
		log.Printf("ERROR: LeaveGroup token: %v", tokenErr)
		writeJSON(w, http.StatusOK, map[string]string{
			"message": "left group successfully. please re-login to update your session.",
		})
		return
	}

	writeJSON(w, http.StatusOK, models.SwitchGroupResponse{
		Token:     token,
		GroupID:   nextGroup.ID,
		GroupName: nextGroup.Name,
		GroupRole: string(nextGroup.MyRole),
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
