package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/lsariol/marquee/internal/models"
)

func (db *DB) InsertSchedule(ctx context.Context, movieID, streamerID int64, channel string, scheduledFor time.Time) (*models.Schedule, error) {
	const q = `
		INSERT INTO marquee.schedules (movie_id, streamer_id, channel, scheduled_for)
		VALUES ($1, $2, $3, $4)
		RETURNING id, movie_id, streamer_id, channel, scheduled_for, created_at, updated_at`

	s := &models.Schedule{}
	err := db.Pool.QueryRow(ctx, q, movieID, streamerID, channel, scheduledFor).Scan(
		&s.ID, &s.MovieID, &s.StreamerID, &s.Channel, &s.ScheduledFor, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert schedule: %w", err)
	}
	return s, nil
}

func (db *DB) GetSchedule(ctx context.Context, scheduleID int64) (*models.Schedule, error) {
	const q = `
		SELECT id, movie_id, streamer_id, channel, scheduled_for, created_at, updated_at
		FROM marquee.schedules WHERE id = $1`

	s := &models.Schedule{}
	err := db.Pool.QueryRow(ctx, q, scheduleID).Scan(
		&s.ID, &s.MovieID, &s.StreamerID, &s.Channel, &s.ScheduledFor, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get schedule: %w", err)
	}
	return s, nil
}

func (db *DB) ListSchedulesForMovie(ctx context.Context, movieID int64) ([]models.ScheduleSimple, error) {
	const q = `
		SELECT s.id, s.channel, s.scheduled_for, u.twitch_login, u.display_name
		FROM marquee.schedules s
		JOIN marquee.users u ON u.id = s.streamer_id
		WHERE s.movie_id = $1
		ORDER BY s.scheduled_for ASC`

	rows, err := db.Pool.Query(ctx, q, movieID)
	if err != nil {
		return nil, fmt.Errorf("list schedules for movie: %w", err)
	}
	defer rows.Close()

	var result []models.ScheduleSimple
	for rows.Next() {
		var s models.ScheduleSimple
		if err := rows.Scan(&s.ID, &s.Channel, &s.ScheduledFor, &s.StreamerLogin, &s.StreamerDisplay); err != nil {
			return nil, fmt.Errorf("scan schedule simple: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (db *DB) ListSchedules(ctx context.Context) ([]models.ScheduleEntry, error) {
	const q = `
		SELECT s.id, s.movie_id, m.title, m.poster_url, s.channel, s.scheduled_for,
		       u.id, u.twitch_login, u.display_name
		FROM marquee.schedules s
		JOIN marquee.movies m ON m.id = s.movie_id
		JOIN marquee.users u ON u.id = s.streamer_id
		ORDER BY s.scheduled_for ASC`

	rows, err := db.Pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list schedules: %w", err)
	}
	defer rows.Close()

	var result []models.ScheduleEntry
	for rows.Next() {
		var e models.ScheduleEntry
		if err := rows.Scan(
			&e.ID, &e.MovieID, &e.MovieTitle, &e.PosterURL,
			&e.Channel, &e.ScheduledFor,
			&e.StreamerID, &e.StreamerLogin, &e.StreamerDisplay,
		); err != nil {
			return nil, fmt.Errorf("scan schedule entry: %w", err)
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func (db *DB) DeleteSchedule(ctx context.Context, scheduleID int64) error {
	const q = `DELETE FROM marquee.schedules WHERE id = $1`
	if _, err := db.Pool.Exec(ctx, q, scheduleID); err != nil {
		return fmt.Errorf("delete schedule: %w", err)
	}
	return nil
}
