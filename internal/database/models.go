// Package database creates a database models and manages CRUD
package database

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNoRecord      = errors.New("record not found")
	ErrEditConflict  = errors.New("edit confict")
	ErrDuplicateUser = errors.New("duplicate email")
)

type Models struct {
	ScrollJar ScrollJarModel
	Users     UserModel
}

func NewModels(dbPool *pgxpool.Pool) Models {
	return Models{
		ScrollJar: ScrollJarModel{dbPool},
		Users:     UserModel{dbPool},
	}
}
