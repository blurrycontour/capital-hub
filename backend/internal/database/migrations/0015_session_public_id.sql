-- +goose Up
-- +goose StatementBegin
-- Add a public, non-secret identifier to sessions so the account UI can list
-- and revoke sessions without ever exposing the real session token (which is
-- the bearer credential in the cookie).
ALTER TABLE sessions ADD COLUMN public_id TEXT NOT NULL DEFAULT '';
UPDATE sessions SET public_id = lower(hex(randomblob(16))) WHERE public_id = '';
CREATE UNIQUE INDEX idx_sessions_public_id ON sessions (public_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_sessions_public_id;
-- Column left in place: SQLite (pre-3.35) cannot DROP COLUMN.
-- +goose StatementEnd
