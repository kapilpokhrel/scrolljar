// Package db creates a database models and manages CRUD
package database

import "database/sql"

type Models struct {
	ScrollJar ScrollJarModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		ScrollJar: ScrollJarModel{db},
	}
}
