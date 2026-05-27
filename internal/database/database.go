package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lsariol/marquee/internal/migrations"
	"github.com/pressly/goose/v3"
)

func RunMigrations(migratorURL string) error {
	slog.Info("running migrations")

	db, err := sql.Open("pgx", migratorURL)
	if err != nil {
		return fmt.Errorf("open migrator connection: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping migrator: %w", err)
	}

	if _, err := db.Exec("SET search_path TO marquee, public"); err != nil {
		return fmt.Errorf("set search_path: %w", err)
	}

	goose.SetBaseFS(migrations.FS)
	goose.SetTableName("marquee.goose_db_version")

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose set dialect: %w", err)
	}

	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	slog.Info("migrations complete")
	return nil
}

func OpenPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	var pool *pgxpool.Pool
	var err error

	for attempt := 1; attempt <= 10; attempt++ {
		pool, err = pgxpool.New(ctx, databaseURL)
		if err == nil {
			if pingErr := pool.Ping(ctx); pingErr == nil {
				slog.Info("database pool ready")
				return pool, nil
			} else {
				pool.Close()
				err = pingErr
			}
		}
		slog.Warn("database not ready, retrying", "attempt", attempt, "error", err)
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("could not connect to database after 10 attempts: %w", err)
}

type DB struct {
	Pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *DB {
	return &DB{Pool: pool}
}
