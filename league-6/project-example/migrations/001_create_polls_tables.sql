-- +goose Up
CREATE TABLE polls (
    id         TEXT        PRIMARY KEY,
    question   TEXT        NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE poll_options (
    id      TEXT    PRIMARY KEY,
    poll_id TEXT    NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    text    TEXT    NOT NULL,
    votes   INTEGER NOT NULL DEFAULT 0
);

-- Один IP — один голос в каждом опросе
CREATE TABLE poll_votes (
    poll_id TEXT NOT NULL,
    ip      TEXT NOT NULL,
    PRIMARY KEY (poll_id, ip)
);

-- +goose Down
DROP TABLE IF EXISTS poll_votes;
DROP TABLE IF EXISTS poll_options;
DROP TABLE IF EXISTS polls;
