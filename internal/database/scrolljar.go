package database

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

const Base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

type Scroll struct {
	ID        int8               `json:"id"`
	JarID     string             `json:"jarid"`
	Title     string             `json:"title,omitempty"`
	Format    string             `json:"format,omitempty"`
	Content   string             `json:"content,omitempty"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
	UpdatedAt pgtype.Timestamptz `json:"-"`
	URI       string             `json:"uri"`
	Jar       *ScrollJar         `json:"-"`
}

type ScrollJar struct {
	ID           string                   `json:"id"`
	Name         string                   `json:"name,omitempty"`
	Access       int                      `json:"access"`
	PasswordHash string                   `json:"-"`
	Tags         pgtype.FlatArray[string] `json:"tags"`
	ExpiresAt    pgtype.Timestamptz       `json:"expires_at"`
	CreatedAt    pgtype.Timestamptz       `json:"created_at"`
	UpdatedAt    pgtype.Timestamptz       `json:"-"`
	URI          string                   `json:"uri"`
}

type ScrollJarModel struct {
	DB *pgx.Conn
}

func (m ScrollJarModel) Insert(jar *ScrollJar) error {
	query := `
		INSERT INTO scrolljar (id, name, access, password_hash, tags, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at 
	`

	for {
		slug, err := gonanoid.Generate(Base62Chars, 8)
		if err != nil {
			return err
		}

		args := []any{slug, jar.Name, jar.Access, jar.PasswordHash, jar.Tags, jar.ExpiresAt}

		err = m.DB.QueryRow(context.Background(), query, args...).Scan(&jar.ID, &jar.CreatedAt, &jar.UpdatedAt)
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

func (m ScrollJarModel) InsertScroll(scroll *Scroll) error {
	query := `
		INSERT INTO scroll (id, jar_id, title, format, content)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING jar_id, created_at, updated_at
	`
	args := []any{scroll.ID, scroll.Jar.ID, scroll.Title, scroll.Format, scroll.Content}
	err := m.DB.QueryRow(context.Background(), query, args...).Scan(&scroll.JarID, &scroll.CreatedAt, &scroll.UpdatedAt)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return ErrNoRecord
	default:
		return err
	}
}

func (m ScrollJarModel) Get(jar *ScrollJar) error {
	query := `
		SELECT name, access, password_hash, tags, expires_at, created_at, updated_at
		FROM scrolljar
		WHERE id = $1
	`

	err := m.DB.QueryRow(context.Background(), query, jar.ID).Scan(&jar.Name, &jar.Access, &jar.PasswordHash, &jar.Tags, &jar.ExpiresAt, &jar.CreatedAt, &jar.UpdatedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return ErrNoRecord
	default:
		return err
	}
}

func (m ScrollJarModel) GetAllScrolls(jar *ScrollJar) ([]*Scroll, error) {
	query := `
		SELECT id, jar_id, title, format, created_at, updated_at
		FROM scroll
		WHERE jar_id = $1
	`

	rows, err := m.DB.Query(context.Background(), query, jar.ID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrNoRecord
		default:
			return nil, err
		}
	}

	scrolls := make([]*Scroll, 0, 255)
	for rows.Next() {
		scroll := Scroll{}
		rows.Scan(&scroll.ID, &scroll.JarID, &scroll.Title, &scroll.Format, &scroll.CreatedAt, &scroll.UpdatedAt)
		scroll.Jar = jar
		scrolls = append(scrolls, &scroll)
	}
	return scrolls, nil
}

func (m ScrollJarModel) GetScroll(scroll *Scroll) error {
	query := `
		SELECT jar_id, title, format, content, created_at, updated_at
		FROM scroll
		WHERE jar_id = $1 AND id = $2
	`
	err := m.DB.QueryRow(context.Background(), query, scroll.Jar.ID, scroll.ID).Scan(&scroll.JarID, &scroll.Title, &scroll.Format, &scroll.Content, &scroll.CreatedAt, &scroll.UpdatedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return ErrNoRecord
	default:
		return err
	}
}

func (m ScrollJarModel) UpdateScroll(scroll *Scroll) error {
	query := `
		UPDATE scroll
		SET title = $1, format = $2, content = $3
		WHERE id = $4 AND jar_id = $5 AND updated_at = $6
		RETURNING updated_at
	`

	err := m.DB.QueryRow(
		context.Background(),
		query,
		scroll.Title, scroll.Format, scroll.Content,
		scroll.ID, scroll.Jar.ID, scroll.UpdatedAt,
	).Scan(&scroll.UpdatedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return ErrEditConflict
	default:
		return err
	}
}
