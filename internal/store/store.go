package store

import (
	"context"
	"go-ocs/internal/logging"
	"go-ocs/internal/store/sqlc"
	"net/url"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBQuerier is the subset of *pgxpool.Pool methods used by dynamic store methods.
// Defining it as an interface allows unit tests to substitute a mock without
// requiring a real PostgreSQL connection.
type DBQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Store struct {
	DB      *pgxpool.Pool // pool lifecycle: Ping, Close, Stat
	Q       *sqlc.Queries
	querier DBQuerier // defaults to DB; substituted in unit tests
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
	return &Store{DB: db, Q: q, querier: db}
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
