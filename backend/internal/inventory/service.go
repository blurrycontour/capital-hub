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

// ErrForbidden is returned when a user can see a collection (shared with them)
// but lacks the permission level required for the requested action.
var ErrForbidden = errors.New("forbidden")

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

// unmarshalStringList decodes a JSON array of strings, returning an empty slice
// on any error so callers never get nil.
func unmarshalStringList(s string) []string {
	out := []string{}
	if strings.TrimSpace(s) == "" {
		return out
	}
	_ = json.Unmarshal([]byte(s), &out)
	if out == nil {
		return []string{}
	}
	return out
}

// normalizeStringList trims entries and drops empties.
func normalizeStringList(v []string) []string {
	out := make([]string, 0, len(v))
	for _, s := range v {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
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
	// Sharing metadata, relative to the requesting user.
	OwnerName   string `json:"ownerName"`
	Shared      bool   `json:"shared"`      // true when not owned by the requester
	AccessLevel string `json:"accessLevel"` // "owner" | "write" | "read"
	ShareCount  int    `json:"shareCount"`  // number of users this collection is shared with
}

// Item is a single asset within a collection.
type Item struct {
	ID            int64         `json:"id"`
	CollectionID  int64         `json:"collectionId"`
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	ImagePath     string        `json:"imagePath"`
	Images        []string      `json:"images"`
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
	Kind        string       `json:"kind"`
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
	Debit    float64 `json:"debit"`
	Credit   float64 `json:"credit"`
	Net      float64 `json:"net"`
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
       c.user_id,
       COALESCE(ou.display_name, ou.username, ''),
       ms.access,
       (SELECT COUNT(*) FROM items i WHERE i.collection_id = c.id),
       (SELECT COUNT(*) FROM collection_shares cs WHERE cs.collection_id = c.id)
FROM collections c
LEFT JOIN users cu ON cu.id = c.created_by
LEFT JOIN users uu ON uu.id = c.updated_by
LEFT JOIN users ou ON ou.id = c.user_id
LEFT JOIN collection_shares ms ON ms.collection_id = c.id AND ms.user_id = ?
`

func scanCollection(s interface{ Scan(...any) error }, userID int64) (Collection, error) {
	var c Collection
	var customFields string
	var ownerID int64
	var ownerName string
	var myAccess sql.NullString
	if err := s.Scan(&c.ID, &c.Name, &c.Description, &c.Currency,
		&c.LocationLat, &c.LocationLng, &c.LocationLabel, &customFields,
		&c.CreatedAt, &c.UpdatedAt, &c.CreatedBy, &c.UpdatedBy,
		&ownerID, &ownerName, &myAccess, &c.ItemCount, &c.ShareCount); err != nil {
		return Collection{}, err
	}
	c.CustomFields = unmarshalCustomFields(customFields)
	c.OwnerName = ownerName
	if ownerID == userID {
		c.AccessLevel = "owner"
		c.Shared = false
	} else {
		c.Shared = true
		if myAccess.Valid && myAccess.String == "write" {
			c.AccessLevel = "write"
		} else {
			c.AccessLevel = "read"
		}
	}
	return c, nil
}

// collectionAccessLevel returns "owner", "write" or "read" for the user's
// access to a collection, or ErrNotFound when they have no access at all.
func (s *Service) collectionAccessLevel(ctx context.Context, userID, collectionID int64) (string, error) {
	var ownerID int64
	err := s.db.QueryRowContext(ctx, `SELECT user_id FROM collections WHERE id = ?`, collectionID).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("lookup collection owner: %w", err)
	}
	if ownerID == userID {
		return "owner", nil
	}
	var access string
	err = s.db.QueryRowContext(ctx,
		`SELECT access FROM collection_shares WHERE collection_id = ? AND user_id = ?`,
		collectionID, userID,
	).Scan(&access)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("lookup collection share: %w", err)
	}
	if access == "write" {
		return "write", nil
	}
	return "read", nil
}

// requireCollectionWrite ensures the user can modify a collection's contents.
// Returns ErrNotFound when there is no access and ErrForbidden for read-only.
func (s *Service) requireCollectionWrite(ctx context.Context, userID, collectionID int64) error {
	lvl, err := s.collectionAccessLevel(ctx, userID, collectionID)
	if err != nil {
		return err
	}
	if lvl == "read" {
		return ErrForbidden
	}
	return nil
}

// itemCollectionID resolves the owning collection of an item.
func (s *Service) itemCollectionID(ctx context.Context, itemID int64) (int64, error) {
	var cid int64
	err := s.db.QueryRowContext(ctx, `SELECT collection_id FROM items WHERE id = ?`, itemID).Scan(&cid)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("lookup item collection: %w", err)
	}
	return cid, nil
}

// requireItemWrite ensures the user can modify an item (and its entries).
func (s *Service) requireItemWrite(ctx context.Context, userID, itemID int64) error {
	cid, err := s.itemCollectionID(ctx, itemID)
	if err != nil {
		return err
	}
	return s.requireCollectionWrite(ctx, userID, cid)
}

// ListCollections returns all collections owned by or shared with the user.
func (s *Service) ListCollections(ctx context.Context, userID int64) ([]Collection, error) {
	rows, err := s.db.QueryContext(ctx,
		collectionSelect+` WHERE c.user_id = ? OR ms.access IS NOT NULL ORDER BY c.name COLLATE NOCASE ASC`,
		userID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list collections: %w", err)
	}
	defer rows.Close()

	out := make([]Collection, 0)
	for rows.Next() {
		c, err := scanCollection(rows, userID)
		if err != nil {
			return nil, fmt.Errorf("scan collection: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetCollection returns a collection the user owns or has been shared.
func (s *Service) GetCollection(ctx context.Context, userID, id int64) (*Collection, error) {
	row := s.db.QueryRowContext(ctx,
		collectionSelect+` WHERE c.id = ? AND (c.user_id = ? OR ms.access IS NOT NULL)`,
		userID, id, userID,
	)
	c, err := scanCollection(row, userID)
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
		in.Currency = "EUR"
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

// ---------- Sharing ----------

// CollectionShare describes a user a collection has been shared with.
type CollectionShare struct {
	UserID      int64  `json:"userId"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Access      string `json:"access"` // "read" | "write"
}

// requireCollectionOwner returns ErrNotFound unless the user owns the collection.
func (s *Service) requireCollectionOwner(ctx context.Context, userID, collectionID int64) error {
	lvl, err := s.collectionAccessLevel(ctx, userID, collectionID)
	if err != nil {
		return err
	}
	if lvl != "owner" {
		return ErrForbidden
	}
	return nil
}

// ListShares returns the users a collection (owned by userID) is shared with.
func (s *Service) ListShares(ctx context.Context, userID, collectionID int64) ([]CollectionShare, error) {
	if err := s.requireCollectionOwner(ctx, userID, collectionID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT u.id, u.username, u.email, COALESCE(u.display_name, ''), cs.access
		 FROM collection_shares cs
		 JOIN users u ON u.id = cs.user_id
		 WHERE cs.collection_id = ?
		 ORDER BY u.username COLLATE NOCASE ASC`,
		collectionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list shares: %w", err)
	}
	defer rows.Close()

	out := make([]CollectionShare, 0)
	for rows.Next() {
		var sh CollectionShare
		if err := rows.Scan(&sh.UserID, &sh.Username, &sh.Email, &sh.DisplayName, &sh.Access); err != nil {
			return nil, fmt.Errorf("scan share: %w", err)
		}
		out = append(out, sh)
	}
	return out, rows.Err()
}

// CollectionName returns the name of a collection without any permission check.
// It is used internally for building notification messages.
func (s *Service) CollectionName(ctx context.Context, collectionID int64) (string, error) {
	var name string
	err := s.db.QueryRowContext(ctx, `SELECT name FROM collections WHERE id = ?`, collectionID).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("lookup collection name: %w", err)
	}
	return name, nil
}

// CollectionAccessorIDs returns the owner's user ID together with the IDs of
// all users the collection is shared with. Used to fan-out notifications.
func (s *Service) CollectionAccessorIDs(ctx context.Context, collectionID int64) ([]int64, error) {
	var ownerID int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT created_by FROM collections WHERE id = ?`, collectionID,
	).Scan(&ownerID); err != nil {
		return nil, fmt.Errorf("lookup collection owner: %w", err)
	}
	ids := []int64{ownerID}

	rows, err := s.db.QueryContext(ctx,
		`SELECT user_id FROM collection_shares WHERE collection_id = ?`, collectionID,
	)
	if err != nil {
		return nil, fmt.Errorf("query shares: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var uid int64
		if err := rows.Scan(&uid); err != nil {
			return nil, fmt.Errorf("scan share: %w", err)
		}
		ids = append(ids, uid)
	}
	return ids, rows.Err()
}

// ShareCollection grants another user (by username or email) read or write
// access to a collection owned by userID.
func (s *Service) ShareCollection(ctx context.Context, userID, collectionID int64, identifier, access string) (*CollectionShare, error) {
	if err := s.requireCollectionOwner(ctx, userID, collectionID); err != nil {
		return nil, err
	}
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return nil, errors.New("username or email is required")
	}
	access = strings.ToLower(strings.TrimSpace(access))
	if access != "read" && access != "write" {
		return nil, errors.New("access must be 'read' or 'write'")
	}

	var target CollectionShare
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, email, COALESCE(display_name, '') FROM users
		 WHERE username = ? OR email = ? LIMIT 1`,
		identifier, strings.ToLower(identifier),
	).Scan(&target.UserID, &target.Username, &target.Email, &target.DisplayName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("no user found with that username or email")
	}
	if err != nil {
		return nil, fmt.Errorf("lookup share target: %w", err)
	}
	if target.UserID == userID {
		return nil, errors.New("you already own this collection")
	}

	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO collection_shares (collection_id, user_id, access)
		 VALUES (?, ?, ?)
		 ON CONFLICT(collection_id, user_id) DO UPDATE SET access = excluded.access`,
		collectionID, target.UserID, access,
	); err != nil {
		return nil, fmt.Errorf("upsert share: %w", err)
	}
	target.Access = access
	return &target, nil
}

// UnshareCollection revokes a user's access to a collection owned by userID.
func (s *Service) UnshareCollection(ctx context.Context, userID, collectionID, targetUserID int64) error {
	if err := s.requireCollectionOwner(ctx, userID, collectionID); err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM collection_shares WHERE collection_id = ? AND user_id = ?`,
		collectionID, targetUserID,
	)
	if err != nil {
		return fmt.Errorf("delete share: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ---------- Items ----------

const itemSelect = `
SELECT i.id, i.collection_id, i.name, i.description, i.image_path, i.images,
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
	var images, attachments, customFields string
	if err := s.Scan(&it.ID, &it.CollectionID, &it.Name, &it.Description, &it.ImagePath, &images,
		&it.LocationLat, &it.LocationLng, &it.LocationLabel, &attachments, &customFields,
		&it.CreatedAt, &it.UpdatedAt, &it.CreatedBy, &it.UpdatedBy, &it.EntryCount); err != nil {
		return Item{}, err
	}
	it.Images = unmarshalStringList(images)
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
		itemSelect+` WHERE i.collection_id = ? ORDER BY i.name COLLATE NOCASE ASC`,
		collectionID,
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

// ItemWithCollection is an item paired with the name of its owning collection.
// Used for the cross-collection items listing.
type ItemWithCollection struct {
	Item
	CollectionName string `json:"collectionName"`
}

const itemWithCollectionSelect = `
SELECT i.id, i.collection_id, i.name, i.description, i.image_path, i.images,
       i.location_lat, i.location_lng, i.location_label, i.attachments, i.custom_fields,
       i.created_at, i.updated_at,
       COALESCE(cu.display_name, cu.username, ''),
       COALESCE(uu.display_name, uu.username, ''),
       (SELECT COUNT(*) FROM entries e WHERE e.item_id = i.id),
       c.name
FROM items i
JOIN collections c ON c.id = i.collection_id
LEFT JOIN users cu ON cu.id = i.created_by
LEFT JOIN users uu ON uu.id = i.updated_by
`

// ListAllItems returns every item across all collections the user can access.
// When includeShared is true, items in collections shared with the user are
// included as well.
func (s *Service) ListAllItems(ctx context.Context, userID int64, includeShared bool) ([]ItemWithCollection, error) {
	scope := "c.user_id = ?"
	args := []any{userID}
	if includeShared {
		scope = "(c.user_id = ? OR EXISTS (SELECT 1 FROM collection_shares cs WHERE cs.collection_id = c.id AND cs.user_id = ?))"
		args = []any{userID, userID}
	}
	rows, err := s.db.QueryContext(ctx,
		itemWithCollectionSelect+` WHERE `+scope+` ORDER BY i.name COLLATE NOCASE ASC`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("list all items: %w", err)
	}
	defer rows.Close()

	out := make([]ItemWithCollection, 0)
	for rows.Next() {
		var iw ItemWithCollection
		var images, attachments, customFields string
		if err := rows.Scan(&iw.ID, &iw.CollectionID, &iw.Name, &iw.Description, &iw.ImagePath, &images,
			&iw.LocationLat, &iw.LocationLng, &iw.LocationLabel, &attachments, &customFields,
			&iw.CreatedAt, &iw.UpdatedAt, &iw.CreatedBy, &iw.UpdatedBy, &iw.EntryCount, &iw.CollectionName); err != nil {
			return nil, fmt.Errorf("scan item: %w", err)
		}
		iw.Images = unmarshalStringList(images)
		iw.Attachments = unmarshalAttachments(attachments)
		iw.CustomFields = unmarshalCustomFields(customFields)
		out = append(out, iw)
	}
	return out, rows.Err()
}

// ItemCollectionAndName returns the owning collection ID and the name of an
// item without any permission check. Used for building notifications.
func (s *Service) ItemCollectionAndName(ctx context.Context, itemID int64) (int64, string, error) {
	var collID int64
	var name string
	err := s.db.QueryRowContext(ctx,
		`SELECT collection_id, name FROM items WHERE id = ?`, itemID,
	).Scan(&collID, &name)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", ErrNotFound
	}
	if err != nil {
		return 0, "", fmt.Errorf("lookup item: %w", err)
	}
	return collID, name, nil
}

// GetItem returns a single item the user owns or has been shared.
func (s *Service) GetItem(ctx context.Context, userID, id int64) (*Item, error) {
	row := s.db.QueryRowContext(ctx, itemSelect+` WHERE i.id = ?`, id)
	it, err := scanItem(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get item: %w", err)
	}
	if _, err := s.collectionAccessLevel(ctx, userID, it.CollectionID); err != nil {
		return nil, err
	}
	return &it, nil
}

// ItemInput carries the editable fields of an item.
type ItemInput struct {
	Name          string
	Description   string
	Images        []string
	LocationLat   *float64
	LocationLng   *float64
	LocationLabel string
	Attachments   []Attachment
	CustomFields  []CustomField
}

// CreateItem inserts an item into a collection the user can write to.
func (s *Service) CreateItem(ctx context.Context, userID, collectionID int64, in ItemInput) (*Item, error) {
	if err := s.requireCollectionWrite(ctx, userID, collectionID); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, errors.New("name is required")
	}
	images := normalizeStringList(in.Images)
	imagePath := ""
	if len(images) > 0 {
		imagePath = images[0]
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO items (collection_id, name, description, image_path, images, location_lat, location_lng, location_label, attachments, custom_fields, created_by, updated_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		collectionID, name, strings.TrimSpace(in.Description), imagePath, marshalJSONField(images), in.LocationLat, in.LocationLng, strings.TrimSpace(in.LocationLabel),
		marshalJSONField(normalizeAttachments(in.Attachments)), marshalJSONField(normalizeCustomFields(in.CustomFields)), userID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("create item: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetItem(ctx, userID, id)
}

// UpdateItem updates an item's editable fields (requires write access).
func (s *Service) UpdateItem(ctx context.Context, userID, id int64, in ItemInput) (*Item, error) {
	if err := s.requireItemWrite(ctx, userID, id); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, errors.New("name is required")
	}
	images := normalizeStringList(in.Images)
	imagePath := ""
	if len(images) > 0 {
		imagePath = images[0]
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE items SET name = ?, description = ?, image_path = ?, images = ?, location_lat = ?, location_lng = ?, location_label = ?,
		 attachments = ?, custom_fields = ?, updated_at = datetime('now'), updated_by = ? WHERE id = ?`,
		name, strings.TrimSpace(in.Description), imagePath, marshalJSONField(images), in.LocationLat, in.LocationLng, strings.TrimSpace(in.LocationLabel),
		marshalJSONField(normalizeAttachments(in.Attachments)), marshalJSONField(normalizeCustomFields(in.CustomFields)), userID, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update item: %w", err)
	}
	return s.GetItem(ctx, userID, id)
}

// MoveItem moves an item to a different collection the user can write to. When
// the target collection uses a different currency, the item's entries are
// re-stamped with the new currency so totals stay consistent.
func (s *Service) MoveItem(ctx context.Context, userID, id, targetCollectionID int64) (*Item, error) {
	if err := s.requireItemWrite(ctx, userID, id); err != nil {
		return nil, err
	}
	if err := s.requireCollectionWrite(ctx, userID, targetCollectionID); err != nil {
		return nil, err
	}
	var currency string
	err := s.db.QueryRowContext(ctx, `SELECT currency FROM collections WHERE id = ?`, targetCollectionID).Scan(&currency)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lookup target currency: %w", err)
	}
	if _, err := s.db.ExecContext(ctx,
		`UPDATE items SET collection_id = ?, updated_at = datetime('now'), updated_by = ? WHERE id = ?`,
		targetCollectionID, userID, id,
	); err != nil {
		return nil, fmt.Errorf("move item: %w", err)
	}
	if _, err := s.db.ExecContext(ctx,
		`UPDATE entries SET currency = ?, updated_at = datetime('now'), updated_by = ? WHERE item_id = ?`,
		currency, userID, id,
	); err != nil {
		return nil, fmt.Errorf("restamp entry currency: %w", err)
	}
	return s.GetItem(ctx, userID, id)
}

func (s *Service) AddItemAttachment(ctx context.Context, userID, id int64, att Attachment) (*Item, error) {
	if err := s.requireItemWrite(ctx, userID, id); err != nil {
		return nil, err
	}
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

// RemoveItemAttachment removes an attachment (matched by path) from an owned
// item and returns the refreshed record.
func (s *Service) RemoveItemAttachment(ctx context.Context, userID, id int64, path string) (*Item, error) {
	if err := s.requireItemWrite(ctx, userID, id); err != nil {
		return nil, err
	}
	item, err := s.GetItem(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	next := make([]Attachment, 0, len(item.Attachments))
	for _, a := range item.Attachments {
		if a.Path != path {
			next = append(next, a)
		}
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE items SET attachments = ?, updated_at = datetime('now'), updated_by = ? WHERE id = ?`,
		marshalJSONField(normalizeAttachments(next)), userID, id,
	)
	if err != nil {
		return nil, fmt.Errorf("remove item attachment: %w", err)
	}
	return s.GetItem(ctx, userID, id)
}

// AddItemImage appends an uploaded image to an owned item. The first image
// also becomes the cover (image_path) used for thumbnails.
func (s *Service) AddItemImage(ctx context.Context, userID, id int64, imagePath string) (*Item, error) {
	if err := s.requireItemWrite(ctx, userID, id); err != nil {
		return nil, err
	}
	item, err := s.GetItem(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	next := append(item.Images, imagePath)
	cover := next[0]
	_, err = s.db.ExecContext(ctx,
		`UPDATE items SET image_path = ?, images = ?, updated_at = datetime('now'), updated_by = ? WHERE id = ?`,
		cover, marshalJSONField(normalizeStringList(next)), userID, id,
	)
	if err != nil {
		return nil, fmt.Errorf("add item image: %w", err)
	}
	return s.GetItem(ctx, userID, id)
}

// RemoveItemImage removes an image from an owned item and returns the refreshed
// record. The cover is updated to the first remaining image (or cleared).
func (s *Service) RemoveItemImage(ctx context.Context, userID, id int64, imagePath string) (*Item, error) {
	if err := s.requireItemWrite(ctx, userID, id); err != nil {
		return nil, err
	}
	item, err := s.GetItem(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	next := make([]string, 0, len(item.Images))
	for _, p := range item.Images {
		if p != imagePath {
			next = append(next, p)
		}
	}
	cover := ""
	if len(next) > 0 {
		cover = next[0]
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE items SET image_path = ?, images = ?, updated_at = datetime('now'), updated_by = ? WHERE id = ?`,
		cover, marshalJSONField(next), userID, id,
	)
	if err != nil {
		return nil, fmt.Errorf("remove item image: %w", err)
	}
	return s.GetItem(ctx, userID, id)
}

// SetItemCover makes the given image the item's display picture by moving it to
// the front of the gallery, which the cover (image_path) is always derived
// from. The image must already belong to the item.
func (s *Service) SetItemCover(ctx context.Context, userID, id int64, imagePath string) (*Item, error) {
	if err := s.requireItemWrite(ctx, userID, id); err != nil {
		return nil, err
	}
	item, err := s.GetItem(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	found := false
	next := make([]string, 0, len(item.Images))
	next = append(next, imagePath)
	for _, p := range item.Images {
		if p == imagePath {
			found = true
			continue
		}
		next = append(next, p)
	}
	if !found {
		return nil, ErrNotFound
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE items SET image_path = ?, images = ?, updated_at = datetime('now'), updated_by = ? WHERE id = ?`,
		imagePath, marshalJSONField(next), userID, id,
	)
	if err != nil {
		return nil, fmt.Errorf("set item cover: %w", err)
	}
	return s.GetItem(ctx, userID, id)
}

// DeleteItem removes an item and cascades to its entries (requires write).
func (s *Service) DeleteItem(ctx context.Context, userID, id int64) error {
	if err := s.requireItemWrite(ctx, userID, id); err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM items WHERE id = ?`, id)
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
SELECT e.id, e.item_id, e.name, e.amount, e.kind, e.currency, e.note, e.occurred_on, e.attachments,
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
	if err := s.Scan(&e.ID, &e.ItemID, &e.Name, &e.Amount, &e.Kind, &e.Currency, &e.Note, &e.OccurredOn, &attachments,
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
		 WHERE i.id = ?`,
		itemID,
	).Scan(&currency)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("resolve collection currency: %w", err)
	}
	if strings.TrimSpace(currency) == "" {
		currency = "EUR"
	}
	return currency, nil
}

// ListEntries returns entries for an item the user can read, newest first.
func (s *Service) ListEntries(ctx context.Context, userID, itemID int64) ([]Entry, error) {
	if _, err := s.GetItem(ctx, userID, itemID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx,
		entrySelect+` WHERE e.item_id = ? ORDER BY e.occurred_on DESC, e.id DESC`,
		itemID,
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
	Kind        string
	Note        string
	OccurredOn  string
	Attachments []Attachment
}

func normalizeEntry(in *EntryInput) error {
	in.Name = strings.TrimSpace(in.Name)
	in.Kind = strings.ToLower(strings.TrimSpace(in.Kind))
	if in.Kind != "credit" {
		in.Kind = "debit"
	}
	in.OccurredOn = strings.TrimSpace(in.OccurredOn)
	if in.OccurredOn == "" {
		in.OccurredOn = time.Now().UTC().Format("2006-01-02")
	}
	in.Note = strings.TrimSpace(in.Note)
	in.Attachments = normalizeAttachments(in.Attachments)
	return nil
}

// GetEntry returns a single entry the user owns or has been shared.
func (s *Service) GetEntry(ctx context.Context, userID, id int64) (*Entry, error) {
	row := s.db.QueryRowContext(ctx, entrySelect+` WHERE e.id = ?`, id)
	e, err := scanEntry(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get entry: %w", err)
	}
	if _, err := s.requireEntryAccess(ctx, userID, e.ItemID); err != nil {
		return nil, err
	}
	return &e, nil
}

// requireEntryAccess verifies the user can at least read the entry's collection.
func (s *Service) requireEntryAccess(ctx context.Context, userID, itemID int64) (string, error) {
	cid, err := s.itemCollectionID(ctx, itemID)
	if err != nil {
		return "", err
	}
	return s.collectionAccessLevel(ctx, userID, cid)
}

// CreateEntry records a new transaction entry against an item the user can
// write to. The currency is inherited from the owning collection.
func (s *Service) CreateEntry(ctx context.Context, userID, itemID int64, in EntryInput) (*Entry, error) {
	if err := s.requireItemWrite(ctx, userID, itemID); err != nil {
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
		`INSERT INTO entries (item_id, name, amount, kind, currency, note, occurred_on, attachments, created_by, updated_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		itemID, in.Name, in.Amount, in.Kind, currency, in.Note, in.OccurredOn, marshalJSONField(in.Attachments), userID, userID,
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
	if err := s.requireItemWrite(ctx, userID, current.ItemID); err != nil {
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
		`UPDATE entries SET name = ?, amount = ?, kind = ?, currency = ?, note = ?, occurred_on = ?, attachments = ?,
		 updated_at = datetime('now'), updated_by = ? WHERE id = ?`,
		in.Name, in.Amount, in.Kind, currency, in.Note, in.OccurredOn, marshalJSONField(in.Attachments), userID, id,
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
	if err := s.requireItemWrite(ctx, userID, entry.ItemID); err != nil {
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

// RemoveEntryAttachment removes an attachment (matched by path) from an entry
// and returns the refreshed record.
func (s *Service) RemoveEntryAttachment(ctx context.Context, userID, id int64, path string) (*Entry, error) {
	entry, err := s.GetEntry(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if err := s.requireItemWrite(ctx, userID, entry.ItemID); err != nil {
		return nil, err
	}
	next := make([]Attachment, 0, len(entry.Attachments))
	for _, a := range entry.Attachments {
		if a.Path != path {
			next = append(next, a)
		}
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE entries SET attachments = ?, updated_at = datetime('now'), updated_by = ? WHERE id = ?`,
		marshalJSONField(normalizeAttachments(next)), userID, id,
	)
	if err != nil {
		return nil, fmt.Errorf("remove entry attachment: %w", err)
	}
	return s.GetEntry(ctx, userID, id)
}

// DeleteEntry removes an entry (requires write access).
func (s *Service) DeleteEntry(ctx context.Context, userID, id int64) error {
	entry, err := s.GetEntry(ctx, userID, id)
	if err != nil {
		return err
	}
	if err := s.requireItemWrite(ctx, userID, entry.ItemID); err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM entries WHERE id = ?`, id)
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
		`SELECT e.currency,
		        SUM(CASE WHEN e.kind = 'credit' THEN e.amount ELSE 0 END),
		        SUM(CASE WHEN e.kind = 'debit' THEN e.amount ELSE 0 END),
		        COUNT(*)
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
		`SELECT currency,
		        SUM(CASE WHEN kind = 'credit' THEN amount ELSE 0 END),
		        SUM(CASE WHEN kind = 'debit' THEN amount ELSE 0 END),
		        COUNT(*)
		 FROM entries WHERE item_id = ? GROUP BY currency ORDER BY currency`,
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
		if err := rows.Scan(&ct.Currency, &ct.Credit, &ct.Debit, &ct.Entries); err != nil {
			return nil, fmt.Errorf("scan total: %w", err)
		}
		ct.Net = ct.Credit - ct.Debit
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
// When includeShared is true, collections shared with the user are counted too.
func (s *Service) PortfolioStats(ctx context.Context, userID int64, includeShared bool) (*PortfolioSummary, error) {
	summary := &PortfolioSummary{Totals: make([]CurrencyTotal, 0)}

	// scope is the WHERE predicate (against collections aliased "c") selecting the
	// collections that count toward the totals, plus its bound arguments.
	scope := "c.user_id = ?"
	args := []any{userID}
	if includeShared {
		scope = "(c.user_id = ? OR EXISTS (SELECT 1 FROM collection_shares cs WHERE cs.collection_id = c.id AND cs.user_id = ?))"
		args = []any{userID, userID}
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT e.currency,
		        SUM(CASE WHEN e.kind = 'credit' THEN e.amount ELSE 0 END),
		        SUM(CASE WHEN e.kind = 'debit' THEN e.amount ELSE 0 END),
		        COUNT(*)
		 FROM entries e
		 JOIN items i ON i.id = e.item_id
		 JOIN collections c ON c.id = i.collection_id
		 WHERE `+scope+`
		 GROUP BY e.currency ORDER BY e.currency`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("portfolio totals: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var ct CurrencyTotal
		if err := rows.Scan(&ct.Currency, &ct.Credit, &ct.Debit, &ct.Entries); err != nil {
			return nil, fmt.Errorf("scan portfolio total: %w", err)
		}
		ct.Net = ct.Credit - ct.Debit
		summary.Totals = append(summary.Totals, ct)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM collections c WHERE `+scope, args...).Scan(&summary.CollectionCount); err != nil {
		return nil, fmt.Errorf("count collections: %w", err)
	}
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM items i JOIN collections c ON c.id = i.collection_id WHERE `+scope,
		args...).Scan(&summary.ItemCount); err != nil {
		return nil, fmt.Errorf("count items: %w", err)
	}
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM entries e JOIN items i ON i.id = e.item_id JOIN collections c ON c.id = i.collection_id WHERE `+scope,
		args...).Scan(&summary.EntryCount); err != nil {
		return nil, fmt.Errorf("count entries: %w", err)
	}
	return summary, nil
}

// RecentItems returns the most recently created or updated items the user can
// access, ordered by last update. Honours the same shared-collection scope as
// the portfolio stats.
func (s *Service) RecentItems(ctx context.Context, userID int64, includeShared bool, limit int) ([]Item, error) {
	if limit <= 0 || limit > 50 {
		limit = 8
	}
	scope := "c.user_id = ?"
	args := []any{userID}
	if includeShared {
		scope = "(c.user_id = ? OR EXISTS (SELECT 1 FROM collection_shares cs WHERE cs.collection_id = c.id AND cs.user_id = ?))"
		args = []any{userID, userID}
	}
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx,
		itemSelect+` WHERE `+scope+` ORDER BY i.updated_at DESC, i.id DESC LIMIT ?`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("recent items: %w", err)
	}
	defer rows.Close()
	out := make([]Item, 0)
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, fmt.Errorf("scan recent item: %w", err)
		}
		out = append(out, it)
	}
	return out, rows.Err()
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
		`SELECT c.id, c.name, c.description FROM collections c
		 LEFT JOIN collection_shares s ON s.collection_id = c.id AND s.user_id = ?
		 WHERE (c.user_id = ? OR s.access IS NOT NULL) AND (c.name LIKE ? OR c.description LIKE ?)
		 ORDER BY c.name COLLATE NOCASE ASC LIMIT 25`,
		userID, userID, like, like,
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
		 LEFT JOIN collection_shares s ON s.collection_id = c.id AND s.user_id = ?
		 WHERE (c.user_id = ? OR s.access IS NOT NULL) AND (i.name LIKE ? OR i.description LIKE ?)
		 ORDER BY i.name COLLATE NOCASE ASC LIMIT 50`,
		userID, userID, like, like,
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
