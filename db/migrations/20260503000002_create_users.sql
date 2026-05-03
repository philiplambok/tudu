-- +goose Up
CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    email         CITEXT    UNIQUE NOT NULL,
    password_hash TEXT             NOT NULL,
    avatar_url    TEXT             NOT NULL,
    created_at    TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS users;
