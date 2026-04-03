package providercredentials

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/DelgadoElias/billax/internal/httputil"
	"github.com/DelgadoElias/billax/internal/middleware"
)

// Handler handles HTTP requests for provider credentials
type Handler struct {
	svc *CredentialsService
}

// NewHandler creates a new credentials handler
func NewHandler(svc *CredentialsService) *Handler {
	return &Handler{svc: svc}
}

// SetProviderCredentials handles POST /v1/provider-credentials/{provider}
// Sets or updates credentials for a payment provider
func (h *Handler) SetProviderCredentials(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	providerName := chi.URLParam(r, "provider")

	// Parse request body
	var input struct {
		AccessToken   string `json:"access_token"`
		WebhookSecret string `json:"webhook_secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	// Convert to config map
	config := map[string]string{
		"access_token":   input.AccessToken,
		"webhook_secret": input.WebhookSecret,
	}

	// Validate and store
	if err := h.svc.SetProviderConfig(r.Context(), tenantID, providerName, config); err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	// Respond with success
	httputil.RespondJSON(w, http.StatusCreated, map[string]string{
		"provider": providerName,
		"message":  "credentials configured",
	})
}

// ListProviders handles GET /v1/provider-credentials
// Lists all configured providers for the tenant
func (h *Handler) ListProviders(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	providers, err := h.svc.ListProviders(r.Context(), tenantID)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	// If no providers, return empty slice (not null)
	if providers == nil {
		providers = []string{}
	}

	httputil.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"providers": providers,
	})
}

// GetProviderStatus handles GET /v1/provider-credentials/{provider}
// Returns whether a provider is configured (doesn't expose credentials)
func (h *Handler) GetProviderStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	providerName := chi.URLParam(r, "provider")

	// Try to fetch config (we don't return it, just check if it exists)
	_, err := h.svc.GetProviderConfig(r.Context(), tenantID, providerName)

	httputil.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"provider":   providerName,
		"configured": err == nil,
	})
}

// DeleteProviderCredentials handles DELETE /v1/provider-credentials/{provider}
// Removes credentials for a provider
func (h *Handler) DeleteProviderCredentials(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	providerName := chi.URLParam(r, "provider")

	if err := h.svc.DeleteProviderConfig(r.Context(), tenantID, providerName); err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RegisterRoutes registers all credentials routes with the router
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/provider-credentials/{provider}", h.SetProviderCredentials)
	r.Get("/provider-credentials", h.ListProviders)
	r.Get("/provider-credentials/{provider}", h.GetProviderStatus)
	r.Delete("/provider-credentials/{provider}", h.DeleteProviderCredentials)
}
