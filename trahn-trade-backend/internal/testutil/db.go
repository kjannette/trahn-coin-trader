package testutil

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// SetupPool creates a pgxpool.Pool for integration tests.
// Connection details come from env vars or sensible defaults.
func SetupPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	_ = godotenv.Load("../../.env")

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		host := EnvOr("DB_HOST", "localhost")
		port := EnvOr("DB_PORT", "5432")
		name := EnvOr("DB_NAME", "trahn_grid_trader")
		user := EnvOr("DB_USER", "postgres")
		pass := EnvOr("DB_PASSWORD", "")
		dsn = "postgres://" + user + ":" + pass + "@" + host + ":" + port + "/" + name + "?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

func EnvOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
