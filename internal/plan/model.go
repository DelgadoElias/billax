package plan

import (
	"time"

	"github.com/google/uuid"
)

// Interval represents billing frequency
type Interval string

const (
	IntervalDay   Interval = "day"
	IntervalWeek  Interval = "week"
	IntervalMonth Interval = "month"
	IntervalYear  Interval = "year"
)

// Plan is the domain model (matches DB row)
type Plan struct {
	ID            uuid.UUID `json:"id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	Slug          string    `json:"slug"`
	Name          string    `json:"name"`
	Description   string    `json:"description,omitempty"`
	Amount        int64     `json:"amount"`         // centavos — NEVER float
	Currency      string    `json:"currency"`       // ISO 4217
	Interval      Interval  `json:"interval"`
	IntervalCount int       `json:"interval_count"`
	TrialDays     int       `json:"trial_days"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CreatePlanInput is what the service accepts from the handler
type CreatePlanInput struct {
	Slug          string   `json:"slug"`           // required; unique per tenant
	Name          string   `json:"name"`           // required
	Description   string   `json:"description"`
	Amount        int64    `json:"amount"`         // required; > 0
	Currency      string   `json:"currency"`       // default "ARS"
	Interval      Interval `json:"interval"`       // required
	IntervalCount int      `json:"interval_count"` // default 1
	TrialDays     int      `json:"trial_days"`     // default 0
}

// UpdatePlanInput for PATCH — all fields optional (pointer = not provided)
type UpdatePlanInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"is_active"`
}

// ListPlansInput for cursor-based pagination + filtering
type ListPlansInput struct {
	IsActive *bool
	Limit    int    // default 20, max 100
	Cursor   string // opaque base64(created_at:id)
}

// ListPlansResult wraps paginated results
type ListPlansResult struct {
	Plans      []Plan `json:"plans"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// UpsertResult carries the plan plus whether it was newly created
type UpsertResult struct {
	Plan    Plan
	Created bool
}
