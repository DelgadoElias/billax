package backoffice

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	appErrors "github.com/DelgadoElias/billax/internal/errors"
	"github.com/DelgadoElias/billax/internal/httputil"
	"github.com/DelgadoElias/billax/internal/middleware"
	"github.com/DelgadoElias/billax/internal/tenant"
)

// Handler handles HTTP requests for backoffice operations
type Handler struct {
	service    *BackofficeService
	tenantRepo tenant.TenantRepo
	logger     *slog.Logger
}

// NewHandler creates a new backoffice handler
func NewHandler(service *BackofficeService, tenantRepo tenant.TenantRepo, logger *slog.Logger) *Handler {
	return &Handler{
		service:    service,
		tenantRepo: tenantRepo,
		logger:     logger,
	}
}

// CheckEmail handles POST /v1/backoffice/check-email
// Public endpoint - finds which tenants have a backoffice user with this email
// Used by UI for tenant selection before password prompt
func (h *Handler) CheckEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, r, appErrors.ErrInvalidInput)
		return
	}

	if req.Email == "" {
		httputil.RespondError(w, r, appErrors.ErrInvalidInput)
		return
	}

	// Find all tenants that have a backoffice user with this email
	tenants, err := h.tenantRepo.GetByEmailAcrossAllTenants(r.Context(), req.Email)
	if err != nil {
		h.logger.Warn("backoffice check-email failed", "error", err.Error(), "email", req.Email)
		httputil.RespondError(w, r, appErrors.ErrNotFound)
		return
	}

	// If only one tenant, tell UI to skip selector
	if len(tenants) == 1 {
		httputil.RespondOK(w, map[string]interface{}{
			"single_tenant": true,
			"tenant_slug":   tenants[0].Slug,
			"tenant_name":   tenants[0].Name,
		})
		return
	}

	// Multiple tenants - return list for UI selector
	type TenantOption struct {
		Slug string `json:"slug"`
		Name string `json:"name"`
		ID   string `json:"id"`
	}

	options := make([]TenantOption, len(tenants))
	for i, t := range tenants {
		options[i] = TenantOption{
			Slug: t.Slug,
			Name: t.Name,
			ID:   t.ID.String(),
		}
	}

	httputil.RespondOK(w, map[string]interface{}{
		"single_tenant": false,
		"tenants":       options,
	})
}

// Login handles POST /v1/backoffice/login
// Public endpoint (no auth required)
// Requires both tenant_slug and email+password
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, r, appErrors.ErrInvalidInput)
		return
	}

	// Validate required fields
	if req.Email == "" || req.Password == "" || req.TenantSlug == "" {
		httputil.RespondError(w, r, appErrors.ErrInvalidInput)
		return
	}

	// Look up tenant by slug
	tenantData, err := h.tenantRepo.GetBySlug(r.Context(), req.TenantSlug)
	if err != nil {
		// Return generic error to prevent tenant enumeration
		h.logger.Warn("backoffice login failed: tenant not found", "tenant_slug", req.TenantSlug)
		httputil.RespondError(w, r, appErrors.ErrInvalidInput)
		return
	}

	token, user, err := h.service.Login(r.Context(), tenantData.ID, req.Email, req.Password)
	if err != nil {
		// Check if it's a must_change_password error
		if err.Error() == "login: must_change_password: user must change password before accessing backoffice" {
			h.logger.Info("login rejected: must change password", "email", req.Email, "tenant_id", tenantData.ID)
			httputil.RespondError(w, r, errors.New("must_change_password: user must change password before accessing backoffice"))
			return
		}
		h.logger.Warn("backoffice login failed", "error", err.Error(), "email", req.Email, "tenant_id", tenantData.ID)
		httputil.RespondError(w, r, err)
		return
	}

	httputil.RespondOK(w, LoginResponse{
		Token: token,
		User:  user,
	})
}

// Me handles GET /v1/backoffice/me
// Protected endpoint (auth required)
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.BackofficeUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		httputil.RespondError(w, r, appErrors.ErrInvalidInput)
		return
	}

	user, err := h.service.repo.GetByID(r.Context(), userID)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	httputil.RespondOK(w, user)
}

// UpdateProfile handles PATCH /v1/backoffice/me
// Protected endpoint (auth required)
// Users can only update their own profile
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.BackofficeUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		httputil.RespondError(w, r, appErrors.ErrInvalidInput)
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, r, appErrors.ErrInvalidInput)
		return
	}

	// Validate request - at least one field must be provided
	if req.Name == "" {
		httputil.RespondError(w, r, errors.New("at least one field must be provided for update"))
		return
	}

	user, err := h.service.UpdateProfile(r.Context(), userID, req.Name)
	if err != nil {
		h.logger.Warn("failed to update profile", "error", err.Error(), "user_id", userID)
		httputil.RespondError(w, r, err)
		return
	}

	h.logger.Info("user profile updated", "user_id", user.ID.String(), "name", user.Name)
	httputil.RespondOK(w, user)
}

// CreateUser handles POST /v1/backoffice/users
// Admin-only endpoint
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	// Check admin permission
	roleStr := middleware.BackofficeRoleFromContext(r.Context())
	if roleStr != string(RoleAdmin) {
		httputil.RespondError(w, r, errors.New("admin_required: only admins can create users"))
		return
	}

	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, r, appErrors.ErrInvalidInput)
		return
	}

	// Validate required fields
	if req.Email == "" {
		httputil.RespondError(w, r, errors.New("email is required"))
		return
	}
	if req.Name == "" {
		httputil.RespondError(w, r, errors.New("name is required"))
		return
	}
	if req.Role != RoleAdmin && req.Role != RoleMember {
		httputil.RespondError(w, r, errors.New("role must be 'admin' or 'member'"))
		return
	}

	tenantID := middleware.BackofficeTenantIDFromContext(r.Context())
	if tenantID == uuid.Nil {
		httputil.RespondError(w, r, appErrors.ErrMissingTenantID)
		return
	}

	user, err := h.service.CreateUser(r.Context(), tenantID, req.Email, req.Name, req.Password, req.Role)
	if err != nil {
		h.logger.Warn("failed to create user", "error", err.Error(), "email", req.Email)
		httputil.RespondError(w, r, err)
		return
	}

	h.logger.Info("user created", "user_id", user.ID.String(), "email", user.Email, "role", user.Role)
	httputil.RespondCreated(w, r, user)
}

// ListUsers handles GET /v1/backoffice/users
// Admin-only endpoint
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	// Check admin permission
	roleStr := middleware.BackofficeRoleFromContext(r.Context())
	if roleStr != string(RoleAdmin) {
		httputil.RespondError(w, r, errors.New("admin_required: only admins can list users"))
		return
	}

	tenantID := middleware.BackofficeTenantIDFromContext(r.Context())
	if tenantID == uuid.Nil {
		httputil.RespondError(w, r, appErrors.ErrMissingTenantID)
		return
	}

	users, err := h.service.ListUsers(r.Context(), tenantID)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	httputil.RespondOK(w, users)
}

// DeactivateUser handles DELETE /v1/backoffice/users/{id}
// Admin-only endpoint
func (h *Handler) DeactivateUser(w http.ResponseWriter, r *http.Request) {
	// Check admin permission
	roleStr := middleware.BackofficeRoleFromContext(r.Context())
	if roleStr != string(RoleAdmin) {
		httputil.RespondError(w, r, errors.New("admin_required: only admins can deactivate users"))
		return
	}

	userIDStr := chi.URLParam(r, "id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		httputil.RespondError(w, r, appErrors.ErrInvalidInput)
		return
	}

	err = h.service.DeactivateUser(r.Context(), userID)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ChangePassword handles PATCH /v1/backoffice/users/{id}/password
// Users can change their own password; admins can change others
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	targetUserID, err := uuid.Parse(userIDStr)
	if err != nil {
		httputil.RespondError(w, r, appErrors.ErrInvalidInput)
		return
	}

	// Check permission: user can change own password, admin can change any
	currentUserID := middleware.BackofficeUserIDFromContext(r.Context())
	roleStr := middleware.BackofficeRoleFromContext(r.Context())

	if currentUserID != targetUserID && roleStr != string(RoleAdmin) {
		httputil.RespondError(w, r, errors.New("forbidden: can only change your own password unless you're an admin"))
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, r, appErrors.ErrInvalidInput)
		return
	}

	err = h.service.ChangePassword(r.Context(), targetUserID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RegisterPublicRoutes registers public backoffice routes (no auth required)
func (h *Handler) RegisterPublicRoutes(r chi.Router) {
	r.Post("/backoffice/check-email", h.CheckEmail)
	r.Post("/backoffice/login", h.Login)
}

// RegisterAuthRoutes registers protected backoffice routes (auth required)
func (h *Handler) RegisterAuthRoutes(r chi.Router) {
	r.Get("/backoffice/me", h.Me)
	r.Patch("/backoffice/me", h.UpdateProfile)
	r.Post("/backoffice/users", h.CreateUser)
	r.Get("/backoffice/users", h.ListUsers)
	r.Delete("/backoffice/users/{id}", h.DeactivateUser)
	r.Patch("/backoffice/users/{id}/password", h.ChangePassword)
}
