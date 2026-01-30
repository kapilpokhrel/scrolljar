// Package database creates a database models and manages CRUD
package database

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNoRecord      = errors.New("record not found")
	ErrEditConflict  = errors.New("edit confict")
	ErrDuplicateUser = errors.New("duplicate email")
)

type DBCFG struct {
	URL          string
	MaxOpenConns int
	MinIdleConns int
	MaxIdleTime  time.Duration
}

type Models struct {
	ScrollJar ScrollJarModel
	Users     UserModel
	Token     TokenModel
}

type BaseModel struct {
	DBPool *pgxpool.Pool
}

func (d BaseModel) GetTx(ctx context.Context) (pgx.Tx, error) {
	return d.DBPool.Begin(ctx)
}

type Queryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func SetupDB(cfg DBCFG) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, err
	}
	config.MaxConnIdleTime = cfg.MaxIdleTime
	config.MaxConns = int32(cfg.MaxOpenConns)
	config.MinIdleConns = int32(cfg.MinIdleConns)
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

func NewModels(dbPool *pgxpool.Pool) Models {
	return Models{
		ScrollJar: ScrollJarModel{BaseModel: BaseModel{DBPool: dbPool}},
		Users:     UserModel{BaseModel: BaseModel{DBPool: dbPool}},
		Token:     TokenModel{BaseModel: BaseModel{DBPool: dbPool}},
	}
}
