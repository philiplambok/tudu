-- +goose Up
ALTER TABLE tasks
    ALTER COLUMN due_date TYPE DATE USING due_date::DATE;

-- +goose Down
ALTER TABLE tasks
    ALTER COLUMN due_date TYPE TIMESTAMPTZ USING due_date::TIMESTAMPTZ;
