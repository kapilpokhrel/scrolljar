package database

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	spec "github.com/kapilpokhrel/scrolljar/internal/api/spec"
)

type User struct {
	spec.User
	Email        string             `json:"-"`
	Activated    bool               `json:"-"`
	PasswordHash string             `json:"-"`
	UpdatedAt    pgtype.Timestamptz `json:"-"`
}

type UserModel struct {
	DBPool *pgxpool.Pool
}

func (m UserModel) Insert(user *User) error {
	query := `
		INSERT INTO users (username, email, password_hash)
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
		if pgErr.Code == "23505" && pgErr.ConstraintName == "users_email_key" {
			return ErrDuplicateUser
		}
		return pgErr
	default:
		return err
	}
}

func (m UserModel) GetByID(user *User) error {
	query := `
		SELECT email, username, password_hash, activated, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DBPool.QueryRow(ctx, query, user.ID).Scan(
		&user.Email,
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

func (m UserModel) GetUserByEmail(user *User) error {
	query := `
		SELECT id, username, password_hash, activated, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DBPool.QueryRow(ctx, query, user.Email).Scan(
		&user.ID,
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
		UPDATE users 
		SET username = $1, email = $2, activated = $3, password_hash = $4
		WHERE id = $5 AND updated_at = $6
		RETURNING updated_at
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DBPool.QueryRow(
		ctx,
		query,
		user.Username, user.Email, user.Activated, user.PasswordHash,
		user.ID, user.UpdatedAt.Time,
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
		DELETE FROM users
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
