package database

import (
	"context"
	"database/sql"
	"fmt"

	"parsa/internal/models"
)

type UserRepository struct {
	db *DB
}

func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, params models.CreateUserParams) (*models.User, error) {
	query := `
		INSERT INTO users (email, name, oauth_provider, oauth_id, password_hash)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, email, name, oauth_provider, oauth_id, password_hash, created_at, updated_at
	`

	var user models.User
	err := r.db.QueryRowContext(
		ctx, query,
		params.Email, params.Name, params.OAuthProvider, params.OAuthID, params.PasswordHash,
	).Scan(
		&user.ID, &user.Email, &user.Name,
		&user.OAuthProvider, &user.OAuthID, &user.PasswordHash,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	query := `
		SELECT id, email, name, oauth_provider, oauth_id, password_hash, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.Name,
		&user.OAuthProvider, &user.OAuthID, &user.PasswordHash,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, name, oauth_provider, oauth_id, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user models.User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.Name,
		&user.OAuthProvider, &user.OAuthID, &user.PasswordHash,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) GetByOAuth(ctx context.Context, provider, oauthID string) (*models.User, error) {
	query := `
		SELECT id, email, name, oauth_provider, oauth_id, password_hash, created_at, updated_at
		FROM users
		WHERE oauth_provider = $1 AND oauth_id = $2
	`

	var user models.User
	err := r.db.QueryRowContext(ctx, query, provider, oauthID).Scan(
		&user.ID, &user.Email, &user.Name,
		&user.OAuthProvider, &user.OAuthID, &user.PasswordHash,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}
