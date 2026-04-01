package plan

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/DelgadoElias/billax/internal/httputil"
)

type Handler struct {
	svc *PlanService
}

func NewHandler(svc *PlanService) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts plan routes onto a chi sub-router
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/plans", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.CreateOrGet)
		r.Get("/{planID}", h.GetByID)
		r.Get("/slug/{slug}", h.GetBySlug)
		r.Patch("/{planID}", h.Update)
		r.Delete("/{planID}", h.Delete)
	})
}

// CreateOrGet creates a new plan or returns the existing one (idempotent by slug)
func (h *Handler) CreateOrGet(w http.ResponseWriter, r *http.Request) {
	var input CreatePlanInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	plan, created, err := h.svc.CreateOrGet(r.Context(), input)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	if created {
		httputil.RespondCreated(w, r, plan)
	} else {
		httputil.RespondOK(w, plan)
	}
}

// List returns a paginated list of plans
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	// Parse query params
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if err := json.Unmarshal([]byte(l), &limit); err != nil {
			limit = 20
		}
	}

	cursor := r.URL.Query().Get("cursor")
	isActive := r.URL.Query().Get("is_active")
	var isActivePtr *bool
	if isActive == "true" {
		t := true
		isActivePtr = &t
	} else if isActive == "false" {
		f := false
		isActivePtr = &f
	}

	input := ListPlansInput{
		Limit:    limit,
		Cursor:   cursor,
		IsActive: isActivePtr,
	}

	result, err := h.svc.List(r.Context(), input)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	httputil.RespondOK(w, result)
}

// GetByID retrieves a plan by UUID
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	planID := chi.URLParam(r, "planID")
	id, err := uuid.Parse(planID)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	plan, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	httputil.RespondOK(w, plan)
}

// GetBySlug retrieves a plan by slug
func (h *Handler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		httputil.RespondError(w, r, nil)
		return
	}

	plan, err := h.svc.GetBySlug(r.Context(), slug)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	httputil.RespondOK(w, plan)
}

// Update patches a plan
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	planID := chi.URLParam(r, "planID")
	id, err := uuid.Parse(planID)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	var input UpdatePlanInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	plan, err := h.svc.Update(r.Context(), id, input)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	httputil.RespondOK(w, plan)
}

// Delete soft-deletes a plan
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	planID := chi.URLParam(r, "planID")
	id, err := uuid.Parse(planID)
	if err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		httputil.RespondError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
