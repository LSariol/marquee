package database

import (
	"context"
	"fmt"
)

func (db *DB) LogModeration(ctx context.Context, actorID int64, action string, movieID, targetUserID *int64, reason *string) error {
	const q = `
		INSERT INTO marquee.moderation_log (actor_id, action, movie_id, target_user_id, reason)
		VALUES ($1, $2, $3, $4, $5)`
	if _, err := db.Pool.Exec(ctx, q, actorID, action, movieID, targetUserID, reason); err != nil {
		return fmt.Errorf("log moderation: %w", err)
	}
	return nil
}
