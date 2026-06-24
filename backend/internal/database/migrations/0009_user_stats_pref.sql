-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN include_shared_in_stats INTEGER NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN include_shared_in_stats;
-- +goose StatementEnd
