package tenant

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/DelgadoElias/billax/internal/config"
	appErrors "github.com/DelgadoElias/billax/internal/errors"
	"github.com/DelgadoElias/billax/internal/httputil"
	"github.com/DelgadoElias/billax/internal/middleware"
)

// BackofficeService defines the interface for backoffice operations
// This is kept minimal to avoid import cycles - all parameters use interface{}
type BackofficeService interface {
	CreateUser(ctx, tenantID interface{}, email, name, password string, role interface{}) (interface{}, error)
}

// Handler handles HTTP requests for tenant management
type Handler struct {
	svc               *TenantService
	backofficeService BackofficeService
	cfg               *config.Config
}

// NewHandler creates a new tenant handler
func NewHandler(svc *TenantService) *Handler {
	return &Handler{svc: svc, cfg: &config.Config{}}
}

// NewHandlerWithBackoffice creates a tenant handler with backoffice admin creation support
func NewHandlerWithBackoffice(svc *TenantService, backofficeService BackofficeService, cfg *config.Config) *Handler {
	if cfg == nil {
		cfg = &config.Config{}
	}
	return &Handler{
		svc:               svc,
		backofficeService: backofficeService,
		cfg:               cfg,
	}
}

// Signup handles POST /v1/signup (public, no auth required)
func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	// Check if signups are allowed
	if !h.cfg.AllowSignup {
		httputil.RespondError(w, r, appErrors.ErrSignupDisabled)
		return
	}

	var input SignupInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	// Validate required fields
	if input.Name == "" || input.Email == "" {
		httputil.RespondError(w, r, fmt.Errorf("name and email are required"))
		return
	}

	// If password is required, validate it
	if h.cfg.SignupRequiresPassword {
		if input.Password == "" {
			httputil.RespondError(w, r, fmt.Errorf("password is required for signup"))
			return
		}
		// Password should be at least 8 characters
		if len(input.Password) < 8 {
			httputil.RespondError(w, r, fmt.Errorf("password must be at least 8 characters"))
			return
		}
	}

	if h.svc == nil {
		httputil.RespondError(w, r, fmt.Errorf("tenant service not initialized"))
		return
	}

	tenant, plaintextKey, err := h.svc.Signup(r.Context(), input)
	if err != nil {
		httputil.RespondError(w, r, fmt.Errorf("signup error: %w", err))
		return
	}

	// Create the admin backoffice user if password is provided and backoffice service is available
	if h.backofficeService != nil && input.Password != "" {
		_, err := h.backofficeService.CreateUser(r.Context(), interface{}(tenant.ID), input.Email, input.Name, input.Password, "admin")
		if err != nil {
			// Log the error but don't fail the signup - the tenant is already created
			// The admin user creation can be retried separately if needed
			fmt.Printf("warning: failed to create admin user during signup: %v\n", err)
		}
	}

	response := SignupResponse{
		Tenant: tenant,
		APIKey: plaintextKey,
		Warning: "Store this key securely. It will not be shown again.",
	}

	httputil.RespondCreated(w, r, response)
}

// CreateKey handles POST /v1/keys (authenticated)
func (h *Handler) CreateKey(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	if tenantID == uuid.Nil {
		httputil.RespondError(w, r, fmt.Errorf("missing tenant context"))
		return
	}

	var input CreateKeyInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	plaintextKey, err := h.svc.CreateKey(r.Context(), tenantID, input)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	httputil.RespondCreated(w, r, plaintextKey)
}

// ListKeys handles GET /v1/keys (authenticated)
func (h *Handler) ListKeys(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	if tenantID == uuid.Nil {
		httputil.RespondError(w, r, fmt.Errorf("missing tenant context"))
		return
	}

	keys, err := h.svc.ListKeys(r.Context(), tenantID)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	if keys == nil {
		keys = []APIKey{}
	}

	response := ListKeysResponse{Keys: keys}
	httputil.RespondOK(w, response)
}

// RevokeKey handles DELETE /v1/keys/{keyID} (authenticated)
func (h *Handler) RevokeKey(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	if tenantID == uuid.Nil {
		httputil.RespondError(w, r, fmt.Errorf("missing tenant context"))
		return
	}

	keyID := chi.URLParam(r, "keyID")
	if keyID == "" {
		httputil.RespondError(w, r, fmt.Errorf("missing keyID parameter"))
		return
	}

	keyUUID, err := uuid.Parse(keyID)
	if err != nil {
		httputil.RespondError(w, r, fmt.Errorf("invalid keyID format"))
		return
	}

	if err := h.svc.RevokeKey(r.Context(), tenantID, keyUUID); err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SetDefaultProvider handles POST /v1/config/default-provider (authenticated)
func (h *Handler) SetDefaultProvider(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	if tenantID == uuid.Nil {
		httputil.RespondError(w, r, fmt.Errorf("missing tenant context"))
		return
	}

	var input struct {
		ProviderName string `json:"provider_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	tenant, err := h.svc.SetDefaultProvider(r.Context(), tenantID, input.ProviderName)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	httputil.RespondOK(w, tenant)
}

// RegisterRoutes registers public tenant routes (signup)
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/signup", h.Signup)
}

// RegisterAuthRoutes registers authenticated tenant routes (key management and configuration)
func (h *Handler) RegisterAuthRoutes(r chi.Router) {
	r.Post("/keys", h.CreateKey)
	r.Get("/keys", h.ListKeys)
	r.Delete("/keys/{keyID}", h.RevokeKey)
	r.Post("/config/default-provider", h.SetDefaultProvider)
}
