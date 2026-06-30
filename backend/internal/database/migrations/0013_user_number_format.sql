-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN number_format TEXT NOT NULL DEFAULT 'international';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN number_format;
-- +goose StatementEnd
