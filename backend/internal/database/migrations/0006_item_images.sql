-- +goose Up
-- +goose StatementBegin
-- Items can hold multiple images, shown as a slideshow on the item page. The
-- existing single image_path remains as the cover (first) image for thumbnails.
ALTER TABLE items ADD COLUMN images TEXT NOT NULL DEFAULT '[]';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE items DROP COLUMN images;
-- +goose StatementEnd
