package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// GetVote returns the current user's vote value for a movie, or 0 if none.
func (db *DB) GetVote(ctx context.Context, movieID, userID int64) (int, error) {
	const q = `SELECT value FROM marquee.votes WHERE movie_id = $1 AND user_id = $2`
	var value int
	err := db.Pool.QueryRow(ctx, q, movieID, userID).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get vote: %w", err)
	}
	return value, nil
}

func (db *DB) UpsertVote(ctx context.Context, movieID, userID int64, value int) error {
	const q = `
		INSERT INTO marquee.votes (movie_id, user_id, value)
		VALUES ($1, $2, $3)
		ON CONFLICT (movie_id, user_id) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`
	if _, err := db.Pool.Exec(ctx, q, movieID, userID, value); err != nil {
		return fmt.Errorf("upsert vote: %w", err)
	}
	return nil
}

func (db *DB) DeleteVote(ctx context.Context, movieID, userID int64) error {
	const q = `DELETE FROM marquee.votes WHERE movie_id = $1 AND user_id = $2`
	if _, err := db.Pool.Exec(ctx, q, movieID, userID); err != nil {
		return fmt.Errorf("delete vote: %w", err)
	}
	return nil
}
