-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS oidc_identities (
    id         INTEGER PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    provider   TEXT    NOT NULL,
    subject    TEXT    NOT NULL,
    created_at TEXT    NOT NULL DEFAULT (datetime('now')),
    UNIQUE(provider, subject),
    UNIQUE(user_id, provider)
);
CREATE INDEX IF NOT EXISTS idx_oidc_identities_user_id ON oidc_identities (user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS oidc_identities;
-- +goose StatementEnd
