-- +goose Up
CREATE TABLE tasks (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT       NOT NULL REFERENCES users(id),
    title        TEXT         NOT NULL,
    description  TEXT,
    status       TEXT         NOT NULL DEFAULT 'pending',
    due_date     DATE,
    completed_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX tasks_user_id_idx ON tasks(user_id);
CREATE INDEX tasks_status_idx  ON tasks(status);

-- +goose Down
DROP TABLE IF EXISTS tasks;
