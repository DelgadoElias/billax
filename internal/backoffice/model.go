package backoffice

import (
	"time"

	"github.com/google/uuid"
)

// Role represents the backoffice user role
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

// User represents a backoffice user (for login/permission, not exposed to API responses)
type User struct {
	ID                 uuid.UUID
	TenantID           uuid.UUID
	Email              string
	Role               Role
	Name               string
	MustChangePassword bool
	IsActive           bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
	// PasswordHash intentionally NOT included in struct to prevent accidental exposure
}

// LoginRequest is the request body for POST /v1/backoffice/login
type LoginRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	TenantSlug string `json:"tenant_slug,omitempty"` // optional: if not provided, will be looked up by email
}

// LoginResponse is the response body for successful login
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// CreateUserRequest is the request body for creating a new backoffice user (admin only)
type CreateUserRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  Role   `json:"role"`
	// Password is optional — if empty, user gets a temporary password + must_change_password flag
	Password string `json:"password,omitempty"`
}

// ChangePasswordRequest is the request body for password changes
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// UpdateProfileRequest is the request body for profile updates (PATCH /v1/backoffice/me)
type UpdateProfileRequest struct {
	Name string `json:"name,omitempty"` // Only name can be updated (email is immutable)
}

// JWTClaims represents the claims in a backoffice JWT token
type JWTClaims struct {
	UserID   uuid.UUID `json:"user_id"`
	TenantID uuid.UUID `json:"tenant_id"`
	Email    string    `json:"email"`
	Role     Role      `json:"role"`
}
