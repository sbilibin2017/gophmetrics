package db

import (
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// Opt defines a function type that applies a configuration to sqlx.DB.
type Opt func(*sqlx.DB)

// NewDB establishes a connection to the database and applies any given options.
func New(driver string, dsn string, opts ...Opt) (*sqlx.DB, error) {
	db, err := sqlx.Connect(driver, dsn)
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		opt(db)
	}
	return db, nil
}

// WithMaxOpenConns sets the maximum number of open connections.
func WithMaxOpenConns(opts ...int) Opt {
	return func(db *sqlx.DB) {
		for _, opt := range opts {
			if opt > 0 {
				db.SetMaxOpenConns(opt)
				break
			}
		}
	}
}

// WithMaxIdleConns sets the maximum number of idle connections.
func WithMaxIdleConns(opts ...int) Opt {
	return func(db *sqlx.DB) {
		for _, opt := range opts {
			if opt > 0 {
				db.SetMaxIdleConns(opt)
				break
			}
		}
	}
}

// WithConnMaxLifetime sets the maximum connection lifetime.
func WithConnMaxLifetime(opts ...time.Duration) Opt {
	return func(db *sqlx.DB) {
		for _, opt := range opts {
			if opt != 0 {
				db.SetConnMaxLifetime(opt)
				break
			}
		}
	}
}
