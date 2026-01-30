// Package database creates a database models and manages CRUD
package database

import (
	"context"
	"errors"
	"flag"
	"os"
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
	MaxOpenConns *int
	MinIdleConns *int
	MaxIdleTime  *time.Duration
}

func (cfg *DBCFG) RegisterFlags(fs *flag.FlagSet) {
	// Sets Defauts Too

	fs.StringVar(&cfg.URL, "db_url", os.Getenv("SCROLLJAR_DB_URL"), "PostgreSQL URL")
	cfg.MaxOpenConns = new(int)
	cfg.MinIdleConns = new(int)
	cfg.MaxIdleTime = new(time.Duration)

	fs.IntVar(cfg.MaxOpenConns,
		"db_max-open-conns", 50,
		"PostgreSQL max open connections",
	)
	fs.IntVar(cfg.MinIdleConns,
		"db_min-idle-conns", 50,
		"PostgreSQL min idle connections",
	)
	fs.DurationVar(cfg.MaxIdleTime,
		"db_max-idle-time", time.Minute*10,
		"PostgreSQL max idle time",
	)
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
	if cfg.MaxIdleTime != nil {
		config.MaxConnIdleTime = *cfg.MaxIdleTime
	}
	if cfg.MaxOpenConns != nil {
		config.MaxConns = int32(*cfg.MaxOpenConns)
	}
	if cfg.MinIdleConns != nil {
		config.MinIdleConns = int32(*cfg.MinIdleConns)
	}
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
