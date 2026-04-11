package backoffice

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/DelgadoElias/billax/internal/errors"
)

// Password validation constants
const (
	MinPasswordLength = 8
	MaxPasswordLength = 128
)

// BackofficeService provides operations for backoffice authentication and user management
type BackofficeService struct {
	repo      UserRepo
	jwtSecret string
	jwtTTL    time.Duration
}

// NewService creates a new backoffice service
func NewService(repo UserRepo, jwtSecret string, jwtTTL time.Duration) *BackofficeService {
	return &BackofficeService{
		repo:      repo,
		jwtSecret: jwtSecret,
		jwtTTL:    jwtTTL,
	}
}

// Login authenticates a user and returns a JWT token
// Returns (token, user, error)
func (s *BackofficeService) Login(ctx context.Context, tenantID uuid.UUID, email, password string) (string, User, error) {
	if tenantID == uuid.Nil {
		return "", User{}, fmt.Errorf("login: %w", errors.ErrMissingTenantID)
	}
	if email == "" {
		return "", User{}, fmt.Errorf("login: email cannot be empty: %w", errors.ErrInvalidInput)
	}
	if password == "" {
		return "", User{}, fmt.Errorf("login: password cannot be empty: %w", errors.ErrInvalidInput)
	}

	// Fetch user by email
	user, passwordHash, err := s.repo.GetByEmail(ctx, tenantID, email)
	if err != nil {
		// Return generic error to prevent email enumeration
		return "", User{}, fmt.Errorf("login: %w", errors.ErrInvalidInput)
	}

	// Check if user is active
	if !user.IsActive {
		return "", User{}, fmt.Errorf("login: user is inactive: %w", errors.ErrInvalidInput)
	}

	// Verify password against bcrypt hash
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		slog.Warn("login failed: invalid password", "email", email, "tenant_id", tenantID)
		// Return generic error
		return "", User{}, fmt.Errorf("login: %w", errors.ErrInvalidInput)
	}

	// Check if user must change password
	if user.MustChangePassword {
		return "", User{}, fmt.Errorf("login: must_change_password: user must change password before accessing backoffice")
	}

	// Generate JWT token
	token, err := GenerateToken(user, s.jwtSecret, s.jwtTTL)
	if err != nil {
		return "", User{}, fmt.Errorf("login: %w", err)
	}

	// Audit log
	slog.Info("backoffice login",
		"tenant_id", tenantID.String(),
		"user_id", user.ID.String(),
		"email", email,
	)

	return token, user, nil
}

// CreateUser creates a new backoffice user (admin only)
// If password is empty, a temporary password is generated and must_change_password is set to true
func (s *BackofficeService) CreateUser(ctx context.Context, tenantID uuid.UUID, email, name, password string, role Role) (User, error) {
	if tenantID == uuid.Nil {
		return User{}, fmt.Errorf("create user: %w", errors.ErrMissingTenantID)
	}
	if email == "" {
		return User{}, fmt.Errorf("create user: email cannot be empty: %w", errors.ErrInvalidInput)
	}
	if name == "" {
		return User{}, fmt.Errorf("create user: name cannot be empty: %w", errors.ErrInvalidInput)
	}
	if role != RoleAdmin && role != RoleMember {
		return User{}, fmt.Errorf("create user: invalid role: %w", errors.ErrInvalidInput)
	}

	// Use provided password or generate temporary one
	pwdToHash := password
	mustChangePassword := false
	if pwdToHash == "" {
		pwdToHash = generateTemporaryPassword()
		mustChangePassword = true
	} else {
		// Validate password requirements
		if err := validatePassword(password); err != nil {
			return User{}, fmt.Errorf("create user: %w", err)
		}
	}

	// Hash password with bcrypt
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(pwdToHash), bcrypt.DefaultCost)
	if err != nil {
		return User{}, fmt.Errorf("create user: hashing password: %w", err)
	}

	// Create user
	user, err := s.repo.Create(ctx, tenantID, email, name, string(passwordHash), role)
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}

	// If password was temporary, mark user as must_change_password
	if mustChangePassword {
		if err := s.repo.UpdateMustChangePassword(ctx, user.ID, true); err != nil {
			return User{}, fmt.Errorf("create user: setting must_change_password: %w", err)
		}
		user.MustChangePassword = true
	}

	// Audit log
	slog.Info("backoffice user created",
		"tenant_id", tenantID.String(),
		"user_id", user.ID.String(),
		"email", email,
		"role", role,
		"must_change_password", mustChangePassword,
	)

	return user, nil
}

// ListUsers returns all users for a tenant (admin only)
func (s *BackofficeService) ListUsers(ctx context.Context, tenantID uuid.UUID) ([]User, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("list users: %w", errors.ErrMissingTenantID)
	}

	users, err := s.repo.List(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	return users, nil
}

// DeactivateUser deactivates a user (admin only)
func (s *BackofficeService) DeactivateUser(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("deactivate user: invalid user id: %w", errors.ErrInvalidInput)
	}

	if err := s.repo.Deactivate(ctx, id); err != nil {
		return fmt.Errorf("deactivate user: %w", err)
	}

	// Audit log
	slog.Info("backoffice user deactivated", "user_id", id.String())

	return nil
}

// ChangePassword changes a user's password
// The current password must match
func (s *BackofficeService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	if userID == uuid.Nil {
		return fmt.Errorf("change password: invalid user id: %w", errors.ErrInvalidInput)
	}
	if currentPassword == "" {
		return fmt.Errorf("change password: current password cannot be empty: %w", errors.ErrInvalidInput)
	}
	if newPassword == "" {
		return fmt.Errorf("change password: new password cannot be empty: %w", errors.ErrInvalidInput)
	}

	// Validate new password requirements
	if err := validatePassword(newPassword); err != nil {
		return fmt.Errorf("change password: %w", err)
	}

	// Prevent reusing the same password
	if currentPassword == newPassword {
		return fmt.Errorf("change password: new password must be different from current password: %w", errors.ErrInvalidInput)
	}

	// Fetch current password hash
	passwordHash, err := s.repo.GetPasswordHashByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("change password: %w", err)
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(currentPassword)); err != nil {
		slog.Warn("change password failed: invalid current password", "user_id", userID.String())
		return fmt.Errorf("change password: current password incorrect: %w", errors.ErrInvalidInput)
	}

	// Hash new password
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("change password: hashing new password: %w", err)
	}

	// Update password
	if err := s.repo.UpdatePassword(ctx, userID, string(newHash)); err != nil {
		return fmt.Errorf("change password: %w", err)
	}

	// Audit log
	slog.Info("backoffice user password changed", "user_id", userID.String())

	return nil
}

// validatePassword validates password against requirements
// Requirements:
// - Minimum 8 characters
// - Maximum 128 characters
// - At least one uppercase letter
// - At least one lowercase letter
// - At least one digit
// - At least one special character (!@#$%^&*)
func validatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("password_too_short: password must be at least %d characters: %w", MinPasswordLength, errors.ErrInvalidInput)
	}
	if len(password) > MaxPasswordLength {
		return fmt.Errorf("password_too_long: password must not exceed %d characters: %w", MaxPasswordLength, errors.ErrInvalidInput)
	}

	// Check for uppercase letter
	if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		return fmt.Errorf("password_missing_uppercase: password must contain at least one uppercase letter: %w", errors.ErrInvalidInput)
	}

	// Check for lowercase letter
	if !regexp.MustCompile(`[a-z]`).MatchString(password) {
		return fmt.Errorf("password_missing_lowercase: password must contain at least one lowercase letter: %w", errors.ErrInvalidInput)
	}

	// Check for digit
	if !regexp.MustCompile(`[0-9]`).MatchString(password) {
		return fmt.Errorf("password_missing_digit: password must contain at least one digit: %w", errors.ErrInvalidInput)
	}

	// Check for special character
	if !regexp.MustCompile(`[!@#$%^&*]`).MatchString(password) {
		return fmt.Errorf("password_missing_special: password must contain at least one special character (!@#$%%^&*): %w", errors.ErrInvalidInput)
	}

	return nil
}

// UpdateProfile updates a user's profile (name only)
func (s *BackofficeService) UpdateProfile(ctx context.Context, userID uuid.UUID, name string) (User, error) {
	if userID == uuid.Nil {
		return User{}, fmt.Errorf("update profile: invalid user id: %w", errors.ErrInvalidInput)
	}
	if name == "" {
		return User{}, fmt.Errorf("update profile: name cannot be empty: %w", errors.ErrInvalidInput)
	}

	// Update name in repository
	if err := s.repo.UpdateName(ctx, userID, name); err != nil {
		return User{}, fmt.Errorf("update profile: %w", err)
	}

	// Fetch updated user
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return User{}, fmt.Errorf("update profile: %w", err)
	}

	// Audit log
	slog.Info("backoffice user profile updated", "user_id", userID.String(), "name", name)

	return user, nil
}

// generateTemporaryPassword generates a random 12-character password with mixed requirements
func generateTemporaryPassword() string {
	// Generate a 12-character password with: uppercase, lowercase, digit, special char
	// Pattern: Aa1!Aa1!Aa1!
	uppercase := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lowercase := "abcdefghijklmnopqrstuvwxyz"
	digits := "0123456789"
	special := "!@#$%^&*"

	b := make([]byte, 12)

	// Ensure at least one of each type
	b[0] = uppercase[randomIndex(len(uppercase))]
	b[1] = lowercase[randomIndex(len(lowercase))]
	b[2] = digits[randomIndex(len(digits))]
	b[3] = special[randomIndex(len(special))]

	// Fill the rest with a mix
	allChars := uppercase + lowercase + digits + special
	for i := 4; i < 12; i++ {
		b[i] = allChars[randomIndex(len(allChars))]
	}

	// Shuffle the bytes
	shuffled := make([]byte, 12)
	for i := 0; i < 12; i++ {
		idx := randomIndex(12)
		shuffled[i] = b[idx]
		// Simple shuffle - swap with random position
		b[idx], b[i] = b[i], b[idx]
	}

	return string(b)
}

// randomIndex returns a random index in range [0, max)
func randomIndex(max int) int {
	b := make([]byte, 1)
	if _, err := rand.Read(b); err != nil {
		// Fallback in case of crypto/rand failure
		return 0
	}
	return int(b[0]) % max
}
