-- +goose Up
-- +goose StatementBegin
CREATE TABLE collection_shares (
    id            INTEGER PRIMARY KEY,
    collection_id INTEGER NOT NULL REFERENCES collections (id) ON DELETE CASCADE,
    user_id       INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    access        TEXT    NOT NULL DEFAULT 'read',  -- 'read' or 'write'
    created_at    TEXT    NOT NULL DEFAULT (datetime('now')),
    UNIQUE(collection_id, user_id)
);
CREATE INDEX idx_collection_shares_user_id ON collection_shares (user_id);
CREATE INDEX idx_collection_shares_collection_id ON collection_shares (collection_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS collection_shares;
-- +goose StatementEnd
