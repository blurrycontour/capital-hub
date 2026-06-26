package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/aditya/capital-hub/internal/inventory"
	"github.com/aditya/capital-hub/internal/notify"
)

// maxUploadBytes caps item image uploads at 10 MiB.
const maxUploadBytes = 10 << 20

func (s *Server) pathID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// writeInventoryError maps domain errors to HTTP responses.
func (s *Server) writeInventoryError(w http.ResponseWriter, r *http.Request, err error, action string) {
	switch {
	case errors.Is(err, inventory.ErrNotFound):
		writeAPIError(w, http.StatusNotFound, "not found")
	case errors.Is(err, inventory.ErrForbidden):
		writeAPIError(w, http.StatusForbidden, "you do not have permission to perform this action")
	default:
		s.logger.ErrorContext(r.Context(), action+" failed", "error", err)
		writeAPIError(w, http.StatusInternalServerError, action+" failed")
	}
}

// ---------- Collections ----------

type collectionPayload struct {
	Name          string                  `json:"name"`
	Description   string                  `json:"description"`
	Currency      string                  `json:"currency"`
	LocationLat   *float64                `json:"locationLat"`
	LocationLng   *float64                `json:"locationLng"`
	LocationLabel string                  `json:"locationLabel"`
	CustomFields  []inventory.CustomField `json:"customFields"`
}

func (p collectionPayload) toInput() inventory.CollectionInput {
	return inventory.CollectionInput{
		Name:          p.Name,
		Description:   p.Description,
		Currency:      p.Currency,
		LocationLat:   p.LocationLat,
		LocationLng:   p.LocationLng,
		LocationLabel: p.LocationLabel,
		CustomFields:  p.CustomFields,
	}
}

func (s *Server) handleListCollections(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	collections, err := s.inventory.ListCollections(r.Context(), user.ID)
	if err != nil {
		s.writeInventoryError(w, r, err, "list collections")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"collections": collections})
}

func (s *Server) handleGetCollection(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	id, ok := s.pathID(r)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "invalid collection id")
		return
	}
	c, err := s.inventory.GetCollection(r.Context(), user.ID, id)
	if err != nil {
		s.writeInventoryError(w, r, err, "get collection")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"collection": c})
}

func (s *Server) handleCreateCollection(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	var req collectionPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid json payload")
		return
	}
	c, err := s.inventory.CreateCollection(r.Context(), user.ID, req.toInput())
	if err != nil {
		if isValidationErr(err) {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.writeInventoryError(w, r, err, "create collection")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"collection": c})
}

func (s *Server) handleUpdateCollection(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	id, ok := s.pathID(r)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "invalid collection id")
		return
	}
	var req collectionPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid json payload")
		return
	}
	c, err := s.inventory.UpdateCollection(r.Context(), user.ID, id, req.toInput())
	if err != nil {
		if isValidationErr(err) {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.writeInventoryError(w, r, err, "update collection")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"collection": c})
}

func (s *Server) handleDeleteCollection(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	id, ok := s.pathID(r)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "invalid collection id")
		return
	}
	if err := s.inventory.DeleteCollection(r.Context(), user.ID, id); err != nil {
		s.writeInventoryError(w, r, err, "delete collection")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleCollectionStats(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	id, ok := s.pathID(r)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "invalid collection id")
		return
	}
	stats, err := s.inventory.CollectionStats(r.Context(), user.ID, id)
	if err != nil {
		s.writeInventoryError(w, r, err, "collection stats")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stats": stats})
}

// ---------- Collection sharing ----------

type shareCollectionRequest struct {
	Identifier string `json:"identifier"`
	Access     string `json:"access"`
}

func (s *Server) handleListCollectionShares(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	id, ok := s.pathID(r)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "invalid collection id")
		return
	}
	shares, err := s.inventory.ListShares(r.Context(), user.ID, id)
	if err != nil {
		s.writeInventoryError(w, r, err, "list shares")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"shares": shares})
}

func (s *Server) handleShareCollection(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	id, ok := s.pathID(r)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "invalid collection id")
		return
	}
	var req shareCollectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid json payload")
		return
	}
	share, err := s.inventory.ShareCollection(r.Context(), user.ID, id, req.Identifier, req.Access)
	if err != nil {
		if errors.Is(err, inventory.ErrNotFound) || errors.Is(err, inventory.ErrForbidden) {
			s.writeInventoryError(w, r, err, "share collection")
			return
		}
		// Remaining errors are user-input problems (unknown user, bad access).
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"share": share})

	// Notify the recipient asynchronously (non-fatal if it fails).
	actorName := user.DisplayName
	if actorName == "" {
		actorName = user.Username
	}
	go func(collID int64, recipientID int64, actor, accessLevel string) {
		ctx := context.Background()
		colName, err := s.inventory.CollectionName(ctx, collID)
		if err != nil {
			return
		}
		accessLabel := "read only"
		if accessLevel == "write" {
			accessLabel = "can edit"
		}
		_ = s.notify.CreateInApp(ctx, notify.InAppInput{
			UserID: recipientID,
			Type:   "collection_shared",
			Title:  fmt.Sprintf("%s shared a collection with you", actor),
			Body:   fmt.Sprintf("%s shared \u201c%s\u201d with you (%s).", actor, colName, accessLabel),
			Link:   fmt.Sprintf("/collections/%d", collID),
		})
	}(id, share.UserID, actorName, share.Access)
}

func (s *Server) handleUnshareCollection(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	id, ok := s.pathID(r)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "invalid collection id")
		return
	}
	targetID, err := strconv.ParseInt(chi.URLParam(r, "userId"), 10, 64)
	if err != nil || targetID <= 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	if err := s.inventory.UnshareCollection(r.Context(), user.ID, id, targetID); err != nil {
		s.writeInventoryError(w, r, err, "unshare collection")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---------- Items ----------

type itemPayload struct {
	Name          string                  `json:"name"`
	Description   string                  `json:"description"`
	Images        []string                `json:"images"`
	LocationLat   *float64                `json:"locationLat"`
	LocationLng   *float64                `json:"locationLng"`
	LocationLabel string                  `json:"locationLabel"`
	Attachments   []inventory.Attachment  `json:"attachments"`
	CustomFields  []inventory.CustomField `json:"customFields"`
}

func (p itemPayload) toInput() inventory.ItemInput {
	return inventory.ItemInput{
		Name:          p.Name,
		Description:   p.Description,
		Images:        p.Images,
		LocationLat:   p.LocationLat,
		LocationLng:   p.LocationLng,
		LocationLabel: p.LocationLabel,
		Attachments:   p.Attachments,
		CustomFields:  p.CustomFields,
	}
}

func (s *Server) handleListItems(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	id, ok := s.pathID(r)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "invalid collection id")
		return
	}
	items, err := s.inventory.ListItems(r.Context(), user.ID, id)
	if err != nil {
		s.writeInventoryError(w, r, err, "list items")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleGetItem(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	id, ok := s.pathID(r)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "invalid item id")
		return
	}
	item, err := s.inventory.GetItem(r.Context(), user.ID, id)
	if err != nil {
		s.writeInventoryError(w, r, err, "get item")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (s *Server) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	collectionID, ok := s.pathID(r)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "invalid collection id")
		return
	}
	var req itemPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid json payload")
		return
	}
	item, err := s.inventory.CreateItem(r.Context(), user.ID, collectionID, req.toInput())
	if err != nil {
		if isValidationErr(err) {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.writeInventoryError(w, r, err, "create item")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"item": item})

	// Notify all other users with access to the collection asynchronously.
	actorName := user.DisplayName
	if actorName == "" {
		actorName = user.Username
	}
	go func(collID int64, creatorID int64, actor, itemName string) {
		ctx := context.Background()
		accessors, err := s.inventory.CollectionAccessorIDs(ctx, collID)
		if err != nil {
			return
		}
		colName, err := s.inventory.CollectionName(ctx, collID)
		if err != nil {
			return
		}
		for _, uid := range accessors {
			if uid == creatorID {
				continue // don't self-notify
			}
			_ = s.notify.CreateInApp(ctx, notify.InAppInput{
				UserID: uid,
				Type:   "item_added",
				Title:  fmt.Sprintf("%s added a new item", actor),
				Body:   fmt.Sprintf("%s added \u201c%s\u201d to \u201c%s\u201d.", actor, itemName, colName),
				Link:   fmt.Sprintf("/collections/%d", collID),
			})
		}
	}(collectionID, user.ID, actorName, item.Name)
}

func (s *Server) handleUpdateItem(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	id, ok := s.pathID(r)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "invalid item id")
		return
	}
	var req itemPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid json payload")
		return
	}
	item, err := s.inventory.UpdateItem(r.Context(), user.ID, id, req.toInput())
	if err != nil {
		if isValidationErr(err) {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.writeInventoryError(w, r, err, "update item")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (s *Server) handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	id, ok := s.pathID(r)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "invalid item id")
		return
	}
	if err := s.inventory.DeleteItem(r.Context(), user.ID, id); err != nil {
		s.writeInventoryError(w, r, err, "delete item")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type moveItemPayload struct {
	CollectionID int64 `json:"collectionId"`
}

func (s *Server) handleMoveItem(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	id, ok := s.pathID(r)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "invalid item id")
		return
	}
	var req moveItemPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid json payload")
		return
	}
	if req.CollectionID <= 0 {
		writeAPIError(w, http.StatusBadRequest, "target collection is required")
		return
	}
	item, err := s.inventory.MoveItem(r.Context(), user.ID, id, req.CollectionID)
	if err != nil {
		if isValidationErr(err) {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.writeInventoryError(w, r, err, "move item")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (s *Server) handleItemStats(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	id, ok := s.pathID(r)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "invalid item id")
		return
	}
	stats, err := s.inventory.ItemStats(r.Context(), user.ID, id)
	if err != nil {
		s.writeInventoryError(w, r, err, "item stats")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stats": stats})
}

// saveUploadedFile reads the multipart "file" field, validates its extension
// against allowed, and stores it under the uploads directory. On failure it
// writes an HTTP error and returns ok=false. On success it returns the public
// path ("/uploads/<name>") and the original (sanitised) filename.
func (s *Server) saveUploadedFile(w http.ResponseWriter, r *http.Request, allowed map[string]bool, action string) (storedPath, originalName string, ok bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		writeAPIError(w, http.StatusBadRequest, "file too large or invalid upload")
		return "", "", false
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "missing file field")
		return "", "", false
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowed[ext] {
		writeAPIError(w, http.StatusBadRequest, "unsupported file type")
		return "", "", false
	}

	name, err := randomFileName(ext)
	if err != nil {
		s.writeInventoryError(w, r, err, action)
		return "", "", false
	}
	dest := filepath.Join(s.cfg.UploadsDir(), name)
	out, err := os.Create(dest)
	if err != nil {
		s.writeInventoryError(w, r, err, action)
		return "", "", false
	}
	if _, err := io.Copy(out, file); err != nil {
		out.Close()
		_ = os.Remove(dest)
		s.writeInventoryError(w, r, err, action)
		return "", "", false
	}
	out.Close()

	return "/uploads/" + name, filepath.Base(header.Filename), true
}

// handleUploadItemImage accepts a multipart "file" field and appends it to the
// item's image gallery.
func (s *Server) handleUploadItemImage(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	id, ok := s.pathID(r)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	stored, _, ok := s.saveUploadedFile(w, r, allowedImageExt, "upload image")
	if !ok {
		return
	}

	item, err := s.inventory.AddItemImage(r.Context(), user.ID, id, stored)
	if err != nil {
		_ = os.Remove(filepath.Join(s.cfg.UploadsDir(), filepath.Base(stored)))
		s.writeInventoryError(w, r, err, "upload image")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"item": item})
}

// handleDeleteItemImage removes an image from an item's gallery and deletes the
// underlying file.
func (s *Server) handleDeleteItemImage(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	id, ok := s.pathID(r)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "invalid item id")
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Path) == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid json payload")
		return
	}
	item, err := s.inventory.RemoveItemImage(r.Context(), user.ID, id, req.Path)
	if err != nil {
		s.writeInventoryError(w, r, err, "delete image")
		return
	}
	// Best-effort removal of the underlying file.
	_ = os.Remove(filepath.Join(s.cfg.UploadsDir(), filepath.Base(req.Path)))
	writeJSON(w, http.StatusOK, map[string]any{"item": item})
}

// handleUploadItemAttachment stores a file and appends it to an item.
func (s *Server) handleUploadItemAttachment(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	id, ok := s.pathID(r)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "invalid item id")
		return
	}
	stored, name, ok := s.saveUploadedFile(w, r, allowedAttachmentExt, "upload attachment")
	if !ok {
		return
	}
	item, err := s.inventory.AddItemAttachment(r.Context(), user.ID, id, inventory.Attachment{Name: name, Path: stored})
	if err != nil {
		_ = os.Remove(filepath.Join(s.cfg.UploadsDir(), filepath.Base(stored)))
		s.writeInventoryError(w, r, err, "upload attachment")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item": item})
}

// handleUploadEntryAttachment stores a file and appends it to an entry.
func (s *Server) handleUploadEntryAttachment(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	id, ok := s.pathID(r)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "invalid entry id")
		return
	}
	stored, name, ok := s.saveUploadedFile(w, r, allowedAttachmentExt, "upload attachment")
	if !ok {
		return
	}
	entry, err := s.inventory.AddEntryAttachment(r.Context(), user.ID, id, inventory.Attachment{Name: name, Path: stored})
	if err != nil {
		_ = os.Remove(filepath.Join(s.cfg.UploadsDir(), filepath.Base(stored)))
		s.writeInventoryError(w, r, err, "upload attachment")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entry": entry})
}

var allowedImageExt = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
}

// allowedAttachmentExt covers common document and image types for attachments.
var allowedAttachmentExt = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
	".pdf":  true,
	".txt":  true,
	".csv":  true,
	".doc":  true,
	".docx": true,
	".xls":  true,
	".xlsx": true,
	".zip":  true,
}

func randomFileName(ext string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf) + ext, nil
}

// ---------- Entries ----------

type entryPayload struct {
	Name        string                 `json:"name"`
	Amount      float64                `json:"amount"`
	Note        string                 `json:"note"`
	OccurredOn  string                 `json:"occurredOn"`
	Attachments []inventory.Attachment `json:"attachments"`
}

func (p entryPayload) toInput() inventory.EntryInput {
	return inventory.EntryInput{
		Name:        p.Name,
		Amount:      p.Amount,
		Note:        p.Note,
		OccurredOn:  p.OccurredOn,
		Attachments: p.Attachments,
	}
}

func (s *Server) handleListEntries(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	id, ok := s.pathID(r)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "invalid item id")
		return
	}
	entries, err := s.inventory.ListEntries(r.Context(), user.ID, id)
	if err != nil {
		s.writeInventoryError(w, r, err, "list entries")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries})
}

func (s *Server) handleCreateEntry(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	itemID, ok := s.pathID(r)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "invalid item id")
		return
	}
	var req entryPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid json payload")
		return
	}
	entry, err := s.inventory.CreateEntry(r.Context(), user.ID, itemID, req.toInput())
	if err != nil {
		if isValidationErr(err) {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.writeInventoryError(w, r, err, "create entry")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"entry": entry})
}

func (s *Server) handleUpdateEntry(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	id, ok := s.pathID(r)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "invalid entry id")
		return
	}
	var req entryPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid json payload")
		return
	}
	entry, err := s.inventory.UpdateEntry(r.Context(), user.ID, id, req.toInput())
	if err != nil {
		if isValidationErr(err) {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.writeInventoryError(w, r, err, "update entry")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entry": entry})
}

func (s *Server) handleDeleteEntry(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	id, ok := s.pathID(r)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "invalid entry id")
		return
	}
	if err := s.inventory.DeleteEntry(r.Context(), user.ID, id); err != nil {
		s.writeInventoryError(w, r, err, "delete entry")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---------- Search & portfolio ----------

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	query := r.URL.Query().Get("q")
	results, err := s.inventory.Search(r.Context(), user.ID, query)
	if err != nil {
		s.writeInventoryError(w, r, err, "search")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

func (s *Server) handlePortfolioStats(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	includeShared, err := s.auth.StatsIncludeShared(r.Context(), user.ID)
	if err != nil {
		s.writeInventoryError(w, r, err, "portfolio stats")
		return
	}
	stats, err := s.inventory.PortfolioStats(r.Context(), user.ID, includeShared)
	if err != nil {
		s.writeInventoryError(w, r, err, "portfolio stats")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stats": stats})
}

func (s *Server) handleRecentItems(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	includeShared, err := s.auth.StatsIncludeShared(r.Context(), user.ID)
	if err != nil {
		s.writeInventoryError(w, r, err, "recent items")
		return
	}
	limit := 8
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, convErr := strconv.Atoi(v); convErr == nil {
			limit = n
		}
	}
	items, err := s.inventory.RecentItems(r.Context(), user.ID, includeShared, limit)
	if err != nil {
		s.writeInventoryError(w, r, err, "recent items")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// isValidationErr reports whether an error is a user-facing validation failure
// (as opposed to an infrastructure error or not-found).
func isValidationErr(err error) bool {
	if err == nil || errors.Is(err, inventory.ErrNotFound) {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "required") || strings.Contains(msg, "too long")
}
