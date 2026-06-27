-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN amount_decimals INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN notify_collection_shared INTEGER NOT NULL DEFAULT 1;
ALTER TABLE users ADD COLUMN notify_item_added INTEGER NOT NULL DEFAULT 1;
ALTER TABLE users ADD COLUMN notify_entry_added INTEGER NOT NULL DEFAULT 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN amount_decimals;
ALTER TABLE users DROP COLUMN notify_collection_shared;
ALTER TABLE users DROP COLUMN notify_item_added;
ALTER TABLE users DROP COLUMN notify_entry_added;
-- +goose StatementEnd
