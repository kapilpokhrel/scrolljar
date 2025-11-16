package database

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Token struct {
	TokenHash string             `json:"-"`
	UserID    int64              `json:"-"`
	Scope     string             `json:"-"`
	ExpiresAt pgtype.Timestamptz `json:"-"`
}

const (
	ScopeActivation = "activation"
)

type TokenModel struct {
	DBPool *pgxpool.Pool
}

func (m TokenModel) Insert(token *Token) error {
	query := `
		INSERT INTO token (token_hash, user_id, scope, expires_at)
		VALUES ($1, $2, $3, $4)
	`

	args := []any{token.TokenHash, token.UserID, token.Scope, token.ExpiresAt}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DBPool.Exec(ctx, query, args...)

	var pgErr *pgconn.PgError
	switch {
	case errors.As(err, &pgErr):
		if pgErr.Code == "23505" && pgErr.ConstraintName == "token_pkey" {
			return ErrDuplicateUser
		}
		return pgErr
	default:
		return err
	}
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

func (m TokenModel) Delete(token *Token) error {
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
