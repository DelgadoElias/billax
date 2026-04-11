package backoffice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/DelgadoElias/billax/internal/errors"
)

// UserRepo defines operations on backoffice users
type UserRepo interface {
	// Create inserts a new backoffice user
	Create(ctx context.Context, tenantID uuid.UUID, email, name string, passwordHash string, role Role) (User, error)

	// GetByEmail retrieves a user by email (returns user + hash for auth)
	GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (User, string, error)

	// GetByID retrieves a user by ID
	GetByID(ctx context.Context, id uuid.UUID) (User, error)

	// GetPasswordHashByID retrieves only the password hash for a user (for password verification)
	GetPasswordHashByID(ctx context.Context, id uuid.UUID) (string, error)

	// List returns all users for a tenant
	List(ctx context.Context, tenantID uuid.UUID) ([]User, error)

	// UpdatePassword updates a user's password hash
	UpdatePassword(ctx context.Context, id uuid.UUID, newPasswordHash string) error

	// UpdateMustChangePassword updates the must_change_password flag
	UpdateMustChangePassword(ctx context.Context, id uuid.UUID, value bool) error

	// UpdateName updates a user's name
	UpdateName(ctx context.Context, id uuid.UUID, name string) error

	// Deactivate marks a user as inactive
	Deactivate(ctx context.Context, id uuid.UUID) error
}

// postgresRepository implements UserRepo using PostgreSQL
type postgresRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new backoffice user repository
func NewRepository(pool *pgxpool.Pool) UserRepo {
	return &postgresRepository{pool: pool}
}

// Create inserts a new backoffice user
func (r *postgresRepository) Create(ctx context.Context, tenantID uuid.UUID, email, name string, passwordHash string, role Role) (User, error) {
	if tenantID == uuid.Nil {
		return User{}, fmt.Errorf("create backoffice user: %w", apperrors.ErrMissingTenantID)
	}
	if email == "" {
		return User{}, fmt.Errorf("create backoffice user: email cannot be empty: %w", apperrors.ErrInvalidInput)
	}
	if passwordHash == "" {
		return User{}, fmt.Errorf("create backoffice user: password_hash cannot be empty: %w", apperrors.ErrInvalidInput)
	}

	// Acquire a connection to set RLS context
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return User{}, fmt.Errorf("create backoffice user: acquiring connection: %w", err)
	}
	defer conn.Release()

	// Set tenant context for RLS using set_config function
	if err := conn.QueryRow(ctx, "SELECT set_config('app.current_tenant_id', $1::text, true)", tenantID.String()).Scan(nil); err != nil {
		return User{}, fmt.Errorf("create backoffice user: setting tenant context: %w", err)
	}

	var user User

	query := `
		INSERT INTO backoffice_users (tenant_id, email, name, password_hash, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, tenant_id, email, role, name, must_change_password, is_active, created_at, updated_at
	`

	err = conn.QueryRow(ctx, query, tenantID, email, name, passwordHash, role).Scan(
		&user.ID, &user.TenantID, &user.Email, &user.Role, &user.Name,
		&user.MustChangePassword, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if err.Error() == "new row for relation \"backoffice_users\" violates unique constraint \"idx_backoffice_users_email_tenant\"" {
			return User{}, fmt.Errorf("create backoffice user: email already exists: %w", apperrors.ErrConflict)
		}
		return User{}, fmt.Errorf("create backoffice user: %w", err)
	}

	return user, nil
}

// GetByEmail retrieves a user by email and returns the password hash for auth
func (r *postgresRepository) GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (User, string, error) {
	if tenantID == uuid.Nil {
		return User{}, "", fmt.Errorf("get backoffice user by email: %w", apperrors.ErrMissingTenantID)
	}
	if email == "" {
		return User{}, "", fmt.Errorf("get backoffice user by email: email cannot be empty: %w", apperrors.ErrInvalidInput)
	}

	// Acquire a connection to set RLS context
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return User{}, "", fmt.Errorf("get backoffice user by email: acquiring connection: %w", err)
	}
	defer conn.Release()

	// Set tenant context for RLS using set_config function
	if err := conn.QueryRow(ctx, "SELECT set_config('app.current_tenant_id', $1::text, true)", tenantID.String()).Scan(nil); err != nil {
		return User{}, "", fmt.Errorf("get backoffice user by email: setting tenant context: %w", err)
	}

	var user User
	var passwordHash string

	query := `
		SELECT id, tenant_id, email, role, name, must_change_password, is_active, created_at, updated_at, password_hash
		FROM backoffice_users
		WHERE tenant_id = $1 AND email = $2
	`

	err = conn.QueryRow(ctx, query, tenantID, email).Scan(
		&user.ID, &user.TenantID, &user.Email, &user.Role, &user.Name,
		&user.MustChangePassword, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		&passwordHash,
	)

	if err == pgx.ErrNoRows {
		return User{}, "", fmt.Errorf("get backoffice user by email: %w", apperrors.ErrNotFound)
	}
	if err != nil {
		return User{}, "", fmt.Errorf("get backoffice user by email: %w", err)
	}

	return user, passwordHash, nil
}

// GetByID retrieves a user by ID
func (r *postgresRepository) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	if id == uuid.Nil {
		return User{}, fmt.Errorf("get backoffice user by id: invalid user id: %w", apperrors.ErrInvalidInput)
	}

	var user User

	query := `
		SELECT id, tenant_id, email, role, name, must_change_password, is_active, created_at, updated_at
		FROM backoffice_users
		WHERE id = $1
	`

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.TenantID, &user.Email, &user.Role, &user.Name,
		&user.MustChangePassword, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return User{}, fmt.Errorf("get backoffice user by id: %w", apperrors.ErrNotFound)
	}
	if err != nil {
		return User{}, fmt.Errorf("get backoffice user by id: %w", err)
	}

	return user, nil
}

// GetPasswordHashByID retrieves only the password hash for a user
func (r *postgresRepository) GetPasswordHashByID(ctx context.Context, id uuid.UUID) (string, error) {
	if id == uuid.Nil {
		return "", fmt.Errorf("get backoffice user password hash: invalid user id: %w", apperrors.ErrInvalidInput)
	}

	var passwordHash string

	query := `
		SELECT password_hash
		FROM backoffice_users
		WHERE id = $1
	`

	err := r.pool.QueryRow(ctx, query, id).Scan(&passwordHash)

	if err == pgx.ErrNoRows {
		return "", fmt.Errorf("get backoffice user password hash: %w", apperrors.ErrNotFound)
	}
	if err != nil {
		return "", fmt.Errorf("get backoffice user password hash: %w", err)
	}

	return passwordHash, nil
}

// List returns all users for a tenant
func (r *postgresRepository) List(ctx context.Context, tenantID uuid.UUID) ([]User, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("list backoffice users: %w", apperrors.ErrMissingTenantID)
	}

	// Acquire a connection to set RLS context
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("list backoffice users: acquiring connection: %w", err)
	}
	defer conn.Release()

	// Set tenant context for RLS
	if _, err := conn.Exec(ctx, fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'::uuid", tenantID)); err != nil {
		return nil, fmt.Errorf("list backoffice users: setting tenant context: %w", err)
	}

	query := `
		SELECT id, tenant_id, email, role, name, must_change_password, is_active, created_at, updated_at
		FROM backoffice_users
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := conn.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list backoffice users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(
			&user.ID, &user.TenantID, &user.Email, &user.Role, &user.Name,
			&user.MustChangePassword, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("list backoffice users: scanning row: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list backoffice users: %w", err)
	}

	return users, nil
}

// UpdatePassword updates a user's password hash
func (r *postgresRepository) UpdatePassword(ctx context.Context, id uuid.UUID, newPasswordHash string) error {
	if id == uuid.Nil {
		return fmt.Errorf("update backoffice user password: invalid user id: %w", apperrors.ErrInvalidInput)
	}
	if newPasswordHash == "" {
		return fmt.Errorf("update backoffice user password: password_hash cannot be empty: %w", apperrors.ErrInvalidInput)
	}

	query := `
		UPDATE backoffice_users
		SET password_hash = $1, must_change_password = false, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.pool.Exec(ctx, query, newPasswordHash, id)
	if err != nil {
		return fmt.Errorf("update backoffice user password: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("update backoffice user password: %w", apperrors.ErrNotFound)
	}

	return nil
}

// UpdateMustChangePassword updates the must_change_password flag
func (r *postgresRepository) UpdateMustChangePassword(ctx context.Context, id uuid.UUID, value bool) error {
	if id == uuid.Nil {
		return fmt.Errorf("update backoffice user must_change_password: invalid user id: %w", apperrors.ErrInvalidInput)
	}

	query := `
		UPDATE backoffice_users
		SET must_change_password = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.pool.Exec(ctx, query, value, id)
	if err != nil {
		return fmt.Errorf("update backoffice user must_change_password: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("update backoffice user must_change_password: %w", apperrors.ErrNotFound)
	}

	return nil
}

// UpdateName updates a user's name
func (r *postgresRepository) UpdateName(ctx context.Context, id uuid.UUID, name string) error {
	if id == uuid.Nil {
		return fmt.Errorf("update backoffice user name: invalid user id: %w", apperrors.ErrInvalidInput)
	}
	if name == "" {
		return fmt.Errorf("update backoffice user name: name cannot be empty: %w", apperrors.ErrInvalidInput)
	}

	query := `
		UPDATE backoffice_users
		SET name = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.pool.Exec(ctx, query, name, id)
	if err != nil {
		return fmt.Errorf("update backoffice user name: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("update backoffice user name: %w", apperrors.ErrNotFound)
	}

	return nil
}

// Deactivate marks a user as inactive
func (r *postgresRepository) Deactivate(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("deactivate backoffice user: invalid user id: %w", apperrors.ErrInvalidInput)
	}

	query := `
		UPDATE backoffice_users
		SET is_active = false, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deactivate backoffice user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("deactivate backoffice user: %w", apperrors.ErrNotFound)
	}

	return nil
}
