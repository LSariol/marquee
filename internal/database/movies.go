package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lsariol/marquee/internal/models"
	"github.com/lsariol/marquee/internal/tmdb"
)

// InsertMovie creates a new movie from TMDB details. Returns (movie, alreadyExists, error).
func (db *DB) InsertMovie(ctx context.Context, details *tmdb.MovieDetails, suggestedBy int64, suggestedWatchDate *string) (*models.Movie, bool, error) {
	const q = `
		INSERT INTO marquee.movies
			(tmdb_id, title, release_year, poster_url, tmdb_rating, genres, overview, suggested_by, suggested_watch_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8,
			CASE WHEN $9::text IS NULL OR $9 = '' THEN NULL
			     ELSE $9::date END)
		RETURNING id, tmdb_id, title, release_year, poster_url, tmdb_rating, genres, overview,
		          status, suggested_by, suggested_watch_date, created_at`

	m := &models.Movie{}
	err := db.Pool.QueryRow(ctx, q,
		details.ID,
		details.Title,
		details.ReleaseYear(),
		details.PosterURL(),
		details.Rating(),
		details.GenreNames(),
		nullableString(details.Overview),
		suggestedBy,
		suggestedWatchDate,
	).Scan(
		&m.ID, &m.TMDBId, &m.Title, &m.ReleaseYear, &m.PosterURL, &m.TMDBRating,
		&m.Genres, &m.Overview, &m.Status, &m.SuggestedBy, &m.SuggestedWatchDate, &m.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, true, nil
		}
		return nil, false, fmt.Errorf("insert movie: %w", err)
	}
	return m, false, nil
}

func (db *DB) GetMovieByTMDBID(ctx context.Context, tmdbID int) (*models.Movie, error) {
	const q = `
		SELECT id, tmdb_id, title, release_year, poster_url, tmdb_rating, genres, overview,
		       status, suggested_by, suggested_watch_date, created_at
		FROM marquee.movies WHERE tmdb_id = $1`

	m := &models.Movie{}
	err := db.Pool.QueryRow(ctx, q, tmdbID).Scan(
		&m.ID, &m.TMDBId, &m.Title, &m.ReleaseYear, &m.PosterURL, &m.TMDBRating,
		&m.Genres, &m.Overview, &m.Status, &m.SuggestedBy, &m.SuggestedWatchDate, &m.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get movie by tmdb id: %w", err)
	}
	return m, nil
}

const poolSelect = `
	SELECT
		m.id, m.title, m.release_year, m.poster_url, m.tmdb_rating, m.genres,
		m.suggested_watch_date,
		u.twitch_login, u.display_name,
		COALESCE(SUM(v.value), 0) AS net_votes,
		COALESCE(uv.value, 0) AS user_vote,
		ws.scheduled_for,
		ws.channel
	FROM marquee.movies m
	JOIN marquee.users u ON u.id = m.suggested_by
	LEFT JOIN marquee.votes v ON v.movie_id = m.id
	LEFT JOIN marquee.votes uv ON uv.movie_id = m.id AND uv.user_id = $1
	LEFT JOIN LATERAL (
		SELECT scheduled_for, channel
		FROM marquee.schedules
		WHERE movie_id = m.id AND scheduled_for >= now()
		ORDER BY scheduled_for ASC
		LIMIT 1
	) ws ON true`

func scanPoolMovie(row interface{ Scan(...any) error }) (models.PoolMovie, error) {
	var p models.PoolMovie
	err := row.Scan(
		&p.ID, &p.Title, &p.ReleaseYear, &p.PosterURL, &p.TMDBRating, &p.Genres,
		&p.SuggestedWatchDate,
		&p.SuggesterLogin, &p.SuggesterDisplay,
		&p.NetVotes, &p.UserVote,
		&p.WatchingSoon, &p.WatchingChannel,
	)
	return p, err
}

// ListPool returns all pool movies ranked by net votes. userID 0 = not logged in.
func (db *DB) ListPool(ctx context.Context, userID int64) ([]models.PoolMovie, error) {
	q := poolSelect + `
		WHERE m.status = 'pool'
		GROUP BY m.id, m.title, m.release_year, m.poster_url, m.tmdb_rating, m.genres,
		         m.suggested_watch_date, m.created_at, u.twitch_login, u.display_name, uv.value,
		         ws.scheduled_for, ws.channel
		ORDER BY net_votes DESC, m.created_at ASC`

	rows, err := db.Pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list pool: %w", err)
	}
	defer rows.Close()

	var pool []models.PoolMovie
	for rows.Next() {
		p, err := scanPoolMovie(rows)
		if err != nil {
			return nil, fmt.Errorf("scan pool row: %w", err)
		}
		pool = append(pool, p)
	}
	return pool, rows.Err()
}

// GetPoolMovieRow returns a single pool movie row with aggregated votes.
func (db *DB) GetPoolMovieRow(ctx context.Context, movieID, userID int64) (*models.PoolMovie, error) {
	q := poolSelect + `
		WHERE m.id = $2 AND m.status = 'pool'
		GROUP BY m.id, m.title, m.release_year, m.poster_url, m.tmdb_rating, m.genres,
		         m.suggested_watch_date, m.created_at, u.twitch_login, u.display_name, uv.value,
		         ws.scheduled_for, ws.channel`

	p, err := scanPoolMovie(db.Pool.QueryRow(ctx, q, userID, movieID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get pool movie row: %w", err)
	}
	return &p, nil
}

// MarkWatched sets a pool movie to watched. Returns false if movie not found or not in pool.
func (db *DB) MarkWatched(ctx context.Context, movieID int64, channel string) (bool, error) {
	const q = `
		UPDATE marquee.movies
		SET status = 'watched', watched_at = now(), watched_channel = $2
		WHERE id = $1 AND status = 'pool'`
	tag, err := db.Pool.Exec(ctx, q, movieID, channel)
	if err != nil {
		return false, fmt.Errorf("mark watched: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// MarkBackToPool reverts a watched movie to pool. Returns false if not found or not watched.
func (db *DB) MarkBackToPool(ctx context.Context, movieID int64) (bool, error) {
	const q = `
		UPDATE marquee.movies
		SET status = 'pool', watched_at = NULL, watched_channel = NULL
		WHERE id = $1 AND status = 'watched'`
	tag, err := db.Pool.Exec(ctx, q, movieID)
	if err != nil {
		return false, fmt.Errorf("mark back to pool: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// MarkRemoved sets a movie to removed. Returns false if not found or already removed.
func (db *DB) MarkRemoved(ctx context.Context, movieID int64) (bool, error) {
	const q = `
		UPDATE marquee.movies
		SET status = 'removed'
		WHERE id = $1 AND status != 'removed'`
	tag, err := db.Pool.Exec(ctx, q, movieID)
	if err != nil {
		return false, fmt.Errorf("mark removed: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (db *DB) ListWatched(ctx context.Context) ([]models.WatchedMovie, error) {
	const q = `
		SELECT m.id, m.title, m.release_year, m.poster_url, m.tmdb_rating, m.genres,
		       u.twitch_login, u.display_name,
		       m.watched_at, m.watched_channel
		FROM marquee.movies m
		JOIN marquee.users u ON u.id = m.suggested_by
		WHERE m.status = 'watched'
		ORDER BY m.watched_at DESC`

	rows, err := db.Pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list watched: %w", err)
	}
	defer rows.Close()

	var result []models.WatchedMovie
	for rows.Next() {
		var m models.WatchedMovie
		if err := rows.Scan(
			&m.ID, &m.Title, &m.ReleaseYear, &m.PosterURL, &m.TMDBRating, &m.Genres,
			&m.SuggesterLogin, &m.SuggesterDisplay,
			&m.WatchedAt, &m.WatchedChannel,
		); err != nil {
			return nil, fmt.Errorf("scan watched movie: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
