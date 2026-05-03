-- +goose Up
CREATE TABLE task_activities (
    id          BIGSERIAL PRIMARY KEY,
    task_id     BIGINT       NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id     BIGINT       NOT NULL REFERENCES users(id),
    action      TEXT         NOT NULL,
    field_name  TEXT,
    old_value   TEXT,
    new_value   TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX task_activities_task_id_idx ON task_activities(task_id);
CREATE INDEX task_activities_user_id_idx ON task_activities(user_id);
CREATE INDEX task_activities_created_at_idx ON task_activities(created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS task_activities;
