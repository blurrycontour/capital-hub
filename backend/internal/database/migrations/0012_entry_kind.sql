-- +goose Up
-- +goose StatementBegin
-- The `kind` column already exists (created in 0004 with default 'valuation').
-- Repurpose it for debit/credit accounting: normalize any legacy/unknown values
-- to 'debit' so every entry has a valid accounting kind.
UPDATE entries SET kind = 'debit' WHERE kind NOT IN ('debit', 'credit');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- No-op: original 'valuation' values are not recoverable and the column predates
-- this migration.
SELECT 1;
-- +goose StatementEnd
