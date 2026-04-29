package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yaroslav/bp-user-service/internal/model"
)

var ErrNotFound = errors.New("user not found")
var ErrAlreadyExists = errors.New("user already exists")

// UserRepository handles user persistence in PostgreSQL.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create inserts a new user. Returns ErrAlreadyExists if the id or email is taken.
func (r *UserRepository) Create(ctx context.Context, user *model.User) (*model.User, error) {
	query := `
		INSERT INTO users (id, email, display_name, avatar_url, currency, timezone)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		user.ID,
		user.Email,
		user.DisplayName,
		user.AvatarURL,
		user.Currency,
		user.Timezone,
	).Scan(&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if isDuplicateError(err) {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("inserting user: %w", err)
	}

	return user, nil
}

// GetByID retrieves a user by ID. Soft-deleted users are treated as not found.
func (r *UserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	query := `
		SELECT id, email, display_name, avatar_url, currency, timezone, created_at, updated_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL`

	u := &model.User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Email,
		&u.DisplayName,
		&u.AvatarURL,
		&u.Currency,
		&u.Timezone,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("querying user: %w", err)
	}

	return u, nil
}

// UpdateFields holds the optional fields for a partial user update.
type UpdateFields struct {
	DisplayName *string
	AvatarURL   *string
	Currency    *string
	Timezone    *string
}

// Update applies a partial update to a user. Returns ErrNotFound if the user
// does not exist or is soft-deleted.
func (r *UserRepository) Update(ctx context.Context, id string, fields UpdateFields) (*model.User, error) {
	setClauses := []string{}
	args := []any{}
	argIdx := 1

	if fields.DisplayName != nil {
		setClauses = append(setClauses, fmt.Sprintf("display_name = $%d", argIdx))
		args = append(args, *fields.DisplayName)
		argIdx++
	}
	if fields.AvatarURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("avatar_url = $%d", argIdx))
		args = append(args, *fields.AvatarURL)
		argIdx++
	}
	if fields.Currency != nil {
		setClauses = append(setClauses, fmt.Sprintf("currency = $%d", argIdx))
		args = append(args, *fields.Currency)
		argIdx++
	}
	if fields.Timezone != nil {
		setClauses = append(setClauses, fmt.Sprintf("timezone = $%d", argIdx))
		args = append(args, *fields.Timezone)
		argIdx++
	}

	if len(setClauses) == 0 {
		return r.GetByID(ctx, id)
	}

	setClauses = append(setClauses, "updated_at = now()")

	query := fmt.Sprintf(`
		UPDATE users
		SET %s
		WHERE id = $%d AND deleted_at IS NULL
		RETURNING id, email, display_name, avatar_url, currency, timezone, created_at, updated_at`,
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	u := &model.User{}
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&u.ID,
		&u.Email,
		&u.DisplayName,
		&u.AvatarURL,
		&u.Currency,
		&u.Timezone,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("updating user: %w", err)
	}

	return u, nil
}

// SoftDelete marks a user as deleted. Returns ErrNotFound if the user
// does not exist or is already soft-deleted.
func (r *UserRepository) SoftDelete(ctx context.Context, id string) error {
	query := `
		UPDATE users
		SET deleted_at = now(), updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("soft-deleting user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// isDuplicateError checks if the error is a PostgreSQL unique_violation (23505).
func isDuplicateError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
