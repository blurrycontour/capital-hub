-- +goose Up
-- +goose StatementBegin

-- Full-text search index (FTS5) spanning collections, items and entries.
-- Each row flattens the searchable text for one entity: names, descriptions,
-- notes, amounts, custom key/values and attachment names. Stats/totals are
-- intentionally excluded. The index is kept in sync by triggers below and
-- backfilled from existing data at the end of this migration.
CREATE VIRTUAL TABLE search_fts USING fts5(
    kind UNINDEXED,          -- 'collection' | 'item' | 'entry'
    entity_id UNINDEXED,     -- id within the source table
    collection_id UNINDEXED, -- owning collection (for access checks + nav)
    item_id UNINDEXED,       -- owning item (entries only; NULL otherwise)
    title,                   -- name
    body,                    -- description / note / amount / fields / attachments
    tokenize = 'unicode61 remove_diacritics 2'
);

-- ---- Collections --------------------------------------------------------

CREATE TRIGGER collections_ai AFTER INSERT ON collections BEGIN
    INSERT INTO search_fts(kind, entity_id, collection_id, item_id, title, body)
    VALUES ('collection', NEW.id, NEW.id, NULL, NEW.name,
        NEW.description || ' ' || coalesce((
            SELECT group_concat(json_extract(je.value,'$.label') || ' ' || json_extract(je.value,'$.value'), ' ')
            FROM json_each(NEW.custom_fields) je), ''));
END;

CREATE TRIGGER collections_au AFTER UPDATE ON collections BEGIN
    DELETE FROM search_fts WHERE kind = 'collection' AND entity_id = OLD.id;
    INSERT INTO search_fts(kind, entity_id, collection_id, item_id, title, body)
    VALUES ('collection', NEW.id, NEW.id, NULL, NEW.name,
        NEW.description || ' ' || coalesce((
            SELECT group_concat(json_extract(je.value,'$.label') || ' ' || json_extract(je.value,'$.value'), ' ')
            FROM json_each(NEW.custom_fields) je), ''));
END;

CREATE TRIGGER collections_ad AFTER DELETE ON collections BEGIN
    DELETE FROM search_fts WHERE kind = 'collection' AND entity_id = OLD.id;
END;

-- ---- Items --------------------------------------------------------------

CREATE TRIGGER items_ai AFTER INSERT ON items BEGIN
    INSERT INTO search_fts(kind, entity_id, collection_id, item_id, title, body)
    VALUES ('item', NEW.id, NEW.collection_id, NULL, NEW.name,
        NEW.description || ' '
        || coalesce((SELECT group_concat(json_extract(je.value,'$.label') || ' ' || json_extract(je.value,'$.value'), ' ')
                     FROM json_each(NEW.custom_fields) je), '') || ' '
        || coalesce((SELECT group_concat(json_extract(je.value,'$.name'), ' ')
                     FROM json_each(NEW.attachments) je), ''));
END;

-- On update, re-index the item and (in case it moved collections) its entries,
-- so entry rows always carry the current collection_id.
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

CREATE TRIGGER items_ad AFTER DELETE ON items BEGIN
    DELETE FROM search_fts WHERE kind = 'item' AND entity_id = OLD.id;
END;

-- ---- Entries ------------------------------------------------------------

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

CREATE TRIGGER entries_ad AFTER DELETE ON entries BEGIN
    DELETE FROM search_fts WHERE kind = 'entry' AND entity_id = OLD.id;
END;

-- ---- Backfill existing data --------------------------------------------

INSERT INTO search_fts(kind, entity_id, collection_id, item_id, title, body)
SELECT 'collection', c.id, c.id, NULL, c.name,
    c.description || ' ' || coalesce((
        SELECT group_concat(json_extract(je.value,'$.label') || ' ' || json_extract(je.value,'$.value'), ' ')
        FROM json_each(c.custom_fields) je), '')
FROM collections c;

INSERT INTO search_fts(kind, entity_id, collection_id, item_id, title, body)
SELECT 'item', i.id, i.collection_id, NULL, i.name,
    i.description || ' '
    || coalesce((SELECT group_concat(json_extract(je.value,'$.label') || ' ' || json_extract(je.value,'$.value'), ' ')
                 FROM json_each(i.custom_fields) je), '') || ' '
    || coalesce((SELECT group_concat(json_extract(je.value,'$.name'), ' ')
                 FROM json_each(i.attachments) je), '')
FROM items i;

INSERT INTO search_fts(kind, entity_id, collection_id, item_id, title, body)
SELECT 'entry', e.id, i.collection_id, e.item_id, e.name,
    e.note || ' ' || CAST(e.amount AS TEXT) || ' '
    || coalesce((SELECT group_concat(json_extract(je.value,'$.name'), ' ')
                 FROM json_each(e.attachments) je), '')
FROM entries e JOIN items i ON i.id = e.item_id;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS entries_ad;
DROP TRIGGER IF EXISTS entries_au;
DROP TRIGGER IF EXISTS entries_ai;
DROP TRIGGER IF EXISTS items_ad;
DROP TRIGGER IF EXISTS items_au;
DROP TRIGGER IF EXISTS items_ai;
DROP TRIGGER IF EXISTS collections_ad;
DROP TRIGGER IF EXISTS collections_au;
DROP TRIGGER IF EXISTS collections_ai;
DROP TABLE IF EXISTS search_fts;
-- +goose StatementEnd
