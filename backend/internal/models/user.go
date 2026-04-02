package models

import "time"

// UserRole defines the access level of a user.
type UserRole string

const (
	RoleAdmin UserRole = "ADMIN"
	RoleUser  UserRole = "USER"
)

// UserStatus defines whether a user can log in.
type UserStatus string

const (
	StatusActive   UserStatus = "ACTIVE"
	StatusInactive UserStatus = "INACTIVE"
)

// User represents a person who can log into the system.
type User struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"-"` // Never expose hash in JSON
	Role         UserRole   `json:"role"`
	Status       UserStatus `json:"status"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

// LoginRequest is the payload for authenticating a user.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterRequest is the payload for creating a new user.
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthResponse is returned upon successful authentication.
type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// UpdateUserStatusRequest is used by Admins to toggle user access.
type UpdateUserStatusRequest struct {
	Status UserStatus `json:"status"`
}
