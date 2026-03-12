package store

import (
	"context"
	"go-ocs/internal/logging"
	"go-ocs/internal/store/sqlc"
	"net/url"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	DB *pgxpool.Pool
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
