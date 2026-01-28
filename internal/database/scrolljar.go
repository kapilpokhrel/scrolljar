package database

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	spec "github.com/kapilpokhrel/scrolljar/internal/api/spec"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

const Base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

type Scroll struct {
	spec.Scroll
	Uploaded  bool
	UpdatedAt pgtype.Timestamptz `json:"-"`
}

type ScrollJar struct {
	spec.Jar
	UserID       *int64             `json:"-"`
	PasswordHash *string            `json:"-"`
	UpdatedAt    pgtype.Timestamptz `json:"-"`
}

type ScrollJarModel struct {
	BaseModel
}

const (
	AccessPublic int = iota
	AccessPrivate
)

func (m ScrollJarModel) insert(ctx context.Context, q Queryer, jar *ScrollJar) error {
	query := `
		INSERT INTO scrolljar (id, user_id, name, access, password_hash, tags, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at 
	`

	for {
		slug, err := gonanoid.Generate(Base62Chars, 8)
		if err != nil {
			return err
		}

		args := []any{slug, jar.UserID, jar.Name, jar.Access, jar.PasswordHash, jar.Tags, jar.ExpiresAt.Time}

		err = q.QueryRow(ctx, query, args...).Scan(&jar.ID, &jar.CreatedAt, &jar.UpdatedAt)
		var pgErr *pgconn.PgError
		switch {
		case errors.As(err, &pgErr):
			if pgErr.Code == "23505" && pgErr.ConstraintName == "scrolljar_pkey" {
				continue
			}
			return pgErr
		default:
			return err
		}
	}
}

func (m ScrollJarModel) Insert(
	ctx context.Context,
	jar *ScrollJar,
) error {
	return m.insert(ctx, m.DBPool, jar)
}

func (m ScrollJarModel) InsertTx(
	ctx context.Context,
	tx pgx.Tx,
	jar *ScrollJar,
) error {
	return m.insert(ctx, tx, jar)
}

func (m ScrollJarModel) insertScroll(ctx context.Context, q Queryer, scroll *Scroll) error {
	query := `
		INSERT INTO scroll (id, jar_id, title, format)
		VALUES ($1, $2, $3, $4)
		RETURNING id, jar_id, created_at, updated_at
	`
	for {
		slug, err := gonanoid.Generate(Base62Chars, 8)
		if err != nil {
			return err
		}

		args := []any{slug, scroll.JarID, scroll.Title, scroll.Format}

		err = q.QueryRow(ctx, query, args...).Scan(&scroll.ID, &scroll.JarID, &scroll.CreatedAt, &scroll.UpdatedAt)
		var pgErr *pgconn.PgError
		switch {
		case errors.As(err, &pgErr):
			if pgErr.Code == "23505" && pgErr.ConstraintName == "scroll_pkey" {
				continue
			}
			return pgErr
		default:
			return err
		}
	}
}

func (m ScrollJarModel) InsertScroll(
	ctx context.Context,
	scroll *Scroll,
) error {
	return m.insertScroll(ctx, m.DBPool, scroll)
}

func (m ScrollJarModel) InsertScrollTx(
	ctx context.Context,
	tx pgx.Tx,
	scroll *Scroll,
) error {
	return m.insertScroll(ctx, tx, scroll)
}

func (m ScrollJarModel) Get(ctx context.Context, jar *ScrollJar) error {
	query := `
		SELECT name, user_id, access, password_hash, tags, expires_at, created_at, updated_at
		FROM scrolljar
		WHERE id = $1 AND (expires_at IS NULL OR expires_at > now());
	`
	err := m.DBPool.QueryRow(ctx, query, jar.ID).Scan(&jar.Name, &jar.UserID, &jar.Access, &jar.PasswordHash, &jar.Tags, &jar.ExpiresAt, &jar.CreatedAt, &jar.UpdatedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return ErrNoRecord
	default:
		return err
	}
}

func (m ScrollJarModel) GetAllByUserID(ctx context.Context, userID int64) ([]*ScrollJar, error) {
	query := `
		SELECT id, name, access, password_hash, tags, expires_at, created_at, updated_at
		FROM scrolljar
		WHERE user_id = $1 AND (expires_at IS NULL OR expires_at > now());
	`
	rows, err := m.DBPool.Query(ctx, query, userID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrNoRecord
		default:
			return nil, err
		}
	}
	defer rows.Close()

	jars := make([]*ScrollJar, 0, 255)
	for rows.Next() {
		jar := ScrollJar{UserID: &userID}
		rows.Scan(&jar.ID, &jar.Name, &jar.Access, &jar.PasswordHash, &jar.Tags, &jar.ExpiresAt, &jar.CreatedAt, &jar.UpdatedAt)
		jars = append(jars, &jar)
	}
	return jars, nil
}

func (m ScrollJarModel) GetAllScrolls(ctx context.Context, jar *ScrollJar) ([]*Scroll, error) {
	query := `
		SELECT s.id, s.jar_id, s.title, s.format, s.uploaded, s.created_at, s.updated_at
		FROM scroll s
		JOIN scrolljar j ON j.id = s.jar_id
		WHERE s.jar_id = $1 AND s.uploaded = TRUE AND (j.expires_at IS NULL OR j.expires_at > now());
	`
	rows, err := m.DBPool.Query(ctx, query, jar.ID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrNoRecord
		default:
			return nil, err
		}
	}
	defer rows.Close()

	scrolls := make([]*Scroll, 0, 255)
	for rows.Next() {
		scroll := Scroll{}
		rows.Scan(&scroll.ID, &scroll.JarID, &scroll.Title, &scroll.Format, &scroll.Uploaded, &scroll.CreatedAt, &scroll.UpdatedAt)
		scrolls = append(scrolls, &scroll)
	}
	return scrolls, nil
}

func (m ScrollJarModel) GetScrollCount(ctx context.Context, jar *ScrollJar) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM scroll s
		JOIN scrolljar j ON j.id = s.jar_id
		WHERE s.jar_id = $1 AND s.uploaded = TRUE AND (j.expires_at IS NULL OR j.expires_at > now());
	`
	var count int

	if err := m.DBPool.QueryRow(ctx, query, jar.ID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (m ScrollJarModel) GetScroll(ctx context.Context, scroll *Scroll) error {
	query := `
		SELECT s.jar_id, s.title, s.format, s.uploaded, s.created_at, s.updated_at
		FROM scroll s
		JOIN scrolljar j ON j.id = j.jar_id
		WHERE s.id = $1 AND s.uploaded = TRUE AND (j.expires_at IS NULL OR j.expires_at > now());
	`
	err := m.DBPool.QueryRow(ctx, query, scroll.ID).Scan(&scroll.JarID, &scroll.Title, &scroll.Format, &scroll.Uploaded, &scroll.CreatedAt, &scroll.UpdatedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return ErrNoRecord
	default:
		return err
	}
}

func (m ScrollJarModel) updateScroll(ctx context.Context, q Queryer, scroll *Scroll) error {
	query := `
		UPDATE scroll
		SET title = $1, format = $2
		WHERE id = $3 AND updated_at = $4
		RETURNING updated_at
	`
	err := q.QueryRow(
		ctx,
		query,
		scroll.Title, scroll.Format,
		scroll.ID, scroll.UpdatedAt.Time,
	).Scan(&scroll.UpdatedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return ErrEditConflict
	default:
		return err
	}
}

func (m ScrollJarModel) UpdateScroll(
	ctx context.Context,
	scroll *Scroll,
) error {
	return m.updateScroll(ctx, m.DBPool, scroll)
}

func (m ScrollJarModel) UpdateScrollTx(
	ctx context.Context,
	tx pgx.Tx,
	scroll *Scroll,
) error {
	return m.updateScroll(ctx, tx, scroll)
}

func (m ScrollJarModel) setScrollUpload(ctx context.Context, q Queryer, scroll *Scroll) error {
	query := `
		UPDATE scroll
		SET uploaded = TRUE
		WHERE id = $1 AND updated_at = $2
		RETURNING updated_at
	`
	err := q.QueryRow(
		ctx,
		query,
		scroll.ID, scroll.UpdatedAt.Time,
	).Scan(&scroll.UpdatedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return ErrEditConflict
	default:
		return err
	}
}

func (m ScrollJarModel) SetScrollUpload(
	ctx context.Context,
	scroll *Scroll,
) error {
	return m.setScrollUpload(ctx, m.DBPool, scroll)
}

func (m ScrollJarModel) SetScrollUploadTx(
	ctx context.Context,
	tx pgx.Tx,
	scroll *Scroll,
) error {
	return m.setScrollUpload(ctx, tx, scroll)
}

func (m ScrollJarModel) deleteScroll(ctx context.Context, q Queryer, scroll *Scroll) error {
	query := `
		DELETE FROM scroll
		WHERE id = $1
	`
	result, err := q.Exec(ctx, query, scroll.ID)
	switch {
	case result.RowsAffected() == 0:
		return ErrNoRecord
	default:
		return err
	}
}

func (m ScrollJarModel) DeleteScroll(
	ctx context.Context,
	scroll *Scroll,
) error {
	return m.deleteScroll(ctx, m.DBPool, scroll)
}

func (m ScrollJarModel) DeleteScrollTx(
	ctx context.Context,
	tx pgx.Tx,
	scroll *Scroll,
) error {
	return m.deleteScroll(ctx, tx, scroll)
}

func (m ScrollJarModel) delete(ctx context.Context, q Queryer, jar *ScrollJar) error {
	query := `
		DELETE FROM scrolljar
		WHERE id = $1
	`
	result, err := q.Exec(ctx, query, jar.ID)
	switch {
	case result.RowsAffected() == 0:
		return ErrNoRecord
	default:
		return err
	}
}

func (m ScrollJarModel) Delete(
	ctx context.Context,
	jar *ScrollJar,
) error {
	return m.delete(ctx, m.DBPool, jar)
}

func (m ScrollJarModel) DeleteTx(
	ctx context.Context,
	tx pgx.Tx,
	jar *ScrollJar,
) error {
	return m.delete(ctx, tx, jar)
}
