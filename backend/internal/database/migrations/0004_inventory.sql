-- +goose Up
-- +goose StatementBegin
CREATE TABLE collections (
    id          INTEGER PRIMARY KEY,
    user_id     INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name        TEXT    NOT NULL,
    description TEXT    NOT NULL DEFAULT '',
    created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    created_by  INTEGER REFERENCES users (id) ON DELETE SET NULL,
    updated_by  INTEGER REFERENCES users (id) ON DELETE SET NULL
);
CREATE INDEX idx_collections_user ON collections (user_id, name);

CREATE TABLE items (
    id             INTEGER PRIMARY KEY,
    collection_id  INTEGER NOT NULL REFERENCES collections (id) ON DELETE CASCADE,
    name           TEXT    NOT NULL,
    description    TEXT    NOT NULL DEFAULT '',
    image_path     TEXT    NOT NULL DEFAULT '',
    location_lat   REAL,
    location_lng   REAL,
    location_label TEXT    NOT NULL DEFAULT '',
    created_at     TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at     TEXT    NOT NULL DEFAULT (datetime('now')),
    created_by     INTEGER REFERENCES users (id) ON DELETE SET NULL,
    updated_by     INTEGER REFERENCES users (id) ON DELETE SET NULL
);
CREATE INDEX idx_items_collection ON items (collection_id, name);

CREATE TABLE entries (
    id          INTEGER PRIMARY KEY,
    item_id     INTEGER NOT NULL REFERENCES items (id) ON DELETE CASCADE,
    kind        TEXT    NOT NULL DEFAULT 'valuation',
    amount      REAL    NOT NULL DEFAULT 0,
    currency    TEXT    NOT NULL DEFAULT 'USD',
    quantity    REAL    NOT NULL DEFAULT 1,
    note        TEXT    NOT NULL DEFAULT '',
    occurred_on TEXT    NOT NULL DEFAULT (date('now')),
    created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    created_by  INTEGER REFERENCES users (id) ON DELETE SET NULL,
    updated_by  INTEGER REFERENCES users (id) ON DELETE SET NULL
);
CREATE INDEX idx_entries_item ON entries (item_id, occurred_on DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS entries;
DROP TABLE IF EXISTS items;
DROP TABLE IF EXISTS collections;
-- +goose StatementEnd
