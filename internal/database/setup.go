package database

import (
	"context"
	"errors"
	"flag"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrEditConflict  = errors.New("edit conflict")
	ErrDuplicateUser = errors.New("duplicate email")
)

type DBCFG struct {
	URL          string
	MaxOpenConns *int
	MinIdleConns *int
	MaxIdleTime  *time.Duration
}

func (cfg *DBCFG) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&cfg.URL, "db_url", os.Getenv("SCROLLJAR_DB_URL"), "PostgreSQL URL")
	cfg.MaxOpenConns = new(int)
	cfg.MinIdleConns = new(int)
	cfg.MaxIdleTime = new(time.Duration)

	fs.IntVar(cfg.MaxOpenConns, "db_max-open-conns", 50, "PostgreSQL max open connections")
	fs.IntVar(cfg.MinIdleConns, "db_min-idle-conns", 50, "PostgreSQL min idle connections")
	fs.DurationVar(cfg.MaxIdleTime, "db_max-idle-time", time.Minute*10, "PostgreSQL max idle time")
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
