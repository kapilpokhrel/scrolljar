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

type User struct {
	ID           int64              `json:"id"`
	Username     string             `json:"username"`
	Email        string             `json:"email"`
	Activated    bool               `json:"-"`
	PasswordHash string             `json:"-"`
	CreatedAt    pgtype.Timestamptz `json:"created_at"`
	UpdatedAt    pgtype.Timestamptz `json:"-"`
}

type UserModel struct {
	DBPool *pgxpool.Pool
}

func (m UserModel) Insert(user *User) error {
	query := `
		INSERT INTO user (username, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at 
	`

	args := []any{user.Username, user.Email, user.PasswordHash}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := m.DBPool.QueryRow(ctx, query, args...).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	var pgErr *pgconn.PgError
	switch {
	case errors.As(err, &pgErr):
		if pgErr.Code == "23505" && pgErr.ConstraintName == "user_email_key" {
			return ErrDuplicateUser
		}
		return pgErr
	default:
		return err
	}
}

func (m ScrollJarModel) GetUserByEmail(user *User) error {
	query := `
		SELECT id, username, password_hash, activated, created_at, updated_at
		FROM user 
		WHERE email = $1
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DBPool.QueryRow(ctx, query, user.Email).Scan(
		&user.Username,
		&user.PasswordHash,
		&user.Activated,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return ErrNoRecord
	default:
		return err
	}
}

func (m UserModel) Update(user *User) error {
	query := `
		UPDATE user 
		SET username = $1, email = $2, password_hash = $3
		WHERE id = $4 AND updated_at = $5
		RETURNING updated_at
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DBPool.QueryRow(
		ctx,
		query,
		user.Username, user.Email, user.PasswordHash,
		user.ID, user.UpdatedAt,
	).Scan(&user.UpdatedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return ErrEditConflict
	default:
		return err
	}
}

func (m UserModel) Delete(user *User) error {
	query := `
		DELETE FROM user 
		WHERE id = $1
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := m.DBPool.Exec(ctx, query, user.ID)
	switch {
	case result.RowsAffected() == 0:
		return ErrNoRecord
	default:
		return err
	}
}
