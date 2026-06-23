// Package inventory implements the core domain: collections that hold items,
// and items that hold transaction entries. All operations are scoped to the
// owning user so a caller can only ever read or mutate their own data.
package inventory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrNotFound is returned when a record does not exist or is not owned by the
// requesting user.
var ErrNotFound = errors.New("not found")

// Service provides CRUD and statistics operations backed by SQLite.
type Service struct {
	db *sql.DB
}

// NewService builds an inventory service.
func NewService(db *sql.DB) *Service { return &Service{db: db} }

// CustomField is a user-defined label/value pair attached to a collection or
// item, allowing arbitrary extra metadata.
type CustomField struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// Attachment is a file uploaded against an item or entry.
type Attachment struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// marshalJSONField serialises a slice for storage in a TEXT column, falling
// back to an empty JSON array on error so the column is never NULL/invalid.
func marshalJSONField(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func unmarshalCustomFields(s string) []CustomField {
	out := []CustomField{}
	if strings.TrimSpace(s) == "" {
		return out
	}
	_ = json.Unmarshal([]byte(s), &out)
	if out == nil {
		return []CustomField{}
	}
	return out
}

func unmarshalAttachments(s string) []Attachment {
	out := []Attachment{}
	if strings.TrimSpace(s) == "" {
		return out
	}
	_ = json.Unmarshal([]byte(s), &out)
	if out == nil {
		return []Attachment{}
	}
	return out
}

func normalizeCustomFields(in []CustomField) []CustomField {
	out := make([]CustomField, 0, len(in))
	for _, f := range in {
		label := strings.TrimSpace(f.Label)
		if label == "" {
			continue
		}
		out = append(out, CustomField{Label: label, Value: strings.TrimSpace(f.Value)})
	}
	return out
}

func normalizeAttachments(in []Attachment) []Attachment {
	out := make([]Attachment, 0, len(in))
	for _, a := range in {
		if strings.TrimSpace(a.Path) == "" {
			continue
		}
		name := strings.TrimSpace(a.Name)
		if name == "" {
			name = a.Path
		}
		out = append(out, Attachment{Name: name, Path: a.Path})
	}
	return out
}

// Collection is a named group of items owned by a user.
type Collection struct {
	ID            int64         `json:"id"`
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	Currency      string        `json:"currency"`
	LocationLat   *float64      `json:"locationLat"`
	LocationLng   *float64      `json:"locationLng"`
	LocationLabel string        `json:"locationLabel"`
	CustomFields  []CustomField `json:"customFields"`
	CreatedAt     string        `json:"createdAt"`
	UpdatedAt     string        `json:"updatedAt"`
	CreatedBy     string        `json:"createdBy"`
	UpdatedBy     string        `json:"updatedBy"`
	ItemCount     int           `json:"itemCount"`
}

// Item is a single asset within a collection.
type Item struct {
	ID            int64         `json:"id"`
	CollectionID  int64         `json:"collectionId"`
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	ImagePath     string        `json:"imagePath"`
	LocationLat   *float64      `json:"locationLat"`
	LocationLng   *float64      `json:"locationLng"`
	LocationLabel string        `json:"locationLabel"`
	Attachments   []Attachment  `json:"attachments"`
	CustomFields  []CustomField `json:"customFields"`
	CreatedAt     string        `json:"createdAt"`
	UpdatedAt     string        `json:"updatedAt"`
	CreatedBy     string        `json:"createdBy"`
	UpdatedBy     string        `json:"updatedBy"`
	EntryCount    int           `json:"entryCount"`
}

// Entry is a single transaction recorded against an item. Its currency is
// inherited from the owning collection.
type Entry struct {
	ID          int64        `json:"id"`
	ItemID      int64        `json:"itemId"`
	Name        string       `json:"name"`
	Amount      float64      `json:"amount"`
	Currency    string       `json:"currency"`
	Note        string       `json:"note"`
	OccurredOn  string       `json:"occurredOn"`
	Attachments []Attachment `json:"attachments"`
	CreatedAt   string       `json:"createdAt"`
	UpdatedAt   string       `json:"updatedAt"`
	CreatedBy   string       `json:"createdBy"`
	UpdatedBy   string       `json:"updatedBy"`
}

// CurrencyTotal is the aggregated value for a single currency.
type CurrencyTotal struct {
	Currency string  `json:"currency"`
	Total    float64 `json:"total"`
	Entries  int     `json:"entries"`
}

// Stats summarises a collection or item.
type Stats struct {
	ItemCount  int             `json:"itemCount"`
	EntryCount int             `json:"entryCount"`
	Totals     []CurrencyTotal `json:"totals"`
}

// ---------- Collections ----------

const collectionSelect = `
SELECT c.id, c.name, c.description, c.currency,
       c.location_lat, c.location_lng, c.location_label, c.custom_fields,
       c.created_at, c.updated_at,
       COALESCE(cu.display_name, cu.username, ''),
       COALESCE(uu.display_name, uu.username, ''),
       (SELECT COUNT(*) FROM items i WHERE i.collection_id = c.id)
FROM collections c
LEFT JOIN users cu ON cu.id = c.created_by
LEFT JOIN users uu ON uu.id = c.updated_by
`

func scanCollection(s interface{ Scan(...any) error }) (Collection, error) {
	var c Collection
	var customFields string
	if err := s.Scan(&c.ID, &c.Name, &c.Description, &c.Currency,
		&c.LocationLat, &c.LocationLng, &c.LocationLabel, &customFields,
		&c.CreatedAt, &c.UpdatedAt, &c.CreatedBy, &c.UpdatedBy, &c.ItemCount); err != nil {
		return Collection{}, err
	}
	c.CustomFields = unmarshalCustomFields(customFields)
	return c, nil
}

// ListCollections returns all collections owned by the user.
func (s *Service) ListCollections(ctx context.Context, userID int64) ([]Collection, error) {
	rows, err := s.db.QueryContext(ctx, collectionSelect+` WHERE c.user_id = ? ORDER BY c.name COLLATE NOCASE ASC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list collections: %w", err)
	}
	defer rows.Close()

	out := make([]Collection, 0)
	for rows.Next() {
		c, err := scanCollection(rows)
		if err != nil {
			return nil, fmt.Errorf("scan collection: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetCollection returns a single owned collection.
func (s *Service) GetCollection(ctx context.Context, userID, id int64) (*Collection, error) {
	row := s.db.QueryRowContext(ctx, collectionSelect+` WHERE c.id = ? AND c.user_id = ?`, id, userID)
	c, err := scanCollection(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get collection: %w", err)
	}
	return &c, nil
}

// CollectionInput carries the editable fields of a collection.
type CollectionInput struct {
	Name          string
	Description   string
	Currency      string
	LocationLat   *float64
	LocationLng   *float64
	LocationLabel string
	CustomFields  []CustomField
}

func normalizeCollection(in *CollectionInput) error {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return errors.New("name is required")
	}
	in.Description = strings.TrimSpace(in.Description)
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	if in.Currency == "" {
		in.Currency = "USD"
	}
	if len(in.Currency) > 8 {
		return errors.New("currency code is too long")
	}
	in.LocationLabel = strings.TrimSpace(in.LocationLabel)
	in.CustomFields = normalizeCustomFields(in.CustomFields)
	return nil
}

// CreateCollection inserts a new collection owned by the user.
func (s *Service) CreateCollection(ctx context.Context, userID int64, in CollectionInput) (*Collection, error) {
	if err := normalizeCollection(&in); err != nil {
		return nil, err
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO collections (user_id, name, description, currency, location_lat, location_lng, location_label, custom_fields, created_by, updated_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, in.Name, in.Description, in.Currency, in.LocationLat, in.LocationLng, in.LocationLabel, marshalJSONField(in.CustomFields), userID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("create collection: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetCollection(ctx, userID, id)
}

// UpdateCollection updates an owned collection.
func (s *Service) UpdateCollection(ctx context.Context, userID, id int64, in CollectionInput) (*Collection, error) {
	if err := normalizeCollection(&in); err != nil {
		return nil, err
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE collections SET name = ?, description = ?, currency = ?, location_lat = ?, location_lng = ?,
		 location_label = ?, custom_fields = ?, updated_at = datetime('now'), updated_by = ?
		 WHERE id = ? AND user_id = ?`,
		in.Name, in.Description, in.Currency, in.LocationLat, in.LocationLng, in.LocationLabel,
		marshalJSONField(in.CustomFields), userID, id, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("update collection: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNotFound
	}
	return s.GetCollection(ctx, userID, id)
}

// DeleteCollection removes an owned collection and cascades to items/entries.
func (s *Service) DeleteCollection(ctx context.Context, userID, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM collections WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("delete collection: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ---------- Items ----------

const itemSelect = `
SELECT i.id, i.collection_id, i.name, i.description, i.image_path,
       i.location_lat, i.location_lng, i.location_label, i.attachments, i.custom_fields,
       i.created_at, i.updated_at,
       COALESCE(cu.display_name, cu.username, ''),
       COALESCE(uu.display_name, uu.username, ''),
       (SELECT COUNT(*) FROM entries e WHERE e.item_id = i.id)
FROM items i
JOIN collections c ON c.id = i.collection_id
LEFT JOIN users cu ON cu.id = i.created_by
LEFT JOIN users uu ON uu.id = i.updated_by
`

func scanItem(s interface{ Scan(...any) error }) (Item, error) {
	var it Item
	var attachments, customFields string
	if err := s.Scan(&it.ID, &it.CollectionID, &it.Name, &it.Description, &it.ImagePath,
		&it.LocationLat, &it.LocationLng, &it.LocationLabel, &attachments, &customFields,
		&it.CreatedAt, &it.UpdatedAt, &it.CreatedBy, &it.UpdatedBy, &it.EntryCount); err != nil {
		return Item{}, err
	}
	it.Attachments = unmarshalAttachments(attachments)
	it.CustomFields = unmarshalCustomFields(customFields)
	return it, nil
}

// ListItems returns items in an owned collection.
func (s *Service) ListItems(ctx context.Context, userID, collectionID int64) ([]Item, error) {
	if _, err := s.GetCollection(ctx, userID, collectionID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx,
		itemSelect+` WHERE i.collection_id = ? AND c.user_id = ? ORDER BY i.name COLLATE NOCASE ASC`,
		collectionID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}
	defer rows.Close()

	out := make([]Item, 0)
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, fmt.Errorf("scan item: %w", err)
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// GetItem returns a single owned item.
func (s *Service) GetItem(ctx context.Context, userID, id int64) (*Item, error) {
	row := s.db.QueryRowContext(ctx, itemSelect+` WHERE i.id = ? AND c.user_id = ?`, id, userID)
	it, err := scanItem(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get item: %w", err)
	}
	return &it, nil
}

// ItemInput carries the editable fields of an item.
type ItemInput struct {
	Name          string
	Description   string
	LocationLat   *float64
	LocationLng   *float64
	LocationLabel string
	Attachments   []Attachment
	CustomFields  []CustomField
}

// CreateItem inserts an item into an owned collection.
func (s *Service) CreateItem(ctx context.Context, userID, collectionID int64, in ItemInput) (*Item, error) {
	if _, err := s.GetCollection(ctx, userID, collectionID); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, errors.New("name is required")
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO items (collection_id, name, description, location_lat, location_lng, location_label, attachments, custom_fields, created_by, updated_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		collectionID, name, strings.TrimSpace(in.Description), in.LocationLat, in.LocationLng, strings.TrimSpace(in.LocationLabel),
		marshalJSONField(normalizeAttachments(in.Attachments)), marshalJSONField(normalizeCustomFields(in.CustomFields)), userID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("create item: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetItem(ctx, userID, id)
}

// UpdateItem updates an owned item's editable fields.
func (s *Service) UpdateItem(ctx context.Context, userID, id int64, in ItemInput) (*Item, error) {
	if _, err := s.GetItem(ctx, userID, id); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, errors.New("name is required")
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE items SET name = ?, description = ?, location_lat = ?, location_lng = ?, location_label = ?,
		 attachments = ?, custom_fields = ?, updated_at = datetime('now'), updated_by = ? WHERE id = ?`,
		name, strings.TrimSpace(in.Description), in.LocationLat, in.LocationLng, strings.TrimSpace(in.LocationLabel),
		marshalJSONField(normalizeAttachments(in.Attachments)), marshalJSONField(normalizeCustomFields(in.CustomFields)), userID, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update item: %w", err)
	}
	return s.GetItem(ctx, userID, id)
}

// AddItemAttachment appends an uploaded file to an owned item and returns the
// refreshed record.
func (s *Service) AddItemAttachment(ctx context.Context, userID, id int64, att Attachment) (*Item, error) {
	item, err := s.GetItem(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	next := append(item.Attachments, att)
	_, err = s.db.ExecContext(ctx,
		`UPDATE items SET attachments = ?, updated_at = datetime('now'), updated_by = ? WHERE id = ?`,
		marshalJSONField(normalizeAttachments(next)), userID, id,
	)
	if err != nil {
		return nil, fmt.Errorf("add item attachment: %w", err)
	}
	return s.GetItem(ctx, userID, id)
}

// SetItemImage stores the relative image path for an owned item and returns the
// previous path (so callers can clean up the old file).
func (s *Service) SetItemImage(ctx context.Context, userID, id int64, imagePath string) (string, error) {
	current, err := s.GetItem(ctx, userID, id)
	if err != nil {
		return "", err
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE items SET image_path = ?, updated_at = datetime('now'), updated_by = ? WHERE id = ?`,
		imagePath, userID, id,
	)
	if err != nil {
		return "", fmt.Errorf("set item image: %w", err)
	}
	return current.ImagePath, nil
}

// DeleteItem removes an owned item and cascades to its entries.
func (s *Service) DeleteItem(ctx context.Context, userID, id int64) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM items WHERE id = ? AND collection_id IN (SELECT id FROM collections WHERE user_id = ?)`,
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("delete item: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ---------- Entries ----------

const entrySelect = `
SELECT e.id, e.item_id, e.name, e.amount, e.currency, e.note, e.occurred_on, e.attachments,
       e.created_at, e.updated_at,
       COALESCE(cu.display_name, cu.username, ''),
       COALESCE(uu.display_name, uu.username, '')
FROM entries e
JOIN items i ON i.id = e.item_id
JOIN collections c ON c.id = i.collection_id
LEFT JOIN users cu ON cu.id = e.created_by
LEFT JOIN users uu ON uu.id = e.updated_by
`

func scanEntry(s interface{ Scan(...any) error }) (Entry, error) {
	var e Entry
	var attachments string
	if err := s.Scan(&e.ID, &e.ItemID, &e.Name, &e.Amount, &e.Currency, &e.Note, &e.OccurredOn, &attachments,
		&e.CreatedAt, &e.UpdatedAt, &e.CreatedBy, &e.UpdatedBy); err != nil {
		return Entry{}, err
	}
	e.Attachments = unmarshalAttachments(attachments)
	return e, nil
}

// collectionCurrencyForItem returns the currency of the collection that owns an
// item, scoped to the requesting user.
func (s *Service) collectionCurrencyForItem(ctx context.Context, userID, itemID int64) (string, error) {
	var currency string
	err := s.db.QueryRowContext(ctx,
		`SELECT c.currency FROM items i JOIN collections c ON c.id = i.collection_id
		 WHERE i.id = ? AND c.user_id = ?`,
		itemID, userID,
	).Scan(&currency)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("resolve collection currency: %w", err)
	}
	if strings.TrimSpace(currency) == "" {
		currency = "USD"
	}
	return currency, nil
}

// ListEntries returns entries for an owned item, newest first.
func (s *Service) ListEntries(ctx context.Context, userID, itemID int64) ([]Entry, error) {
	if _, err := s.GetItem(ctx, userID, itemID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx,
		entrySelect+` WHERE e.item_id = ? AND c.user_id = ? ORDER BY e.occurred_on DESC, e.id DESC`,
		itemID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list entries: %w", err)
	}
	defer rows.Close()

	out := make([]Entry, 0)
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// EntryInput carries the editable fields of an entry. The currency is not part
// of the input: it is inherited from the owning collection.
type EntryInput struct {
	Name        string
	Amount      float64
	Note        string
	OccurredOn  string
	Attachments []Attachment
}

func normalizeEntry(in *EntryInput) error {
	in.Name = strings.TrimSpace(in.Name)
	in.OccurredOn = strings.TrimSpace(in.OccurredOn)
	if in.OccurredOn == "" {
		in.OccurredOn = time.Now().UTC().Format("2006-01-02")
	}
	in.Note = strings.TrimSpace(in.Note)
	in.Attachments = normalizeAttachments(in.Attachments)
	return nil
}

// GetEntry returns a single owned entry.
func (s *Service) GetEntry(ctx context.Context, userID, id int64) (*Entry, error) {
	row := s.db.QueryRowContext(ctx, entrySelect+` WHERE e.id = ? AND c.user_id = ?`, id, userID)
	e, err := scanEntry(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get entry: %w", err)
	}
	return &e, nil
}

// CreateEntry records a new transaction entry against an owned item. The
// currency is inherited from the owning collection.
func (s *Service) CreateEntry(ctx context.Context, userID, itemID int64, in EntryInput) (*Entry, error) {
	if _, err := s.GetItem(ctx, userID, itemID); err != nil {
		return nil, err
	}
	if err := normalizeEntry(&in); err != nil {
		return nil, err
	}
	currency, err := s.collectionCurrencyForItem(ctx, userID, itemID)
	if err != nil {
		return nil, err
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO entries (item_id, name, amount, currency, note, occurred_on, attachments, created_by, updated_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		itemID, in.Name, in.Amount, currency, in.Note, in.OccurredOn, marshalJSONField(in.Attachments), userID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("create entry: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetEntry(ctx, userID, id)
}

// UpdateEntry updates an owned entry. The currency is re-synced from the owning
// collection so it always reflects the collection's configured currency.
func (s *Service) UpdateEntry(ctx context.Context, userID, id int64, in EntryInput) (*Entry, error) {
	current, err := s.GetEntry(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if err := normalizeEntry(&in); err != nil {
		return nil, err
	}
	currency, err := s.collectionCurrencyForItem(ctx, userID, current.ItemID)
	if err != nil {
		return nil, err
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE entries SET name = ?, amount = ?, currency = ?, note = ?, occurred_on = ?, attachments = ?,
		 updated_at = datetime('now'), updated_by = ? WHERE id = ?`,
		in.Name, in.Amount, currency, in.Note, in.OccurredOn, marshalJSONField(in.Attachments), userID, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update entry: %w", err)
	}
	return s.GetEntry(ctx, userID, id)
}

// AddEntryAttachment appends an uploaded file to an owned entry.
func (s *Service) AddEntryAttachment(ctx context.Context, userID, id int64, att Attachment) (*Entry, error) {
	entry, err := s.GetEntry(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	next := append(entry.Attachments, att)
	_, err = s.db.ExecContext(ctx,
		`UPDATE entries SET attachments = ?, updated_at = datetime('now'), updated_by = ? WHERE id = ?`,
		marshalJSONField(normalizeAttachments(next)), userID, id,
	)
	if err != nil {
		return nil, fmt.Errorf("add entry attachment: %w", err)
	}
	return s.GetEntry(ctx, userID, id)
}

// DeleteEntry removes an owned entry.
func (s *Service) DeleteEntry(ctx context.Context, userID, id int64) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM entries WHERE id = ? AND item_id IN (
			SELECT i.id FROM items i JOIN collections c ON c.id = i.collection_id WHERE c.user_id = ?
		)`,
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("delete entry: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ---------- Statistics ----------

// CollectionStats aggregates totals across every item in an owned collection.
func (s *Service) CollectionStats(ctx context.Context, userID, collectionID int64) (*Stats, error) {
	if _, err := s.GetCollection(ctx, userID, collectionID); err != nil {
		return nil, err
	}
	return s.aggregate(ctx,
		`SELECT e.currency, SUM(e.amount), COUNT(*)
		 FROM entries e JOIN items i ON i.id = e.item_id
		 WHERE i.collection_id = ?
		 GROUP BY e.currency ORDER BY e.currency`,
		collectionID,
		`SELECT COUNT(*) FROM items WHERE collection_id = ?`,
		`SELECT COUNT(*) FROM entries e JOIN items i ON i.id = e.item_id WHERE i.collection_id = ?`,
	)
}

// ItemStats aggregates totals across entries of an owned item.
func (s *Service) ItemStats(ctx context.Context, userID, itemID int64) (*Stats, error) {
	if _, err := s.GetItem(ctx, userID, itemID); err != nil {
		return nil, err
	}
	stats, err := s.aggregate(ctx,
		`SELECT currency, SUM(amount), COUNT(*) FROM entries WHERE item_id = ? GROUP BY currency ORDER BY currency`,
		itemID,
		``,
		`SELECT COUNT(*) FROM entries WHERE item_id = ?`,
	)
	if err != nil {
		return nil, err
	}
	stats.ItemCount = 1
	return stats, nil
}

// aggregate runs the shared totals/count queries. countItemsSQL may be empty.
func (s *Service) aggregate(ctx context.Context, totalsSQL string, scopeID int64, countItemsSQL, countEntriesSQL string) (*Stats, error) {
	stats := &Stats{Totals: make([]CurrencyTotal, 0)}

	rows, err := s.db.QueryContext(ctx, totalsSQL, scopeID)
	if err != nil {
		return nil, fmt.Errorf("aggregate totals: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var ct CurrencyTotal
		if err := rows.Scan(&ct.Currency, &ct.Total, &ct.Entries); err != nil {
			return nil, fmt.Errorf("scan total: %w", err)
		}
		stats.Totals = append(stats.Totals, ct)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if countItemsSQL != "" {
		if err := s.db.QueryRowContext(ctx, countItemsSQL, scopeID).Scan(&stats.ItemCount); err != nil {
			return nil, fmt.Errorf("count items: %w", err)
		}
	}
	if err := s.db.QueryRowContext(ctx, countEntriesSQL, scopeID).Scan(&stats.EntryCount); err != nil {
		return nil, fmt.Errorf("count entries: %w", err)
	}
	return stats, nil
}

// PortfolioSummary aggregates totals across everything the user owns.
type PortfolioSummary struct {
	CollectionCount int             `json:"collectionCount"`
	ItemCount       int             `json:"itemCount"`
	EntryCount      int             `json:"entryCount"`
	Totals          []CurrencyTotal `json:"totals"`
}

// PortfolioStats aggregates totals across every collection owned by the user.
func (s *Service) PortfolioStats(ctx context.Context, userID int64) (*PortfolioSummary, error) {
	summary := &PortfolioSummary{Totals: make([]CurrencyTotal, 0)}

	rows, err := s.db.QueryContext(ctx,
		`SELECT e.currency, SUM(e.amount), COUNT(*)
		 FROM entries e
		 JOIN items i ON i.id = e.item_id
		 JOIN collections c ON c.id = i.collection_id
		 WHERE c.user_id = ?
		 GROUP BY e.currency ORDER BY e.currency`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("portfolio totals: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var ct CurrencyTotal
		if err := rows.Scan(&ct.Currency, &ct.Total, &ct.Entries); err != nil {
			return nil, fmt.Errorf("scan portfolio total: %w", err)
		}
		summary.Totals = append(summary.Totals, ct)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM collections WHERE user_id = ?`, userID).Scan(&summary.CollectionCount); err != nil {
		return nil, fmt.Errorf("count collections: %w", err)
	}
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM items i JOIN collections c ON c.id = i.collection_id WHERE c.user_id = ?`,
		userID).Scan(&summary.ItemCount); err != nil {
		return nil, fmt.Errorf("count items: %w", err)
	}
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM entries e JOIN items i ON i.id = e.item_id JOIN collections c ON c.id = i.collection_id WHERE c.user_id = ?`,
		userID).Scan(&summary.EntryCount); err != nil {
		return nil, fmt.Errorf("count entries: %w", err)
	}
	return summary, nil
}

// ---------- Search ----------

// SearchResult is a flattened hit across collections and items.
type SearchResult struct {
	Type           string `json:"type"` // "collection" | "item"
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	CollectionID   int64  `json:"collectionId"`
	CollectionName string `json:"collectionName"`
}

// Search finds collections and items by name or description for the user.
func (s *Service) Search(ctx context.Context, userID int64, query string) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []SearchResult{}, nil
	}
	like := "%" + query + "%"
	results := make([]SearchResult, 0)

	colRows, err := s.db.QueryContext(ctx,
		`SELECT id, name, description FROM collections
		 WHERE user_id = ? AND (name LIKE ? OR description LIKE ?)
		 ORDER BY name COLLATE NOCASE ASC LIMIT 25`,
		userID, like, like,
	)
	if err != nil {
		return nil, fmt.Errorf("search collections: %w", err)
	}
	defer colRows.Close()
	for colRows.Next() {
		var r SearchResult
		r.Type = "collection"
		if err := colRows.Scan(&r.ID, &r.Name, &r.Description); err != nil {
			return nil, fmt.Errorf("scan collection hit: %w", err)
		}
		r.CollectionID = r.ID
		r.CollectionName = r.Name
		results = append(results, r)
	}
	if err := colRows.Err(); err != nil {
		return nil, err
	}

	itemRows, err := s.db.QueryContext(ctx,
		`SELECT i.id, i.name, i.description, c.id, c.name
		 FROM items i JOIN collections c ON c.id = i.collection_id
		 WHERE c.user_id = ? AND (i.name LIKE ? OR i.description LIKE ?)
		 ORDER BY i.name COLLATE NOCASE ASC LIMIT 50`,
		userID, like, like,
	)
	if err != nil {
		return nil, fmt.Errorf("search items: %w", err)
	}
	defer itemRows.Close()
	for itemRows.Next() {
		var r SearchResult
		r.Type = "item"
		if err := itemRows.Scan(&r.ID, &r.Name, &r.Description, &r.CollectionID, &r.CollectionName); err != nil {
			return nil, fmt.Errorf("scan item hit: %w", err)
		}
		results = append(results, r)
	}
	return results, itemRows.Err()
}
