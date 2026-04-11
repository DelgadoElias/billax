package tenant

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/DelgadoElias/billax/internal/httputil"
	"github.com/DelgadoElias/billax/internal/middleware"
)

// Handler handles HTTP requests for tenant management
type Handler struct {
	svc *TenantService
}

// NewHandler creates a new tenant handler
func NewHandler(svc *TenantService) *Handler {
	return &Handler{svc: svc}
}

// Signup handles POST /v1/signup (public, no auth required)
func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	var input SignupInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httputil.RespondError(w, r, err)
		return
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
