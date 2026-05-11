package models

import "time"

// GroupRole defines the permission level within a group.
type GroupRole string

const (
	RoleOwner  GroupRole = "OWNER"
	RoleEditor GroupRole = "EDITOR"
	RoleViewer GroupRole = "VIEWER"
)

// UserGroup defines a collection of users who share transactions and categories.
type UserGroup struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

// CreateGroupRequest is used by Admins to manually create a new group.
type CreateGroupRequest struct {
	Name string `json:"name"`
}

// ManageGroupMembersRequest is used to add/remove users from a group.
type ManageGroupMembersRequest struct {
	UserIDs []string `json:"userIDs"`
}

// GroupMember represents a user with their role in a group.
type GroupMember struct {
	UserID   string    `json:"userId"`
	Username string    `json:"username"`
	Role     GroupRole `json:"role"`
	JoinedAt string    `json:"joinedAt"`
}

// GroupInfo is returned by GET /api/me/group (the user's current group context).
type GroupInfo struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	MemberCount int           `json:"memberCount"`
	Members     []GroupMember `json:"members"`
	MyRole      GroupRole     `json:"myRole"`
}

// ActivityLogEntry represents one activity record.
type ActivityLogEntry struct {
	ID         string `json:"id"`
	GroupID    string `json:"groupId"`
	UserID     string `json:"userId"`
	Username   string `json:"username"`
	Action     string `json:"action"`
	EntityType string `json:"entityType"`
	EntityID   string `json:"entityId"`
	Details    string `json:"details"`
	CreatedAt  string `json:"createdAt"`
}

// GroupInvite represents an invitation to join a group.
type GroupInvite struct {
	Code      string `json:"code"`
	GroupName string `json:"groupName"`
	CreatedBy string `json:"createdBy"`
	ExpiresAt string `json:"expiresAt"`
}

// CreateInviteResponse is the response after creating an invite.
type CreateInviteResponse struct {
	Code      string `json:"code"`
	ExpiresAt string `json:"expiresAt"`
}

// JoinGroupRequest is the payload for joining a group via invite code.
type JoinGroupRequest struct {
	InviteCode string `json:"inviteCode"`
}

// UpdateGroupNameRequest is the payload for PATCH /api/me/group.
type UpdateGroupNameRequest struct {
	Name string `json:"name"`
}
