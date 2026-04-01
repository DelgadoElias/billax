package subscription

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/DelgadoElias/billax/internal/payment"
	"github.com/DelgadoElias/billax/internal/plan"
)

// Status represents subscription lifecycle state
type Status string

const (
	StatusTrialing  Status = "trialing"
	StatusActive    Status = "active"
	StatusPastDue   Status = "past_due"
	StatusCanceled  Status = "canceled"
	StatusExpired   Status = "expired"
)

// Subscription is the domain model
type Subscription struct {
	ID                     uuid.UUID       `json:"id"`
	TenantID               uuid.UUID       `json:"tenant_id"`
	PlanID                 *uuid.UUID      `json:"plan_id,omitempty"`            // nullable (nil = planless)
	Amount                 int64           `json:"amount"`                       // centavos; source of truth for charging
	Currency               string          `json:"currency"`                     // ISO 4217
	Interval               plan.Interval   `json:"interval"`                     // day|week|month|year
	IntervalCount          int             `json:"interval_count"`
	SubscriptionKey        uuid.UUID       `json:"subscription_key"`             // UUIDv7, stable external ID
	ExternalCustomerID     string          `json:"external_customer_id,omitempty"`
	Status                 Status          `json:"status"`
	ProviderName           string          `json:"provider_name,omitempty"`
	ProviderSubscriptionID string          `json:"provider_subscription_id,omitempty"`
	CurrentPeriodStart     time.Time       `json:"current_period_start"`
	CurrentPeriodEnd       time.Time       `json:"current_period_end"`
	TrialEndsAt            *time.Time      `json:"trial_ends_at,omitempty"`
	CanceledAt             *time.Time      `json:"canceled_at,omitempty"`
	CancelAtPeriodEnd      bool            `json:"cancel_at_period_end"`
	Tags                   []string        `json:"tags"`                         // first-class, filterable
	Metadata               json.RawMessage `json:"metadata,omitempty"`           // arbitrary JSON
	Payments               []payment.Payment `json:"payments,omitempty"`         // enriched view (last 10)
	CreatedAt              time.Time       `json:"created_at"`
	UpdatedAt              time.Time       `json:"updated_at"`
}

// CreateSubscriptionInput is what the service accepts from the handler
type CreateSubscriptionInput struct {
	// Plan-based path
	PlanSlug string `json:"plan_slug"` // optional; planless if empty
	// Planless path (required when PlanSlug is empty)
	Amount       int64          `json:"amount"`
	Currency     string         `json:"currency"`
	Interval     plan.Interval  `json:"interval"`
	IntervalCount int           `json:"interval_count"`
	// Common
	ExternalCustomerID     string          `json:"external_customer_id"`
	ProviderName           string          `json:"provider_name"`
	ProviderSubscriptionID string          `json:"provider_subscription_id"`
	Tags                   []string        `json:"tags"`
	Metadata               json.RawMessage `json:"metadata"`
}

// UpdateSubscriptionInput for PATCH
type UpdateSubscriptionInput struct {
	Status            *Status         `json:"status"`
	CancelAtPeriodEnd *bool           `json:"cancel_at_period_end"`
	Amount            *int64          `json:"amount"`               // nil = no change (pay-per-use gate in service)
	Tags              []string        `json:"tags"`                 // nil = no change, [] = clear
	TagsProvided      bool            `json:"-"`                   // set by handler when tags key present
	Metadata          json.RawMessage `json:"metadata"`            // null = clear
}

// ListSubscriptionsInput for cursor-based pagination + tag filtering
type ListSubscriptionsInput struct {
	Status []Status  // optional multi-value filter
	Tags   []string  // filter: subscriptions that contain ALL of these tags (AND semantics)
	Limit  int       // default 20, max 100
	Cursor string    // opaque cursor
}

// ListSubscriptionsResult wraps paginated results
type ListSubscriptionsResult struct {
	Subscriptions []Subscription `json:"subscriptions"`
	NextCursor    string         `json:"next_cursor,omitempty"`
}
