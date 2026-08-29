package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// APIToken represents an API token in the system
type APIToken struct {
	ID          int64      `json:"id" db:"id"`
	UserID      int64      `json:"user_id" db:"user_id"`
	Token       string     `json:"token" db:"token"`
	Description string     `json:"description" db:"description"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

// TokenRepository handles database operations for API tokens
type TokenRepository struct {
	db *sql.DB
}

// NewTokenRepository creates a new token repository
func NewTokenRepository(db *sql.DB) *TokenRepository {
	return &TokenRepository{
		db: db,
	}
}

// GetTokenByID retrieves a token by ID
func (r *TokenRepository) GetTokenByID(ctx context.Context, id int64) (*APIToken, error) {
	query := `SELECT id, user_id, token, description, expires_at, created_at FROM api_tokens WHERE id = ?`

	var token APIToken
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&token.ID,
		&token.UserID,
		&token.Token,
		&token.Description,
		&token.ExpiresAt,
		&token.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("token not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	return &token, nil
}

// GetTokenByValue retrieves a token by its value
func (r *TokenRepository) GetTokenByValue(ctx context.Context, tokenValue string) (*APIToken, error) {
	query := `SELECT id, user_id, token, description, expires_at, created_at FROM api_tokens WHERE token = ?`

	var token APIToken
	err := r.db.QueryRowContext(ctx, query, tokenValue).Scan(
		&token.ID,
		&token.UserID,
		&token.Token,
		&token.Description,
		&token.ExpiresAt,
		&token.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("token not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	// Check if token is expired
	if token.ExpiresAt != nil && token.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("token expired")
	}

	return &token, nil
}

// ListTokensForUser retrieves all tokens for a user
func (r *TokenRepository) ListTokensForUser(ctx context.Context, userID int64) ([]*APIToken, error) {
	query := `SELECT id, user_id, token, description, expires_at, created_at FROM api_tokens WHERE user_id = ?`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*APIToken
	for rows.Next() {
		var token APIToken
		err := rows.Scan(
			&token.ID,
			&token.UserID,
			&token.Token,
			&token.Description,
			&token.ExpiresAt,
			&token.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan token: %w", err)
		}
		tokens = append(tokens, &token)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating token rows: %w", err)
	}

	return tokens, nil
}

// CreateToken creates a new API token
func (r *TokenRepository) CreateToken(ctx context.Context, token *APIToken) error {
	// Set creation timestamp
	token.CreatedAt = time.Now()

	query := `
		INSERT INTO api_tokens (user_id, token, description, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query, token.UserID, token.Token, token.Description, token.ExpiresAt, token.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create token: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get token ID: %w", err)
	}

	token.ID = id
	return nil
}

// UpdateToken updates an existing API token
func (r *TokenRepository) UpdateToken(ctx context.Context, token *APIToken) error {
	query := `
		UPDATE api_tokens
		SET description = ?, expires_at = ?
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query, token.Description, token.ExpiresAt, token.ID)
	if err != nil {
		return fmt.Errorf("failed to update token: %w", err)
	}

	return nil
}

// DeleteToken deletes an API token
func (r *TokenRepository) DeleteToken(ctx context.Context, id int64) error {
	query := `DELETE FROM api_tokens WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}

	return nil
}

// DeleteExpiredTokens deletes all expired tokens
func (r *TokenRepository) DeleteExpiredTokens(ctx context.Context) (int64, error) {
	query := `DELETE FROM api_tokens WHERE expires_at IS NOT NULL AND expires_at < ?`

	result, err := r.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired tokens: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows count: %w", err)
	}

	return count, nil
}
