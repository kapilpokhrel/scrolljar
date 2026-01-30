package database

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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
	BaseModel
}

func (m TokenModel) insert(ctx context.Context, q Queryer, token *Token) error {
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
	_, err := q.Exec(ctx, query, args...)
	return err
}

func (m TokenModel) Insert(
	ctx context.Context,
	token *Token,
) error {
	return m.insert(ctx, m.DBPool, token)
}

func (m TokenModel) InsertTx(
	ctx context.Context,
	tx pgx.Tx,
	token *Token,
) error {
	return m.insert(ctx, tx, token)
}

func (m TokenModel) GetTokenByHash(ctx context.Context, token *Token) error {
	query := `
		SELECT user_id, scope, expires_at 
		FROM token 
		WHERE token_hash = $1 AND (expires_at IS NULL OR expires_at > now())
	`
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

func (m TokenModel) deleteByHash(ctx context.Context, q Queryer, token *Token) error {
	query := `
		DELETE FROM token 
		WHERE token_hash = $1
	`
	result, err := q.Exec(ctx, query, token.TokenHash)
	switch {
	case result.RowsAffected() == 0:
		return ErrNoRecord
	default:
		return err
	}
}

func (m TokenModel) DeleteByHash(
	ctx context.Context,
	token *Token,
) error {
	return m.deleteByHash(ctx, m.DBPool, token)
}

func (m TokenModel) DeleteByHashTx(
	ctx context.Context,
	tx pgx.Tx,
	token *Token,
) error {
	return m.deleteByHash(ctx, tx, token)
}

func (m TokenModel) deleteAllByUser(ctx context.Context, q Queryer, token *Token) error {
	query := `
		DELETE FROM token 
		WHERE user_id = $1
	`
	result, err := q.Exec(ctx, query, token.UserID)
	switch {
	case result.RowsAffected() == 0:
		return ErrNoRecord
	default:
		return err
	}
}

func (m TokenModel) DeleteAllByUser(
	ctx context.Context,
	token *Token,
) error {
	return m.deleteAllByUser(ctx, m.DBPool, token)
}

func (m TokenModel) DeleteAllByUserTx(
	ctx context.Context,
	tx pgx.Tx,
	token *Token,
) error {
	return m.deleteAllByUser(ctx, tx, token)
}

func (m TokenModel) DeleteExpired(ctx context.Context) error {
	query := `
		DELETE FROM token 
		WHERE expires_at <= now()
	`
	result, err := m.DBPool.Exec(ctx, query)
	switch {
	case result.RowsAffected() == 0:
		return nil
	default:
		return err
	}
}
