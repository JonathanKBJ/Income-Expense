package models

import "time"

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
