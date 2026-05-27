package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/lsariol/marquee/internal/models"
)

// UpsertUser inserts or updates a user by Twitch ID.
// If the user's Twitch ID matches ownerTwitchID and they are not already owner, role is promoted.
func (db *DB) UpsertUser(ctx context.Context, twitchID, login, displayName, avatarURL, ownerTwitchID string) (*models.User, error) {
	const q = `
		INSERT INTO marquee.users (twitch_id, twitch_login, display_name, avatar_url)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (twitch_id) DO UPDATE SET
			twitch_login = EXCLUDED.twitch_login,
			display_name = EXCLUDED.display_name,
			avatar_url   = EXCLUDED.avatar_url,
			updated_at   = now()
		RETURNING id, twitch_id, twitch_login, display_name, avatar_url, role, created_at, updated_at`

	u := &models.User{}
	err := db.Pool.QueryRow(ctx, q, twitchID, login, displayName, avatarURL).
		Scan(&u.ID, &u.TwitchID, &u.TwitchLogin, &u.DisplayName, &u.AvatarURL, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}

	if u.TwitchID == ownerTwitchID && u.Role != "owner" {
		const promote = `UPDATE marquee.users SET role = 'owner', updated_at = now() WHERE id = $1`
		if _, err := db.Pool.Exec(ctx, promote, u.ID); err != nil {
			return nil, fmt.Errorf("promote owner: %w", err)
		}
		u.Role = "owner"
	}

	return u, nil
}

func (db *DB) GetUserByLogin(ctx context.Context, login string) (*models.User, error) {
	const q = `
		SELECT id, twitch_id, twitch_login, display_name, avatar_url, role, created_at, updated_at
		FROM marquee.users WHERE twitch_login = $1`

	u := &models.User{}
	err := db.Pool.QueryRow(ctx, q, login).
		Scan(&u.ID, &u.TwitchID, &u.TwitchLogin, &u.DisplayName, &u.AvatarURL, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by login: %w", err)
	}
	return u, nil
}

func (db *DB) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	const q = `
		SELECT id, twitch_id, twitch_login, display_name, avatar_url, role, created_at, updated_at
		FROM marquee.users WHERE id = $1`

	u := &models.User{}
	err := db.Pool.QueryRow(ctx, q, id).
		Scan(&u.ID, &u.TwitchID, &u.TwitchLogin, &u.DisplayName, &u.AvatarURL, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

func (db *DB) CreateSession(ctx context.Context, tokenHash string, userID int64, expiresAt time.Time) error {
	const q = `
		INSERT INTO marquee.sessions (token_hash, user_id, expires_at)
		VALUES ($1, $2, $3)`
	if _, err := db.Pool.Exec(ctx, q, tokenHash, userID, expiresAt); err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// GetSessionUser looks up an active session and returns its user. Returns nil, nil if not found or expired.
func (db *DB) GetSessionUser(ctx context.Context, tokenHash string) (*models.User, error) {
	const q = `
		SELECT u.id, u.twitch_id, u.twitch_login, u.display_name, u.avatar_url, u.role, u.created_at, u.updated_at
		FROM marquee.sessions s
		JOIN marquee.users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > now()`

	u := &models.User{}
	err := db.Pool.QueryRow(ctx, q, tokenHash).
		Scan(&u.ID, &u.TwitchID, &u.TwitchLogin, &u.DisplayName, &u.AvatarURL, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get session user: %w", err)
	}
	return u, nil
}

func (db *DB) DeleteSession(ctx context.Context, tokenHash string) error {
	const q = `DELETE FROM marquee.sessions WHERE token_hash = $1`
	if _, err := db.Pool.Exec(ctx, q, tokenHash); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (db *DB) DeleteExpiredSessions(ctx context.Context) error {
	const q = `DELETE FROM marquee.sessions WHERE expires_at <= now()`
	if _, err := db.Pool.Exec(ctx, q); err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}
	return nil
}

func (db *DB) ListAllUsers(ctx context.Context) ([]*models.User, error) {
	const q = `
		SELECT id, twitch_id, twitch_login, display_name, avatar_url, role, created_at, updated_at
		FROM marquee.users ORDER BY created_at ASC`

	rows, err := db.Pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list all users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		u := &models.User{}
		if err := rows.Scan(
			&u.ID, &u.TwitchID, &u.TwitchLogin, &u.DisplayName, &u.AvatarURL, &u.Role, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (db *DB) SetUserRole(ctx context.Context, userID int64, role string) error {
	const q = `UPDATE marquee.users SET role = $2, updated_at = now() WHERE id = $1`
	if _, err := db.Pool.Exec(ctx, q, userID, role); err != nil {
		return fmt.Errorf("set user role: %w", err)
	}
	return nil
}
