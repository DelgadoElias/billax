package webhook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/DelgadoElias/billax/internal/errors"
	"github.com/DelgadoElias/billax/internal/httputil"
	"github.com/DelgadoElias/billax/internal/middleware"
	"github.com/DelgadoElias/billax/internal/payment"
	"github.com/DelgadoElias/billax/internal/provider"
)

// Handler handles incoming webhooks from payment providers
type Handler struct {
	paymentRepo   payment.PaymentRepo
	credentialsSvc CredentialsService // interface for webhook secret lookup
	adapter       *provider.ProviderAdapter
}

// CredentialsService defines how to fetch credentials by webhook secret
type CredentialsService interface {
	// LookupByWebhookSecret discovers tenant and provider by webhook secret
	// Used to route incoming webhooks without auth context
	LookupByWebhookSecret(ctx context.Context, secret string) (uuid.UUID, string, map[string]string, error)
}

// NewHandler creates a new webhook handler
func NewHandler(paymentRepo payment.PaymentRepo, credentialsSvc CredentialsService, adapter *provider.ProviderAdapter) *Handler {
	return &Handler{
		paymentRepo:   paymentRepo,
		credentialsSvc: credentialsSvc,
		adapter:       adapter,
	}
}

// HandleWebhook handles POST /webhooks/{webhookSecret}
// This is a generic handler that routes to the correct provider based on the secret lookup
// The webhook secret is embedded in the URL for public access without authentication
func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Extract webhook secret from URL parameter
	webhookSecret := chi.URLParam(r, "webhookSecret")
	if webhookSecret == "" {
		httputil.RespondError(w, r, apperrors.ErrInvalidInput)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		httputil.RespondError(w, r, apperrors.ErrInvalidInput)
		return
	}
	defer r.Body.Close()

	// Look up tenant and provider by webhook secret
	tenantID, providerName, _, err := h.credentialsSvc.LookupByWebhookSecret(r.Context(), webhookSecret)
	if err != nil {
		// Secret not found or invalid
		httputil.RespondError(w, r, apperrors.ErrWebhookSignatureInvalid)
		return
	}

	// Set tenant context for RLS and audit logging
	ctx := middleware.WithTenantID(r.Context(), tenantID)

	// Extract provider-specific headers for signature validation
	// All providers follow a pattern: some have a signature header, some have a request ID for replay protection
	xSignature := r.Header.Get("X-Signature")
	xRequestID := r.Header.Get("X-Request-ID")

	// For Mercado Pago: assemble signature in the convention expected by the connector:
	// "<webhook_secret>|<x_request_id>|<raw_x_signature>"
	// Other providers may have different conventions
	var signature string
	if providerName == "mercadopago" {
		if xSignature == "" || xRequestID == "" {
			httputil.RespondError(w, r, apperrors.ErrInvalidInput)
			return
		}
		signature = webhookSecret + "|" + xRequestID + "|" + xSignature
	} else {
		// For future providers, adapt as needed
		signature = xSignature
	}

	// Validate signature and parse webhook via provider adapter
	_, err = h.adapter.HandleWebhook(ctx, providerName, body, signature)
	if err != nil {
		// Adapt the error to the appropriate HTTP response
		httputil.RespondError(w, r, err)
		return
	}

	// At this point, signature is valid
	// The parsed event contains the payment ID and status

	// For MVP: just acknowledge receipt
	// In production: would fetch actual payment status from provider and update our DB
	//
	// TODO: Implement in next phase (subscription lifecycle)
	// 1. Fetch payment from provider based on provider charge ID
	// 2. Update our payment record with actual status
	// 3. Trigger subscription renewal job if status is succeeded

	// Return 200 OK to acknowledge
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "webhook received",
	})
}

// RegisterRoutes registers webhook routes on a chi.Router
func (h *Handler) RegisterRoutes(r chi.Router) {
	// Public webhook endpoint: no authentication required
	// Route: POST /webhooks/{webhookSecret}
	// The webhook secret is used to discover tenant and provider
	r.Post("/webhooks/{webhookSecret}", h.HandleWebhook)
}
