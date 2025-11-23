package database

import (
	"context"
	"database/sql"
	"fmt"

	"parsa/internal/crypto"
	"parsa/internal/models"
)

type UserRepository struct {
	db        *DB
	encryptor *crypto.Encryptor
}

func NewUserRepository(db *DB, encryptor *crypto.Encryptor) *UserRepository {
	return &UserRepository{
		db:        db,
		encryptor: encryptor,
	}
}

// decryptProviderKey decrypts the provider key if it exists
func (r *UserRepository) decryptProviderKey(user *models.User) error {
	if user.ProviderKey == "" {
		return nil
	}

	decrypted, err := r.encryptor.Decrypt(user.ProviderKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt provider key: %w", err)
	}
	user.ProviderKey = decrypted
	return nil
}

// encryptProviderKey encrypts the provider key if it exists
func (r *UserRepository) encryptProviderKey(providerKey *string) (*string, error) {
	if providerKey == nil || *providerKey == "" {
		return providerKey, nil
	}

	encrypted, err := r.encryptor.Encrypt(*providerKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt provider key: %w", err)
	}
	return &encrypted, nil
}

func (r *UserRepository) Create(ctx context.Context, params models.CreateUserParams) (*models.User, error) {
	query := `
    INSERT INTO users (email, name, first_name, last_name, avatar_url, oauth_provider, oauth_id, password_hash)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    RETURNING id, email, name, first_name, last_name, oauth_provider, oauth_id, password_hash, avatar_url, created_at, updated_at
`

	var user models.User
	err := r.db.QueryRowContext(
		ctx, query,
		params.Email, params.Name, params.FirstName, params.LastName, params.AvatarURL,
		params.OAuthProvider, params.OAuthID, params.PasswordHash,
	).Scan(
		&user.ID, &user.Email, &user.Name, &user.FirstName, &user.LastName,
		&user.OAuthProvider, &user.OAuthID, &user.PasswordHash, &user.AvatarURL,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*models.User, error) {
	query := `
		SELECT id, email, name, first_name, last_name, oauth_provider, oauth_id, password_hash, avatar_url, provider_key, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.Name, &user.FirstName, &user.LastName,
		&user.OAuthProvider, &user.OAuthID, &user.PasswordHash, &user.AvatarURL, &user.ProviderKey,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if err := r.decryptProviderKey(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, name, first_name, last_name, oauth_provider, oauth_id, password_hash, avatar_url, provider_key, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user models.User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.Name, &user.FirstName, &user.LastName,
		&user.OAuthProvider, &user.OAuthID, &user.PasswordHash, &user.AvatarURL, &user.ProviderKey,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if err := r.decryptProviderKey(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetByOAuth(ctx context.Context, provider, oauthID string) (*models.User, error) {
	query := `
		SELECT id, email, name, first_name, last_name, oauth_provider, oauth_id, password_hash, avatar_url, provider_key, created_at, updated_at
		FROM users
		WHERE oauth_provider = $1 AND oauth_id = $2
	`

	var user models.User
	err := r.db.QueryRowContext(ctx, query, provider, oauthID).Scan(
		&user.ID, &user.Email, &user.Name, &user.FirstName, &user.LastName,
		&user.OAuthProvider, &user.OAuthID, &user.PasswordHash, &user.AvatarURL, &user.ProviderKey,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if err := r.decryptProviderKey(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) List(ctx context.Context) ([]*models.User, error) {
	query := `
		SELECT id, email, name, first_name, last_name, oauth_provider, oauth_id, password_hash, avatar_url, provider_key, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID, &user.Email, &user.Name, &user.FirstName, &user.LastName,
			&user.OAuthProvider, &user.OAuthID, &user.PasswordHash, &user.AvatarURL, &user.ProviderKey,
			&user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		if err := r.decryptProviderKey(&user); err != nil {
			return nil, err
		}

		users = append(users, &user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

func (r *UserRepository) Update(ctx context.Context, userID int64, params models.UpdateUserParams) (*models.User, error) {
	// Encrypt provider key if provided
	encryptedProviderKey, err := r.encryptProviderKey(params.ProviderKey)
	if err != nil {
		return nil, err
	}

	query := `
		UPDATE users
		SET name = COALESCE($2, name),
		    first_name = COALESCE($3, first_name),
		    last_name = COALESCE($4, last_name),
		    avatar_url = COALESCE($5, avatar_url),
		    provider_key = COALESCE($6, provider_key),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, email, name, first_name, last_name, oauth_provider, oauth_id, password_hash, avatar_url, provider_key, created_at, updated_at
	`

	var user models.User
	err = r.db.QueryRowContext(
		ctx, query,
		userID, params.Name, params.FirstName, params.LastName, params.AvatarURL, encryptedProviderKey,
	).Scan(
		&user.ID, &user.Email, &user.Name, &user.FirstName, &user.LastName,
		&user.OAuthProvider, &user.OAuthID, &user.PasswordHash, &user.AvatarURL, &user.ProviderKey,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	if err := r.decryptProviderKey(&user); err != nil {
		return nil, err
	}

	return &user, nil
}
