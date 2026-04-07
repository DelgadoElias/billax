package subscription

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/DelgadoElias/billax/internal/errors"
)

// SubscriptionRepo is the interface the service depends on
type SubscriptionRepo interface {
	Create(ctx context.Context, sub Subscription) (Subscription, error)
	GetByKey(ctx context.Context, key uuid.UUID) (Subscription, error)
	GetByID(ctx context.Context, id uuid.UUID) (Subscription, error)
	Update(ctx context.Context, id uuid.UUID, input UpdateSubscriptionInput) (Subscription, error)
	List(ctx context.Context, tenantID uuid.UUID, input ListSubscriptionsInput) (ListSubscriptionsResult, error)
	Cancel(ctx context.Context, id uuid.UUID, atPeriodEnd bool) (Subscription, error)
	CountByStatus(ctx context.Context) (map[Status]int64, error)
}

type postgresRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) SubscriptionRepo {
	return &postgresRepository{pool: pool}
}

// scanSubscription is a helper to scan subscription rows
func scanSubscription(row pgx.Row) (Subscription, error) {
	var sub Subscription
	var metadataJSON json.RawMessage

	err := row.Scan(
		&sub.ID, &sub.TenantID, &sub.PlanID, &sub.Amount, &sub.Currency, &sub.Interval, &sub.IntervalCount,
		&sub.SubscriptionKey, &sub.ExternalCustomerID,
		&sub.Status, &sub.ProviderName, &sub.ProviderSubscriptionID,
		&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.TrialEndsAt, &sub.CanceledAt,
		&sub.CancelAtPeriodEnd, &sub.Tags, &metadataJSON, &sub.CreatedAt, &sub.UpdatedAt,
	)

	if err != nil {
		return Subscription{}, err
	}

	if len(metadataJSON) > 0 {
		sub.Metadata = metadataJSON
	}

	return sub, nil
}

// Create creates a new subscription
func (r *postgresRepository) Create(ctx context.Context, sub Subscription) (Subscription, error) {
	metadataJSON, _ := json.Marshal(sub.Metadata)

	var createdSub Subscription
	var metadataResult json.RawMessage

	err := r.pool.QueryRow(ctx,
		`INSERT INTO subscriptions (tenant_id, plan_id, amount, currency, interval, interval_count, subscription_key, external_customer_id, status, provider_name, provider_subscription_id, current_period_start, current_period_end, trial_ends_at, canceled_at, cancel_at_period_end, tags, metadata, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, now(), now())
		 RETURNING id, tenant_id, plan_id, amount, currency, interval, interval_count, subscription_key, external_customer_id, status, provider_name, provider_subscription_id, current_period_start, current_period_end, trial_ends_at, canceled_at, cancel_at_period_end, tags, metadata, created_at, updated_at`,
		sub.TenantID, sub.PlanID, sub.Amount, sub.Currency, sub.Interval, sub.IntervalCount, sub.SubscriptionKey, sub.ExternalCustomerID, sub.Status, sub.ProviderName, sub.ProviderSubscriptionID,
		sub.CurrentPeriodStart, sub.CurrentPeriodEnd, sub.TrialEndsAt, sub.CanceledAt, sub.CancelAtPeriodEnd, sub.Tags, metadataJSON,
	).Scan(
		&createdSub.ID, &createdSub.TenantID, &createdSub.PlanID, &createdSub.Amount, &createdSub.Currency, &createdSub.Interval, &createdSub.IntervalCount,
		&createdSub.SubscriptionKey, &createdSub.ExternalCustomerID,
		&createdSub.Status, &createdSub.ProviderName, &createdSub.ProviderSubscriptionID,
		&createdSub.CurrentPeriodStart, &createdSub.CurrentPeriodEnd, &createdSub.TrialEndsAt, &createdSub.CanceledAt,
		&createdSub.CancelAtPeriodEnd, &createdSub.Tags, &metadataResult, &createdSub.CreatedAt, &createdSub.UpdatedAt,
	)

	if err != nil {
		return Subscription{}, fmt.Errorf("create subscription: %w", err)
	}

	if len(metadataResult) > 0 {
		createdSub.Metadata = metadataResult
	}

	return createdSub, nil
}

// GetByKey retrieves a subscription by its stable external key
func (r *postgresRepository) GetByKey(ctx context.Context, key uuid.UUID) (Subscription, error) {
	var sub Subscription
	var metadataJSON json.RawMessage

	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, plan_id, amount, currency, interval, interval_count, subscription_key, external_customer_id, status, provider_name, provider_subscription_id, current_period_start, current_period_end, trial_ends_at, canceled_at, cancel_at_period_end, tags, metadata, created_at, updated_at
		 FROM subscriptions WHERE subscription_key = $1`,
		key,
	).Scan(
		&sub.ID, &sub.TenantID, &sub.PlanID, &sub.Amount, &sub.Currency, &sub.Interval, &sub.IntervalCount,
		&sub.SubscriptionKey, &sub.ExternalCustomerID,
		&sub.Status, &sub.ProviderName, &sub.ProviderSubscriptionID,
		&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.TrialEndsAt, &sub.CanceledAt,
		&sub.CancelAtPeriodEnd, &sub.Tags, &metadataJSON, &sub.CreatedAt, &sub.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return Subscription{}, apperrors.ErrNotFound
		}
		return Subscription{}, fmt.Errorf("get subscription by key: %w", err)
	}

	if len(metadataJSON) > 0 {
		sub.Metadata = metadataJSON
	}

	return sub, nil
}

// GetByID retrieves a subscription by its UUID
func (r *postgresRepository) GetByID(ctx context.Context, id uuid.UUID) (Subscription, error) {
	var sub Subscription
	var metadataJSON json.RawMessage

	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, plan_id, amount, currency, interval, interval_count, subscription_key, external_customer_id, status, provider_name, provider_subscription_id, current_period_start, current_period_end, trial_ends_at, canceled_at, cancel_at_period_end, tags, metadata, created_at, updated_at
		 FROM subscriptions WHERE id = $1`,
		id,
	).Scan(
		&sub.ID, &sub.TenantID, &sub.PlanID, &sub.Amount, &sub.Currency, &sub.Interval, &sub.IntervalCount,
		&sub.SubscriptionKey, &sub.ExternalCustomerID,
		&sub.Status, &sub.ProviderName, &sub.ProviderSubscriptionID,
		&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.TrialEndsAt, &sub.CanceledAt,
		&sub.CancelAtPeriodEnd, &sub.Tags, &metadataJSON, &sub.CreatedAt, &sub.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return Subscription{}, apperrors.ErrNotFound
		}
		return Subscription{}, fmt.Errorf("get subscription by id: %w", err)
	}

	if len(metadataJSON) > 0 {
		sub.Metadata = metadataJSON
	}

	return sub, nil
}

// Update applies partial updates to a subscription
func (r *postgresRepository) Update(ctx context.Context, id uuid.UUID, input UpdateSubscriptionInput) (Subscription, error) {
	var sub Subscription
	var metadataJSON json.RawMessage

	// Build dynamic UPDATE clause
	updates := []string{"updated_at = now()"}
	args := []any{id}
	argNum := 2

	if input.Status != nil {
		updates = append(updates, fmt.Sprintf("status = $%d", argNum))
		args = append(args, *input.Status)
		argNum++
	}

	if input.CancelAtPeriodEnd != nil {
		updates = append(updates, fmt.Sprintf("cancel_at_period_end = $%d", argNum))
		args = append(args, *input.CancelAtPeriodEnd)
		argNum++
	}

	if input.TagsProvided {
		updates = append(updates, fmt.Sprintf("tags = $%d", argNum))
		args = append(args, input.Tags)
		argNum++
	}

	// Handle metadata: null value = clear, non-null = set
	if input.Amount != nil {
		updates = append(updates, fmt.Sprintf("amount = $%d", argNum))
		args = append(args, *input.Amount)
		argNum++
	}

	if input.Metadata != nil {
		if len(input.Metadata) == 0 || string(input.Metadata) == "null" {
			updates = append(updates, fmt.Sprintf("metadata = $%d", argNum))
			args = append(args, nil)
		} else {
			updates = append(updates, fmt.Sprintf("metadata = $%d", argNum))
			args = append(args, input.Metadata)
		}
		argNum++
	}

	query := fmt.Sprintf(
		`UPDATE subscriptions SET %s WHERE id = $1 RETURNING id, tenant_id, plan_id, amount, currency, interval, interval_count, subscription_key, external_customer_id, status, provider_name, provider_subscription_id, current_period_start, current_period_end, trial_ends_at, canceled_at, cancel_at_period_end, tags, metadata, created_at, updated_at`,
		strings.Join(updates, ", "),
	)

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&sub.ID, &sub.TenantID, &sub.PlanID, &sub.Amount, &sub.Currency, &sub.Interval, &sub.IntervalCount,
		&sub.SubscriptionKey, &sub.ExternalCustomerID,
		&sub.Status, &sub.ProviderName, &sub.ProviderSubscriptionID,
		&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.TrialEndsAt, &sub.CanceledAt,
		&sub.CancelAtPeriodEnd, &sub.Tags, &metadataJSON, &sub.CreatedAt, &sub.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return Subscription{}, apperrors.ErrNotFound
		}
		return Subscription{}, fmt.Errorf("update subscription: %w", err)
	}

	if len(metadataJSON) > 0 {
		sub.Metadata = metadataJSON
	}

	return sub, nil
}

// List returns a cursor-paginated, optionally filtered list of subscriptions
func (r *postgresRepository) List(ctx context.Context, tenantID uuid.UUID, input ListSubscriptionsInput) (ListSubscriptionsResult, error) {
	limit := input.Limit
	if limit == 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// Decode cursor if provided
	var cursorTime time.Time
	var cursorID uuid.UUID

	if input.Cursor != "" {
		decoded, err := base64.URLEncoding.DecodeString(input.Cursor)
		if err != nil {
			return ListSubscriptionsResult{}, apperrors.ErrInvalidInput
		}

		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			return ListSubscriptionsResult{}, apperrors.ErrInvalidInput
		}

		cursorTime, err = time.Parse(time.RFC3339Nano, parts[0])
		if err != nil {
			return ListSubscriptionsResult{}, apperrors.ErrInvalidInput
		}

		cursorID, err = uuid.Parse(parts[1])
		if err != nil {
			return ListSubscriptionsResult{}, apperrors.ErrInvalidInput
		}
	}

	// Build WHERE clause
	whereClause := "tenant_id = $1"
	args := []any{tenantID}
	argNum := 2

	if len(input.Status) > 0 {
		whereClause += fmt.Sprintf(" AND status = ANY($%d)", argNum)
		args = append(args, input.Status)
		argNum++
	}

	if len(input.Tags) > 0 {
		whereClause += fmt.Sprintf(" AND tags @> $%d", argNum)
		args = append(args, input.Tags)
		argNum++
	}

	if input.Cursor != "" {
		whereClause += fmt.Sprintf(" AND (created_at, id) > ($%d, $%d)", argNum, argNum+1)
		args = append(args, cursorTime, cursorID)
		argNum += 2
	}

	query := fmt.Sprintf(
		`SELECT id, tenant_id, plan_id, amount, currency, interval, interval_count, subscription_key, external_customer_id, status, provider_name, provider_subscription_id, current_period_start, current_period_end, trial_ends_at, canceled_at, cancel_at_period_end, tags, metadata, created_at, updated_at
		 FROM subscriptions
		 WHERE %s
		 ORDER BY created_at ASC, id ASC
		 LIMIT %d`,
		whereClause, limit+1,
	)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return ListSubscriptionsResult{}, fmt.Errorf("list subscriptions: %w", err)
	}
	defer rows.Close()

	subscriptions := []Subscription{}
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return ListSubscriptionsResult{}, fmt.Errorf("scan subscription: %w", err)
		}
		subscriptions = append(subscriptions, sub)
	}

	if err = rows.Err(); err != nil {
		return ListSubscriptionsResult{}, fmt.Errorf("rows error: %w", err)
	}

	// Check if there are more results
	result := ListSubscriptionsResult{Subscriptions: subscriptions}
	if len(subscriptions) > limit {
		// Remove the extra row
		result.Subscriptions = subscriptions[:limit]
		// Set next cursor from the last row
		lastSub := subscriptions[limit]
		cursorStr := fmt.Sprintf("%s:%s", lastSub.CreatedAt.UTC().Format(time.RFC3339Nano), lastSub.ID)
		result.NextCursor = base64.URLEncoding.EncodeToString([]byte(cursorStr))
	}

	return result, nil
}

// Cancel cancels a subscription
func (r *postgresRepository) Cancel(ctx context.Context, id uuid.UUID, atPeriodEnd bool) (Subscription, error) {
	var sub Subscription
	var metadataJSON json.RawMessage

	var canceledAt *time.Time
	if !atPeriodEnd {
		now := time.Now()
		canceledAt = &now
	}

	updateClause := `cancel_at_period_end = true, updated_at = now()`
	if !atPeriodEnd {
		updateClause = fmt.Sprintf(`status = $2, canceled_at = $3, cancel_at_period_end = false, updated_at = now()`)
	}

	query := fmt.Sprintf(
		`UPDATE subscriptions SET %s WHERE id = $1 RETURNING id, tenant_id, plan_id, amount, currency, interval, interval_count, subscription_key, external_customer_id, status, provider_name, provider_subscription_id, current_period_start, current_period_end, trial_ends_at, canceled_at, cancel_at_period_end, tags, metadata, created_at, updated_at`,
		updateClause,
	)

	var err error
	if !atPeriodEnd {
		err = r.pool.QueryRow(ctx, query, id, StatusCanceled, canceledAt).Scan(
			&sub.ID, &sub.TenantID, &sub.PlanID, &sub.Amount, &sub.Currency, &sub.Interval, &sub.IntervalCount,
			&sub.SubscriptionKey, &sub.ExternalCustomerID,
			&sub.Status, &sub.ProviderName, &sub.ProviderSubscriptionID,
			&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.TrialEndsAt, &sub.CanceledAt,
			&sub.CancelAtPeriodEnd, &sub.Tags, &metadataJSON, &sub.CreatedAt, &sub.UpdatedAt,
		)
	} else {
		err = r.pool.QueryRow(ctx, query, id).Scan(
			&sub.ID, &sub.TenantID, &sub.PlanID, &sub.Amount, &sub.Currency, &sub.Interval, &sub.IntervalCount,
			&sub.SubscriptionKey, &sub.ExternalCustomerID,
			&sub.Status, &sub.ProviderName, &sub.ProviderSubscriptionID,
			&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.TrialEndsAt, &sub.CanceledAt,
			&sub.CancelAtPeriodEnd, &sub.Tags, &metadataJSON, &sub.CreatedAt, &sub.UpdatedAt,
		)
	}

	if err != nil {
		if err == pgx.ErrNoRows {
			return Subscription{}, apperrors.ErrNotFound
		}
		return Subscription{}, fmt.Errorf("cancel subscription: %w", err)
	}

	if len(metadataJSON) > 0 {
		sub.Metadata = metadataJSON
	}

	return sub, nil
}

// CountByStatus returns the count of subscriptions grouped by status
// This is used by metrics collection and does not filter by tenant
func (r *postgresRepository) CountByStatus(ctx context.Context) (map[Status]int64, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT status, COUNT(*) FROM subscriptions GROUP BY status`,
	)
	if err != nil {
		return nil, fmt.Errorf("count subscriptions by status: %w", err)
	}
	defer rows.Close()

	counts := make(map[Status]int64)
	for rows.Next() {
		var status Status
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan status count: %w", err)
		}
		counts[status] = count
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return counts, nil
}
