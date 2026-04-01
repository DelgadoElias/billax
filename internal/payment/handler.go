package payment

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/DelgadoElias/billax/internal/httputil"
)

type Handler struct {
	svc *PaymentService
}

func NewHandler(svc *PaymentService) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts payment routes onto a chi sub-router
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/payments", h.List)
	r.Route("/subscriptions/{subscriptionKey}", func(r chi.Router) {
		r.Post("/payments", h.CreatePayment)
		r.Get("/payments", h.ListBySubscription)
	})
}

// CreatePayment creates a new payment (idempotent via Idempotency-Key header)
func (h *Handler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	// Extract Idempotency-Key header
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		httputil.RespondError(w, r, nil)
		return
	}

	var input CreatePaymentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	input.IdempotencyKey = idempotencyKey

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
		httputil.RespondError(w, r, nil)
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
		if _, err := json.Unmarshal([]byte(l), &limit); err != nil {
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
