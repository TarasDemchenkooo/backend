package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TarasDemchenkooo/backend/services/auth/internal/domain"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

const createOrReplaceUnverifiedQuery = `
INSERT INTO users (id, email, password_hash, role, is_verified, created_at, updated_at)
VALUES ($1, $2, $3, $4, FALSE, $5, $5)
ON CONFLICT (email) DO UPDATE
	SET id = EXCLUDED.id,
    	password_hash = EXCLUDED.password_hash,
       	role = EXCLUDED.role,
	   	created_at = EXCLUDED.created_at,
       	updated_at = EXCLUDED.updated_at
WHERE users.is_verified = FALSE
RETURNING id`

func (r *UserRepository) CreateOrReplaceUnverified(ctx context.Context, u *domain.User) error {
	row := r.pool.QueryRow(ctx, createOrReplaceUnverifiedQuery,
		u.ID, u.Email.String(), u.PasswordHash, u.Role.String(), u.CreatedAt)

	if err := row.Scan(&u.ID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrEmailAlreadyTaken
		}
		return fmt.Errorf("insert user: %w", err)
	}

	return nil
}
