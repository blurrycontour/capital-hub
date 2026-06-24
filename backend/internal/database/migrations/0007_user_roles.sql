-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'editor';
UPDATE users SET role = 'administrator' WHERE is_admin = 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- SQLite supports DROP COLUMN since 3.35.0
ALTER TABLE users DROP COLUMN role;
-- +goose StatementEnd
