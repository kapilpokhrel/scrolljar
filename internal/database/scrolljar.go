package database

import (
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/matoous/go-nanoid/v2"
)

const Base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

type Scroll struct {
	ID        int64     `json:"id"`
	JarID     int64     `json:"-"`
	Title     string    `json:"title,omitempty"`
	Format    string    `json:"format,omitempty"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"-"`
}

type ScrollJar struct {
	ID        int64 `json:"-"`
	Slug      string
	Name      string    `json:"name,omitempty"`
	Access    int       `json:"access"`
	Tags      []string  `json:"tags,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"-"`
}

type ScrollJarModel struct {
	DB *sql.DB
}

func (m ScrollJarModel) Insert(sJ *ScrollJar) error {
	query := `
		INSERT INTO scrolljar (slug, name, access, tags, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, slug, created_at, updated_at 
	`
	
	for {
		slug, err := gonanoid.Generate(Base62Chars, 8)
		if err != nil {
			return err
		}
		args := []any{slug, sJ.Name, sJ.Access, sJ.Tags, sJ.ExpiresAt}

		err = m.DB.QueryRow(query, args...).Scan(&sJ.ID, &sJ.Slug, &sJ.CreatedAt, &sJ.UpdatedAt)
		var pgErr *pgconn.PgError
		switch {
		case errors.As(err, &pgErr):
			if pgErr.Code == "23505" && pgErr.ConstraintName == "scrolljar_slug_key" {
				continue
			}
			return pgErr
		default:
			return err
		}
	}
}
