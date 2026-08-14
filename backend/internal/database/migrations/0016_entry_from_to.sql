-- +goose Up
-- +goose StatementBegin

-- Optional counterparty fields on an entry: who the money came from and who it
-- went to. Named `entry_from` / `entry_to` because FROM and TO are SQL
-- keywords, which would otherwise need quoting at every call site.
ALTER TABLE entries ADD COLUMN entry_from TEXT NOT NULL DEFAULT '';
ALTER TABLE entries ADD COLUMN entry_to   TEXT NOT NULL DEFAULT '';

-- Fold the new fields into the full-text index so search finds them. The
-- triggers below replace the ones created in 0014; only the entry body changes.
DROP TRIGGER IF EXISTS entries_ai;
DROP TRIGGER IF EXISTS entries_au;
DROP TRIGGER IF EXISTS items_au;

CREATE TRIGGER entries_ai AFTER INSERT ON entries BEGIN
    INSERT INTO search_fts(kind, entity_id, collection_id, item_id, title, body)
    VALUES ('entry', NEW.id,
        (SELECT collection_id FROM items WHERE id = NEW.item_id),
        NEW.item_id, NEW.name,
        NEW.note || ' ' || CAST(NEW.amount AS TEXT) || ' '
        || NEW.entry_from || ' ' || NEW.entry_to || ' '
        || coalesce((SELECT group_concat(json_extract(je.value,'$.name'), ' ')
                     FROM json_each(NEW.attachments) je), ''));
END;

CREATE TRIGGER entries_au AFTER UPDATE ON entries BEGIN
    DELETE FROM search_fts WHERE kind = 'entry' AND entity_id = OLD.id;
    INSERT INTO search_fts(kind, entity_id, collection_id, item_id, title, body)
    VALUES ('entry', NEW.id,
        (SELECT collection_id FROM items WHERE id = NEW.item_id),
        NEW.item_id, NEW.name,
        NEW.note || ' ' || CAST(NEW.amount AS TEXT) || ' '
        || NEW.entry_from || ' ' || NEW.entry_to || ' '
        || coalesce((SELECT group_concat(json_extract(je.value,'$.name'), ' ')
                     FROM json_each(NEW.attachments) je), ''));
END;

-- Moving an item between collections re-indexes its entries, so this one has
-- to learn the new body shape too.
CREATE TRIGGER items_au AFTER UPDATE ON items BEGIN
    DELETE FROM search_fts WHERE kind = 'item' AND entity_id = OLD.id;
    INSERT INTO search_fts(kind, entity_id, collection_id, item_id, title, body)
    VALUES ('item', NEW.id, NEW.collection_id, NULL, NEW.name,
        NEW.description || ' '
        || coalesce((SELECT group_concat(json_extract(je.value,'$.label') || ' ' || json_extract(je.value,'$.value'), ' ')
                     FROM json_each(NEW.custom_fields) je), '') || ' '
        || coalesce((SELECT group_concat(json_extract(je.value,'$.name'), ' ')
                     FROM json_each(NEW.attachments) je), ''));
    DELETE FROM search_fts WHERE kind = 'entry' AND item_id = OLD.id;
    INSERT INTO search_fts(kind, entity_id, collection_id, item_id, title, body)
    SELECT 'entry', e.id, NEW.collection_id, e.item_id, e.name,
        e.note || ' ' || CAST(e.amount AS TEXT) || ' '
        || e.entry_from || ' ' || e.entry_to || ' '
        || coalesce((SELECT group_concat(json_extract(je.value,'$.name'), ' ')
                     FROM json_each(e.attachments) je), '')
    FROM entries e WHERE e.item_id = NEW.id;
END;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- SQLite cannot drop a column without rebuilding the table; leaving the columns
-- in place is harmless because they default to ''. Only the triggers revert.
DROP TRIGGER IF EXISTS entries_ai;
DROP TRIGGER IF EXISTS entries_au;
DROP TRIGGER IF EXISTS items_au;

CREATE TRIGGER entries_ai AFTER INSERT ON entries BEGIN
    INSERT INTO search_fts(kind, entity_id, collection_id, item_id, title, body)
    VALUES ('entry', NEW.id,
        (SELECT collection_id FROM items WHERE id = NEW.item_id),
        NEW.item_id, NEW.name,
        NEW.note || ' ' || CAST(NEW.amount AS TEXT) || ' '
        || coalesce((SELECT group_concat(json_extract(je.value,'$.name'), ' ')
                     FROM json_each(NEW.attachments) je), ''));
END;

CREATE TRIGGER entries_au AFTER UPDATE ON entries BEGIN
    DELETE FROM search_fts WHERE kind = 'entry' AND entity_id = OLD.id;
    INSERT INTO search_fts(kind, entity_id, collection_id, item_id, title, body)
    VALUES ('entry', NEW.id,
        (SELECT collection_id FROM items WHERE id = NEW.item_id),
        NEW.item_id, NEW.name,
        NEW.note || ' ' || CAST(NEW.amount AS TEXT) || ' '
        || coalesce((SELECT group_concat(json_extract(je.value,'$.name'), ' ')
                     FROM json_each(NEW.attachments) je), ''));
END;

CREATE TRIGGER items_au AFTER UPDATE ON items BEGIN
    DELETE FROM search_fts WHERE kind = 'item' AND entity_id = OLD.id;
    INSERT INTO search_fts(kind, entity_id, collection_id, item_id, title, body)
    VALUES ('item', NEW.id, NEW.collection_id, NULL, NEW.name,
        NEW.description || ' '
        || coalesce((SELECT group_concat(json_extract(je.value,'$.label') || ' ' || json_extract(je.value,'$.value'), ' ')
                     FROM json_each(NEW.custom_fields) je), '') || ' '
        || coalesce((SELECT group_concat(json_extract(je.value,'$.name'), ' ')
                     FROM json_each(NEW.attachments) je), ''));
    DELETE FROM search_fts WHERE kind = 'entry' AND item_id = OLD.id;
    INSERT INTO search_fts(kind, entity_id, collection_id, item_id, title, body)
    SELECT 'entry', e.id, NEW.collection_id, e.item_id, e.name,
        e.note || ' ' || CAST(e.amount AS TEXT) || ' '
        || coalesce((SELECT group_concat(json_extract(je.value,'$.name'), ' ')
                     FROM json_each(e.attachments) je), '')
    FROM entries e WHERE e.item_id = NEW.id;
END;
-- +goose StatementEnd
