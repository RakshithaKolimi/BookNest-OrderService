package postgres

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultConnectTimeout = 5 * time.Second

func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	// Build the Postgres connection string (DSN) from environment variables.
	dsn := buildDSNFromEnv()

	// Parse the DSN into a pgxpool.Config struct.
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	// Create a new connection pool with the provided configuration and a connection timeout.
	connectCtx, cancel := context.WithTimeout(ctx, defaultConnectTimeout)
	defer cancel()

	// Attempt to establish a connection to the Postgres database using the connection pool.
	pool, err := pgxpool.NewWithConfig(connectCtx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}

	// Ping the database to verify that the connection is valid and the database is reachable.
	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	// If we reach this point, the connection was successful and the database is reachable.
	return pool, nil
}

func buildDSNFromEnv() string {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "booknest")
	name := getEnv("DB_NAME", "booknest")
	sslmode := getEnv("DB_SSLMODE", "disable")

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user,
		password,
		host,
		port,
		name,
		sslmode,
	)
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
