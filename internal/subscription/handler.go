package subscription

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/DelgadoElias/billax/internal/errors"
	"github.com/DelgadoElias/billax/internal/httputil"
)

type Handler struct {
	svc *SubscriptionService
}

func NewHandler(svc *SubscriptionService) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts subscription routes onto a chi sub-router
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/subscriptions", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Get("/{subscriptionKey}", h.GetByKey)
		r.Patch("/{subscriptionKey}", h.Update)
		r.Post("/{subscriptionKey}/cancel", h.Cancel)
	})
}

// Create creates a new subscription
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	// Check for Idempotency-Key header
	if r.Header.Get("Idempotency-Key") == "" {
		httputil.RespondError(w, r, errors.ErrMissingIdempotencyKey)
		return
	}

	var input CreateSubscriptionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	sub, err := h.svc.Create(r.Context(), input)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	httputil.RespondCreated(w, r, sub)
}

// List returns a paginated list of subscriptions
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	// Parse filters
	statusParams := r.URL.Query()["status"]
	tagParams := r.URL.Query()["tag"]

	limit := 20
	cursor := r.URL.Query().Get("cursor")

	var statuses []Status
	for _, s := range statusParams {
		statuses = append(statuses, Status(s))
	}

	input := ListSubscriptionsInput{
		Status: statuses,
		Tags:   tagParams,
		Limit:  limit,
		Cursor: cursor,
	}

	result, err := h.svc.List(r.Context(), input)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	httputil.RespondOK(w, result)
}

// GetByKey retrieves a subscription by its external key
func (h *Handler) GetByKey(w http.ResponseWriter, r *http.Request) {
	subscriptionKey := chi.URLParam(r, "subscriptionKey")
	if subscriptionKey == "" {
		httputil.RespondError(w, r, errors.ErrInvalidInput)
		return
	}

	key, err := uuid.Parse(subscriptionKey)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	sub, err := h.svc.GetByKey(r.Context(), key)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	httputil.RespondOK(w, sub)
}

// Update patches a subscription
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	subscriptionKey := chi.URLParam(r, "subscriptionKey")
	if subscriptionKey == "" {
		httputil.RespondError(w, r, errors.ErrInvalidInput)
		return
	}

	key, err := uuid.Parse(subscriptionKey)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	// Get subscription by key first to find its ID
	sub, err := h.svc.GetByKey(r.Context(), key)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	// Parse update input — need to detect if tags key is present
	var rawInput map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&rawInput); err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	var input UpdateSubscriptionInput

	// Manually unmarshal fields, tracking which ones are present
	for key, rawValue := range rawInput {
		switch key {
		case "status":
			var s Status
			if err := json.Unmarshal(rawValue, &s); err == nil {
				input.Status = &s
			}
		case "cancel_at_period_end":
			var b bool
			if err := json.Unmarshal(rawValue, &b); err == nil {
				input.CancelAtPeriodEnd = &b
			}
		case "amount":
			var amt int64
			if err := json.Unmarshal(rawValue, &amt); err == nil {
				input.Amount = &amt
			}
		case "tags":
			var tags []string
			if err := json.Unmarshal(rawValue, &tags); err == nil {
				input.Tags = tags
				input.TagsProvided = true
			}
		case "metadata":
			input.Metadata = rawValue
		}
	}

	updatedSub, err := h.svc.Update(r.Context(), sub.ID, input)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	httputil.RespondOK(w, updatedSub)
}

// Cancel cancels a subscription
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	subscriptionKey := chi.URLParam(r, "subscriptionKey")
	if subscriptionKey == "" {
		httputil.RespondError(w, r, errors.ErrInvalidInput)
		return
	}

	key, err := uuid.Parse(subscriptionKey)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	var input struct {
		AtPeriodEnd bool `json:"at_period_end"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		// Default to immediate cancellation if no body
		input.AtPeriodEnd = false
	}

	sub, err := h.svc.Cancel(r.Context(), key, input.AtPeriodEnd)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	httputil.RespondOK(w, sub)
}
