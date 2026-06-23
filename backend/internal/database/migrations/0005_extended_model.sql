-- +goose Up
-- +goose StatementBegin

-- Collections gain a per-collection currency, optional geolocation (shown on a
-- map when the collection is opened), and user-defined custom fields.
ALTER TABLE collections ADD COLUMN currency       TEXT NOT NULL DEFAULT 'USD';
ALTER TABLE collections ADD COLUMN location_lat   REAL;
ALTER TABLE collections ADD COLUMN location_lng   REAL;
ALTER TABLE collections ADD COLUMN location_label TEXT NOT NULL DEFAULT '';
ALTER TABLE collections ADD COLUMN custom_fields  TEXT NOT NULL DEFAULT '[]';

-- Items gain file attachments and user-defined custom fields.
ALTER TABLE items ADD COLUMN attachments   TEXT NOT NULL DEFAULT '[]';
ALTER TABLE items ADD COLUMN custom_fields TEXT NOT NULL DEFAULT '[]';

-- Entries gain a name and file attachments. Their currency is inherited from
-- the owning collection at creation time.
ALTER TABLE entries ADD COLUMN name        TEXT NOT NULL DEFAULT '';
ALTER TABLE entries ADD COLUMN attachments TEXT NOT NULL DEFAULT '[]';

-- Users gain an optional avatar (profile picture) path.
ALTER TABLE users ADD COLUMN avatar_path TEXT NOT NULL DEFAULT '';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- SQLite (pre-3.35) cannot DROP COLUMN; columns are left in place on rollback.
-- +goose StatementEnd
