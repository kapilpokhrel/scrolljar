// Package database creates a database models and manages CRUD
package database

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNoRecord     = errors.New("record not found")
	ErrEditConflict = errors.New("edit confict")
)

type Models struct {
	ScrollJar ScrollJarModel
}

func NewModels(dbPool *pgxpool.Pool) Models {
	return Models{
		ScrollJar: ScrollJarModel{dbPool},
	}
}
