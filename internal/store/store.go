package store

import (
	"context"
	"go-ocs/internal/logging"
	"go-ocs/internal/store/sqlc"
	"net/url"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolDB is the subset of pgxpool.Pool methods used by the dynamic store queries.
// Defining it as an interface allows unit tests to substitute a mock without
// requiring a real PostgreSQL connection. pgxpool.Pool satisfies this interface.
type PoolDB interface {
	sqlc.DBTX
	Close()
}

type Store struct {
	DB PoolDB
	Q  *sqlc.Queries
}

func NewStore(dbUrl string) *Store {
	db, err := pgxpool.New(context.Background(), dbUrl)
	if err != nil {
		logging.Fatal("create db pool failed", "err", err)
	}

	if err := db.Ping(context.Background()); err != nil {
		db.Close()
		logging.Fatal("db ping failed", "err", err)
	}

	logging.Info("Connected to database", "url", sanitizeDBURL(dbUrl))

	q := sqlc.New(db)
	return &Store{DB: db, Q: q}
}

func sanitizeDBURL(dbURL string) string {
	u, err := url.Parse(dbURL)
	if err != nil {
		return "<invalid-db-url>"
	}

	if u.User != nil {
		u.User = url.UserPassword(u.User.Username(), "xxxxx")
	}

	return u.String()
}
