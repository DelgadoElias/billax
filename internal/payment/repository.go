package payment

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
	"github.com/DelgadoElias/billax/internal/provider"
)

// PaymentRepo is the interface the service depends on
type PaymentRepo interface {
	Create(ctx context.Context, payment Payment) (CreatePaymentResult, error)
	GetByKey(ctx context.Context, tenantID uuid.UUID, idempotencyKey string) (Payment, error)
	GetByID(ctx context.Context, id uuid.UUID) (Payment, error)
	List(ctx context.Context, tenantID uuid.UUID, input ListPaymentsInput) (ListPaymentsResult, error)
	ListBySubscription(ctx context.Context, subscriptionID uuid.UUID, limit int) ([]Payment, error)
}

type postgresRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) PaymentRepo {
	return &postgresRepository{pool: pool}
}

// Create creates a new payment with idempotency via ON CONFLICT DO NOTHING
func (r *postgresRepository) Create(ctx context.Context, payment Payment) (CreatePaymentResult, error) {
	paymentMethodJSON, err := json.Marshal(payment.PaymentMethod)
	if err != nil {
		return CreatePaymentResult{}, fmt.Errorf("marshal payment method: %w", err)
	}

	var createdPayment Payment
	var wasInserted bool

	err = r.pool.QueryRow(ctx,
		`INSERT INTO payments (tenant_id, subscription_id, idempotency_key, provider_name, provider_charge_id, amount, currency, status, failure_reason, payment_method, provider_response, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, now(), now())
		 ON CONFLICT (tenant_id, idempotency_key) DO NOTHING
		 RETURNING id, tenant_id, subscription_id, idempotency_key, provider_name, provider_charge_id, amount, currency, status, failure_reason, payment_method, provider_response, created_at, updated_at, (xmax = 0) AS was_inserted`,
		payment.TenantID, payment.SubscriptionID, payment.IdempotencyKey, payment.ProviderName, payment.ProviderChargeID,
		payment.Amount, payment.Currency, payment.Status, payment.FailureReason, paymentMethodJSON, payment.ProviderResponse,
	).Scan(&createdPayment.ID, &createdPayment.TenantID, &createdPayment.SubscriptionID, &createdPayment.IdempotencyKey,
		&createdPayment.ProviderName, &createdPayment.ProviderChargeID, &createdPayment.Amount, &createdPayment.Currency,
		&createdPayment.Status, &createdPayment.FailureReason, &paymentMethodJSON, &createdPayment.ProviderResponse,
		&createdPayment.CreatedAt, &createdPayment.UpdatedAt, &wasInserted)

	if err != nil {
		if err == pgx.ErrNoRows {
			// Conflict occurred (row already exists) — fetch the existing payment
			existingPayment, err := r.GetByKey(ctx, payment.TenantID, payment.IdempotencyKey)
			if err != nil {
				return CreatePaymentResult{}, fmt.Errorf("fetch existing payment on conflict: %w", err)
			}
			return CreatePaymentResult{Payment: existingPayment, Created: false}, nil
		}
		return CreatePaymentResult{}, fmt.Errorf("create payment: %w", err)
	}

	// Unmarshal payment method
	if len(paymentMethodJSON) > 0 {
		var pm provider.PaymentMethodInfo
		if err := json.Unmarshal(paymentMethodJSON, &pm); err == nil {
			createdPayment.PaymentMethod = &pm
		}
	}

	return CreatePaymentResult{Payment: createdPayment, Created: wasInserted}, nil
}

// GetByKey retrieves a payment by idempotency key
func (r *postgresRepository) GetByKey(ctx context.Context, tenantID uuid.UUID, idempotencyKey string) (Payment, error) {
	var payment Payment
	var paymentMethodJSON json.RawMessage

	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, subscription_id, idempotency_key, provider_name, provider_charge_id, amount, currency, status, failure_reason, payment_method, provider_response, created_at, updated_at
		 FROM payments WHERE tenant_id = $1 AND idempotency_key = $2`,
		tenantID, idempotencyKey,
	).Scan(&payment.ID, &payment.TenantID, &payment.SubscriptionID, &payment.IdempotencyKey,
		&payment.ProviderName, &payment.ProviderChargeID, &payment.Amount, &payment.Currency,
		&payment.Status, &payment.FailureReason, &paymentMethodJSON, &payment.ProviderResponse,
		&payment.CreatedAt, &payment.UpdatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return Payment{}, apperrors.ErrNotFound
		}
		return Payment{}, fmt.Errorf("get payment by key: %w", err)
	}

	// Unmarshal payment method
	if len(paymentMethodJSON) > 0 {
		var pm provider.PaymentMethodInfo
		if err := json.Unmarshal(paymentMethodJSON, &pm); err == nil {
			payment.PaymentMethod = &pm
		}
	}

	return payment, nil
}

// GetByID retrieves a payment by its UUID
func (r *postgresRepository) GetByID(ctx context.Context, id uuid.UUID) (Payment, error) {
	var payment Payment
	var paymentMethodJSON json.RawMessage

	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, subscription_id, idempotency_key, provider_name, provider_charge_id, amount, currency, status, failure_reason, payment_method, provider_response, created_at, updated_at
		 FROM payments WHERE id = $1`,
		id,
	).Scan(&payment.ID, &payment.TenantID, &payment.SubscriptionID, &payment.IdempotencyKey,
		&payment.ProviderName, &payment.ProviderChargeID, &payment.Amount, &payment.Currency,
		&payment.Status, &payment.FailureReason, &paymentMethodJSON, &payment.ProviderResponse,
		&payment.CreatedAt, &payment.UpdatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return Payment{}, apperrors.ErrNotFound
		}
		return Payment{}, fmt.Errorf("get payment by id: %w", err)
	}

	// Unmarshal payment method
	if len(paymentMethodJSON) > 0 {
		var pm provider.PaymentMethodInfo
		if err := json.Unmarshal(paymentMethodJSON, &pm); err == nil {
			payment.PaymentMethod = &pm
		}
	}

	return payment, nil
}

// List returns a cursor-paginated, optionally filtered list of payments
func (r *postgresRepository) List(ctx context.Context, tenantID uuid.UUID, input ListPaymentsInput) (ListPaymentsResult, error) {
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
			return ListPaymentsResult{}, apperrors.ErrInvalidInput
		}

		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			return ListPaymentsResult{}, apperrors.ErrInvalidInput
		}

		cursorTime, err = time.Parse(time.RFC3339Nano, parts[0])
		if err != nil {
			return ListPaymentsResult{}, apperrors.ErrInvalidInput
		}

		cursorID, err = uuid.Parse(parts[1])
		if err != nil {
			return ListPaymentsResult{}, apperrors.ErrInvalidInput
		}
	}

	// Build WHERE clause
	whereClause := "tenant_id = $1"
	args := []any{tenantID}
	argNum := 2

	if input.ProviderName != nil {
		whereClause += fmt.Sprintf(" AND provider_name = $%d", argNum)
		args = append(args, *input.ProviderName)
		argNum++
	}

	if len(input.Status) > 0 {
		whereClause += fmt.Sprintf(" AND status = ANY($%d)", argNum)
		args = append(args, input.Status)
		argNum++
	}

	if input.Cursor != "" {
		whereClause += fmt.Sprintf(" AND (created_at, id) > ($%d, $%d)", argNum, argNum+1)
		args = append(args, cursorTime, cursorID)
		argNum += 2
	}

	query := fmt.Sprintf(
		`SELECT id, tenant_id, subscription_id, idempotency_key, provider_name, provider_charge_id, amount, currency, status, failure_reason, payment_method, provider_response, created_at, updated_at
		 FROM payments
		 WHERE %s
		 ORDER BY created_at DESC, id DESC
		 LIMIT %d`,
		whereClause, limit+1,
	)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return ListPaymentsResult{}, fmt.Errorf("list payments: %w", err)
	}
	defer rows.Close()

	payments := []Payment{}
	for rows.Next() {
		var payment Payment
		var paymentMethodJSON json.RawMessage
		err := rows.Scan(&payment.ID, &payment.TenantID, &payment.SubscriptionID, &payment.IdempotencyKey,
			&payment.ProviderName, &payment.ProviderChargeID, &payment.Amount, &payment.Currency,
			&payment.Status, &payment.FailureReason, &paymentMethodJSON, &payment.ProviderResponse,
			&payment.CreatedAt, &payment.UpdatedAt)
		if err != nil {
			return ListPaymentsResult{}, fmt.Errorf("scan payment: %w", err)
		}

		// Unmarshal payment method
		if len(paymentMethodJSON) > 0 {
			var pm provider.PaymentMethodInfo
			if err := json.Unmarshal(paymentMethodJSON, &pm); err == nil {
				payment.PaymentMethod = &pm
			}
		}

		payments = append(payments, payment)
	}

	if err = rows.Err(); err != nil {
		return ListPaymentsResult{}, fmt.Errorf("rows error: %w", err)
	}

	// Check if there are more results
	result := ListPaymentsResult{Payments: payments}
	if len(payments) > limit {
		// Remove the extra row
		result.Payments = payments[:limit]
		// Set next cursor from the last row
		lastPayment := payments[limit]
		cursorStr := fmt.Sprintf("%s:%s", lastPayment.CreatedAt.UTC().Format(time.RFC3339Nano), lastPayment.ID)
		result.NextCursor = base64.URLEncoding.EncodeToString([]byte(cursorStr))
	}

	return result, nil
}

// ListBySubscription returns the most recent payments for a subscription (for enriched views)
func (r *postgresRepository) ListBySubscription(ctx context.Context, subscriptionID uuid.UUID, limit int) ([]Payment, error) {
	if limit == 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, subscription_id, idempotency_key, provider_name, provider_charge_id, amount, currency, status, failure_reason, payment_method, provider_response, created_at, updated_at
		 FROM payments
		 WHERE subscription_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2`,
		subscriptionID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list by subscription: %w", err)
	}
	defer rows.Close()

	payments := []Payment{}
	for rows.Next() {
		var payment Payment
		var paymentMethodJSON json.RawMessage
		err := rows.Scan(&payment.ID, &payment.TenantID, &payment.SubscriptionID, &payment.IdempotencyKey,
			&payment.ProviderName, &payment.ProviderChargeID, &payment.Amount, &payment.Currency,
			&payment.Status, &payment.FailureReason, &paymentMethodJSON, &payment.ProviderResponse,
			&payment.CreatedAt, &payment.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan payment: %w", err)
		}

		// Unmarshal payment method
		if len(paymentMethodJSON) > 0 {
			var pm provider.PaymentMethodInfo
			if err := json.Unmarshal(paymentMethodJSON, &pm); err == nil {
				payment.PaymentMethod = &pm
			}
		}

		payments = append(payments, payment)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return payments, nil
}
