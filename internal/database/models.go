// Package database creates a database models and manages CRUD
package database

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

var (
	ErrNoRecord     = errors.New("record not found")
	ErrEditConflict = errors.New("edit confict")
)

type Models struct {
	ScrollJar ScrollJarModel
}

func NewModels(db *pgx.Conn) Models {
	return Models{
		ScrollJar: ScrollJarModel{db},
	}
}
