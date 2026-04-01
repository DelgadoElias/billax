package plan

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/DelgadoElias/billax/internal/errors"
)

// PlanRepo is the interface the service depends on (enables mocking in unit tests)
type PlanRepo interface {
	Upsert(ctx context.Context, tenantID uuid.UUID, input CreatePlanInput) (UpsertResult, error)
	GetByID(ctx context.Context, id uuid.UUID) (Plan, error)
	GetBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (Plan, error)
	Update(ctx context.Context, id uuid.UUID, input UpdatePlanInput) (Plan, error)
	List(ctx context.Context, tenantID uuid.UUID, input ListPlansInput) (ListPlansResult, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type postgresRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) PlanRepo {
	return &postgresRepository{pool: pool}
}

// Upsert implements idempotent plan creation by slug using ON CONFLICT
func (r *postgresRepository) Upsert(ctx context.Context, tenantID uuid.UUID, input CreatePlanInput) (UpsertResult, error) {
	var plan Plan
	var wasInserted bool

	err := r.pool.QueryRow(ctx,
		`INSERT INTO plans (tenant_id, slug, name, description, amount, currency, interval, interval_count, trial_days, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, true, now(), now())
		 ON CONFLICT (tenant_id, slug) DO UPDATE SET updated_at = now()
		 RETURNING id, tenant_id, slug, name, description, amount, currency, interval, interval_count, trial_days, is_active, created_at, updated_at, (xmax = 0) AS was_inserted`,
		tenantID, input.Slug, input.Name, input.Description, input.Amount, input.Currency, input.Interval, input.IntervalCount, input.TrialDays,
	).Scan(&plan.ID, &plan.TenantID, &plan.Slug, &plan.Name, &plan.Description, &plan.Amount, &plan.Currency,
		&plan.Interval, &plan.IntervalCount, &plan.TrialDays, &plan.IsActive, &plan.CreatedAt, &plan.UpdatedAt, &wasInserted)

	if err != nil {
		return UpsertResult{}, fmt.Errorf("upsert plan: %w", err)
	}

	return UpsertResult{Plan: plan, Created: wasInserted}, nil
}

// GetByID retrieves a plan by its UUID
func (r *postgresRepository) GetByID(ctx context.Context, id uuid.UUID) (Plan, error) {
	var plan Plan

	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, slug, name, description, amount, currency, interval, interval_count, trial_days, is_active, created_at, updated_at
		 FROM plans WHERE id = $1`,
		id,
	).Scan(&plan.ID, &plan.TenantID, &plan.Slug, &plan.Name, &plan.Description, &plan.Amount, &plan.Currency,
		&plan.Interval, &plan.IntervalCount, &plan.TrialDays, &plan.IsActive, &plan.CreatedAt, &plan.UpdatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return Plan{}, apperrors.ErrNotFound
		}
		return Plan{}, fmt.Errorf("get plan by id: %w", err)
	}

	return plan, nil
}

// GetBySlug retrieves a plan by its slug, scoped to the tenant
func (r *postgresRepository) GetBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (Plan, error) {
	var plan Plan

	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, slug, name, description, amount, currency, interval, interval_count, trial_days, is_active, created_at, updated_at
		 FROM plans WHERE tenant_id = $1 AND slug = $2`,
		tenantID, slug,
	).Scan(&plan.ID, &plan.TenantID, &plan.Slug, &plan.Name, &plan.Description, &plan.Amount, &plan.Currency,
		&plan.Interval, &plan.IntervalCount, &plan.TrialDays, &plan.IsActive, &plan.CreatedAt, &plan.UpdatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return Plan{}, apperrors.ErrNotFound
		}
		return Plan{}, fmt.Errorf("get plan by slug: %w", err)
	}

	return plan, nil
}

// Update applies partial updates to a plan
func (r *postgresRepository) Update(ctx context.Context, id uuid.UUID, input UpdatePlanInput) (Plan, error) {
	var plan Plan

	// Build dynamic UPDATE clause
	updates := []string{"updated_at = now()"}
	args := []any{id}
	argNum := 2

	if input.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argNum))
		args = append(args, *input.Name)
		argNum++
	}

	if input.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argNum))
		args = append(args, *input.Description)
		argNum++
	}

	if input.IsActive != nil {
		updates = append(updates, fmt.Sprintf("is_active = $%d", argNum))
		args = append(args, *input.IsActive)
		argNum++
	}

	query := fmt.Sprintf(
		`UPDATE plans SET %s WHERE id = $1 RETURNING id, tenant_id, slug, name, description, amount, currency, interval, interval_count, trial_days, is_active, created_at, updated_at`,
		strings.Join(updates, ", "),
	)

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&plan.ID, &plan.TenantID, &plan.Slug, &plan.Name, &plan.Description, &plan.Amount, &plan.Currency,
		&plan.Interval, &plan.IntervalCount, &plan.TrialDays, &plan.IsActive, &plan.CreatedAt, &plan.UpdatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return Plan{}, apperrors.ErrNotFound
		}
		return Plan{}, fmt.Errorf("update plan: %w", err)
	}

	return plan, nil
}

// List returns a cursor-paginated list of plans
func (r *postgresRepository) List(ctx context.Context, tenantID uuid.UUID, input ListPlansInput) (ListPlansResult, error) {
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
			return ListPlansResult{}, apperrors.ErrInvalidInput
		}

		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			return ListPlansResult{}, apperrors.ErrInvalidInput
		}

		cursorTime, err = time.Parse(time.RFC3339Nano, parts[0])
		if err != nil {
			return ListPlansResult{}, apperrors.ErrInvalidInput
		}

		cursorID, err = uuid.Parse(parts[1])
		if err != nil {
			return ListPlansResult{}, apperrors.ErrInvalidInput
		}
	}

	// Build WHERE clause for isActive filter
	whereClause := "tenant_id = $1"
	args := []any{tenantID}
	argNum := 2

	if input.IsActive != nil {
		whereClause += fmt.Sprintf(" AND is_active = $%d", argNum)
		args = append(args, *input.IsActive)
		argNum++
	}

	if input.Cursor != "" {
		whereClause += fmt.Sprintf(" AND (created_at, id) > ($%d, $%d)", argNum, argNum+1)
		args = append(args, cursorTime, cursorID)
		argNum += 2
	}

	// Fetch limit+1 rows to detect if there are more
	query := fmt.Sprintf(
		`SELECT id, tenant_id, slug, name, description, amount, currency, interval, interval_count, trial_days, is_active, created_at, updated_at
		 FROM plans
		 WHERE %s
		 ORDER BY created_at ASC, id ASC
		 LIMIT %d`,
		whereClause, limit+1,
	)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return ListPlansResult{}, fmt.Errorf("list plans: %w", err)
	}
	defer rows.Close()

	plans := []Plan{}
	for rows.Next() {
		var plan Plan
		err := rows.Scan(&plan.ID, &plan.TenantID, &plan.Slug, &plan.Name, &plan.Description, &plan.Amount, &plan.Currency,
			&plan.Interval, &plan.IntervalCount, &plan.TrialDays, &plan.IsActive, &plan.CreatedAt, &plan.UpdatedAt)
		if err != nil {
			return ListPlansResult{}, fmt.Errorf("scan plan: %w", err)
		}
		plans = append(plans, plan)
	}

	if err = rows.Err(); err != nil {
		return ListPlansResult{}, fmt.Errorf("rows error: %w", err)
	}

	// Check if there are more results
	result := ListPlansResult{Plans: plans}
	if len(plans) > limit {
		// Remove the extra row
		result.Plans = plans[:limit]
		// Set next cursor from the last row
		lastPlan := plans[limit]
		cursorStr := fmt.Sprintf("%s:%s", lastPlan.CreatedAt.UTC().Format(time.RFC3339Nano), lastPlan.ID)
		result.NextCursor = base64.URLEncoding.EncodeToString([]byte(cursorStr))
	}

	return result, nil
}

// Delete soft-deletes a plan by setting is_active = false
func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE plans SET is_active = false, updated_at = now() WHERE id = $1`,
		id,
	)

	if err != nil {
		return fmt.Errorf("delete plan: %w", err)
	}

	return nil
}
