package database

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Token struct {
	TokenHash []byte             `json:"-"`
	UserID    int64              `json:"-"`
	Scope     string             `json:"-"`
	ExpiresAt pgtype.Timestamptz `json:"-"`
}

const (
	ScopeActivation    = "activation"
	ScopeAuthorization = "access"
	ScopeRefresh       = "refresh"
)

type TokenModel struct {
	DBPool *pgxpool.Pool
}

func (m TokenModel) Insert(token *Token) error {
	// DO UPDATE will be useful later when there will be a endpoint for token regeneration

	query := `
		INSERT INTO token (token_hash, user_id, expires_at, scope)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT ON CONSTRAINT unique_user_scope_token
		DO UPDATE SET
			token_hash = $1,
			expires_at = $3
	`

	args := []any{token.TokenHash, token.UserID, token.ExpiresAt.Time, token.Scope}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DBPool.Exec(ctx, query, args...)
	return err
}

func (m TokenModel) GetTokenByHash(token *Token) error {
	query := `
		SELECT user_id, scope, expires_at 
		FROM token 
		WHERE token_hash = $1
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DBPool.QueryRow(ctx, query, token.TokenHash).Scan(
		&token.UserID,
		&token.Scope,
		&token.ExpiresAt,
	)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return ErrNoRecord
	default:
		return err
	}
}

func (m TokenModel) DeleteByHash(token *Token) error {
	query := `
		DELETE FROM token 
		WHERE token_hash = $1
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := m.DBPool.Exec(ctx, query, token.TokenHash)
	switch {
	case result.RowsAffected() == 0:
		return ErrNoRecord
	default:
		return err
	}
}

func (m TokenModel) DeleteAllByUser(token *Token) error {
	query := `
		DELETE FROM token 
		WHERE user_id = $1
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := m.DBPool.Exec(ctx, query, token.UserID)
	switch {
	case result.RowsAffected() == 0:
		return ErrNoRecord
	default:
		return err
	}
}
