package database

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
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
	BaseModel
}

func (m UserModel) insert(ctx context.Context, q Queryer, user *User) error {
	query := `
		INSERT INTO user_account (username, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at 
	`

	args := []any{user.Username, user.Email, user.PasswordHash}

	err := q.QueryRow(ctx, query, args...).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	var pgErr *pgconn.PgError
	switch {
	case errors.As(err, &pgErr):
		if pgErr.Code == "23505" && pgErr.ConstraintName == "user_account_email_key" {
			return ErrDuplicateUser
		}
		return pgErr
	default:
		return err
	}
}

func (m UserModel) Insert(
	ctx context.Context,
	user *User,
) error {
	return m.insert(ctx, m.DBPool, user)
}

func (m UserModel) InsertTx(
	ctx context.Context,
	tx pgx.Tx,
	user *User,
) error {
	return m.insert(ctx, tx, user)
}

func (m UserModel) GetByID(ctx context.Context, user *User) error {
	query := `
		SELECT email, username, password_hash, activated, created_at, updated_at
		FROM user_account
		WHERE id = $1
	`
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

func (m UserModel) GetUserByEmail(ctx context.Context, user *User) error {
	query := `
		SELECT id, username, password_hash, activated, created_at, updated_at
		FROM user_account
		WHERE email = $1
	`
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

func (m UserModel) update(ctx context.Context, q Queryer, user *User) error {
	query := `
		UPDATE user_account
		SET username = $1, email = $2, activated = $3, password_hash = $4
		WHERE id = $5 AND updated_at = $6
		RETURNING updated_at
	`
	err := q.QueryRow(
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

func (m UserModel) Update(
	ctx context.Context,
	user *User,
) error {
	return m.update(ctx, m.DBPool, user)
}

func (m UserModel) UpdateTx(
	ctx context.Context,
	tx pgx.Tx,
	user *User,
) error {
	return m.update(ctx, tx, user)
}

func (m UserModel) delete(ctx context.Context, q Queryer, user *User) error {
	query := `
		DELETE FROM user_account
		WHERE id = $1
	`
	result, err := q.Exec(ctx, query, user.ID)
	switch {
	case result.RowsAffected() == 0:
		return ErrNoRecord
	default:
		return err
	}
}

func (m UserModel) Delete(
	ctx context.Context,
	user *User,
) error {
	return m.delete(ctx, m.DBPool, user)
}

func (m UserModel) DeleteTx(
	ctx context.Context,
	tx pgx.Tx,
	user *User,
) error {
	return m.delete(ctx, tx, user)
}
