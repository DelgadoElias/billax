package payment

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/DelgadoElias/billax/internal/errors"
	"github.com/DelgadoElias/billax/internal/httputil"
	"github.com/DelgadoElias/billax/internal/middleware"
	"github.com/DelgadoElias/billax/internal/providercredentials"
)

type Handler struct {
	svc     *PaymentService
	credSvc *providercredentials.CredentialsService
}

func NewHandler(svc *PaymentService, credSvc *providercredentials.CredentialsService) *Handler {
	return &Handler{svc: svc, credSvc: credSvc}
}

// RegisterRoutes mounts payment routes onto a chi sub-router
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/payments", h.List)
	r.Post("/subscriptions/{subscriptionKey}/payments", h.CreatePayment)
	r.Get("/subscriptions/{subscriptionKey}/payments", h.ListBySubscription)
}

// CreatePayment creates a new payment (idempotent via Idempotency-Key header)
func (h *Handler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	// Extract Idempotency-Key header
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		httputil.RespondError(w, r, errors.ErrMissingIdempotencyKey)
		return
	}

	var input CreatePaymentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	input.IdempotencyKey = idempotencyKey

	// Fetch stored provider credentials from database
	tenantID := middleware.TenantIDFromContext(r.Context())
	if tenantID == uuid.Nil {
		httputil.RespondError(w, r, errors.ErrMissingTenantID)
		return
	}

	// If provider_name is provided, fetch stored credentials
	if input.ProviderName != "" {
		storedConfig, err := h.credSvc.GetProviderConfig(r.Context(), tenantID, input.ProviderName)
		if err != nil {
			httputil.RespondError(w, r, err)
			return
		}
		// Merge stored credentials with any request-provided config (request takes precedence)
		if input.ProviderConfig == nil {
			input.ProviderConfig = storedConfig
		} else {
			for k, v := range storedConfig {
				if _, exists := input.ProviderConfig[k]; !exists {
					input.ProviderConfig[k] = v
				}
			}
		}
	}

	payment, created, err := h.svc.CreatePayment(r.Context(), input)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	if created {
		httputil.RespondCreated(w, r, payment)
	} else {
		httputil.RespondOK(w, payment)
	}
}

// List returns a paginated list of all payments for the authenticated tenant
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	limit := 20
	cursor := r.URL.Query().Get("cursor")

	var providerPtr *string
	if provider != "" {
		providerPtr = &provider
	}

	input := ListPaymentsInput{
		ProviderName: providerPtr,
		Limit:        limit,
		Cursor:       cursor,
	}

	result, err := h.svc.List(r.Context(), input)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	httputil.RespondOK(w, result)
}

// ListBySubscription returns paginated payments for a specific subscription
func (h *Handler) ListBySubscription(w http.ResponseWriter, r *http.Request) {
	subscriptionKey := chi.URLParam(r, "subscriptionKey")
	if subscriptionKey == "" {
		httputil.RespondError(w, r, errors.ErrInvalidInput)
		return
	}

	// Parse subscription key as UUID
	subscriptionID, err := uuid.Parse(subscriptionKey)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if err := json.Unmarshal([]byte(l), &limit); err != nil {
			limit = 20
		}
	}

	payments, err := h.svc.ListBySubscription(r.Context(), subscriptionID, limit)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	result := map[string]any{
		"payments": payments,
	}
	httputil.RespondOK(w, result)
}
