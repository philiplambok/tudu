-- +goose Up
ALTER TABLE users DROP COLUMN avatar_url;

-- +goose Down
ALTER TABLE users ADD COLUMN avatar_url TEXT NOT NULL DEFAULT '';
