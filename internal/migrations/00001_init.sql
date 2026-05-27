-- +goose Up

CREATE TABLE IF NOT EXISTS marquee.users (
    id           BIGSERIAL PRIMARY KEY,
    twitch_id    TEXT UNIQUE NOT NULL,
    twitch_login TEXT NOT NULL,
    display_name TEXT NOT NULL,
    avatar_url   TEXT,
    role         TEXT NOT NULL DEFAULT 'user'
                 CHECK (role IN ('user', 'streamer', 'admin', 'owner')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS marquee.sessions (
    token_hash TEXT PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES marquee.users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS sessions_user_id_idx ON marquee.sessions (user_id);
CREATE INDEX IF NOT EXISTS sessions_expires_at_idx ON marquee.sessions (expires_at);

CREATE TABLE IF NOT EXISTS marquee.movies (
    id                   BIGSERIAL PRIMARY KEY,
    tmdb_id              INTEGER UNIQUE NOT NULL,
    title                TEXT NOT NULL,
    release_year         INTEGER,
    poster_url           TEXT,
    tmdb_rating          NUMERIC(3, 1),
    genres               TEXT[] NOT NULL DEFAULT '{}',
    overview             TEXT,
    status               TEXT NOT NULL DEFAULT 'pool'
                         CHECK (status IN ('pool', 'watched', 'removed')),
    suggested_by         BIGINT NOT NULL REFERENCES marquee.users(id),
    suggested_watch_date DATE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    watched_at           TIMESTAMPTZ,
    watched_channel      TEXT
);

CREATE INDEX IF NOT EXISTS movies_status_idx ON marquee.movies (status);
CREATE INDEX IF NOT EXISTS movies_suggested_by_idx ON marquee.movies (suggested_by);

CREATE TABLE IF NOT EXISTS marquee.votes (
    id         BIGSERIAL PRIMARY KEY,
    movie_id   BIGINT NOT NULL REFERENCES marquee.movies(id) ON DELETE CASCADE,
    user_id    BIGINT NOT NULL REFERENCES marquee.users(id) ON DELETE CASCADE,
    value      SMALLINT NOT NULL CHECK (value IN (-1, 1)),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (movie_id, user_id)
);

CREATE INDEX IF NOT EXISTS votes_movie_id_idx ON marquee.votes (movie_id);

CREATE TABLE IF NOT EXISTS marquee.schedules (
    id            BIGSERIAL PRIMARY KEY,
    movie_id      BIGINT NOT NULL REFERENCES marquee.movies(id) ON DELETE CASCADE,
    streamer_id   BIGINT NOT NULL REFERENCES marquee.users(id),
    channel       TEXT NOT NULL,
    scheduled_for TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS schedules_movie_id_idx ON marquee.schedules (movie_id);
CREATE INDEX IF NOT EXISTS schedules_streamer_id_idx ON marquee.schedules (streamer_id);
CREATE INDEX IF NOT EXISTS schedules_scheduled_for_idx ON marquee.schedules (scheduled_for);

CREATE TABLE IF NOT EXISTS marquee.moderation_log (
    id             BIGSERIAL PRIMARY KEY,
    actor_id       BIGINT NOT NULL REFERENCES marquee.users(id),
    action         TEXT NOT NULL,
    movie_id       BIGINT REFERENCES marquee.movies(id),
    target_user_id BIGINT REFERENCES marquee.users(id),
    reason         TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS moderation_log_actor_id_idx ON marquee.moderation_log (actor_id);

-- +goose Down

DROP TABLE IF EXISTS marquee.moderation_log;
DROP TABLE IF EXISTS marquee.schedules;
DROP TABLE IF EXISTS marquee.votes;
DROP TABLE IF EXISTS marquee.movies;
DROP TABLE IF EXISTS marquee.sessions;
DROP TABLE IF EXISTS marquee.users;
