CREATE TABLE users (
    id           TEXT PRIMARY KEY,
    email        TEXT NOT NULL UNIQUE,
    display_name TEXT,
    avatar_url   TEXT,
    currency     TEXT NOT NULL DEFAULT 'USD',
    timezone     TEXT NOT NULL DEFAULT 'UTC',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);
