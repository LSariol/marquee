package database

import (
	"context"
	"fmt"

	"github.com/lsariol/marquee/internal/models"
)

func (db *DB) ListUserSuggestions(ctx context.Context, userID int64) ([]models.UserSuggestion, error) {
	const q = `
		SELECT m.id, m.title, m.release_year, m.poster_url, m.status,
		       COALESCE(SUM(v.value), 0) AS net_votes,
		       m.watched_at
		FROM marquee.movies m
		LEFT JOIN marquee.votes v ON v.movie_id = m.id
		WHERE m.suggested_by = $1 AND m.status != 'removed'
		GROUP BY m.id, m.title, m.release_year, m.poster_url, m.status, m.watched_at, m.created_at
		ORDER BY m.created_at DESC`

	rows, err := db.Pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list user suggestions: %w", err)
	}
	defer rows.Close()

	var result []models.UserSuggestion
	for rows.Next() {
		var s models.UserSuggestion
		if err := rows.Scan(
			&s.ID, &s.Title, &s.ReleaseYear, &s.PosterURL, &s.Status, &s.NetVotes, &s.WatchedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user suggestion: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (db *DB) ListUserVotes(ctx context.Context, userID int64) ([]models.UserVote, error) {
	const q = `
		SELECT m.id, m.title, m.release_year, m.poster_url, uv.value, m.status
		FROM marquee.votes uv
		JOIN marquee.movies m ON m.id = uv.movie_id
		WHERE uv.user_id = $1 AND m.status != 'removed'
		ORDER BY uv.updated_at DESC`

	rows, err := db.Pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list user votes: %w", err)
	}
	defer rows.Close()

	var result []models.UserVote
	for rows.Next() {
		var v models.UserVote
		if err := rows.Scan(
			&v.MovieID, &v.Title, &v.ReleaseYear, &v.PosterURL, &v.Value, &v.Status,
		); err != nil {
			return nil, fmt.Errorf("scan user vote: %w", err)
		}
		result = append(result, v)
	}
	return result, rows.Err()
}
