package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/aditya/capital-hub/internal/inventory"
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
	default:
		s.logger.ErrorContext(r.Context(), action+" failed", "error", err)
		writeAPIError(w, http.StatusInternalServerError, action+" failed")
	}
}

// ---------- Collections ----------

type collectionPayload struct {
	Name        string `json:"name"`
	Description string `json:"description"`
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
	c, err := s.inventory.CreateCollection(r.Context(), user.ID, req.Name, req.Description)
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
	c, err := s.inventory.UpdateCollection(r.Context(), user.ID, id, req.Name, req.Description)
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

// ---------- Items ----------

type itemPayload struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	LocationLat   *float64 `json:"locationLat"`
	LocationLng   *float64 `json:"locationLng"`
	LocationLabel string   `json:"locationLabel"`
}

func (p itemPayload) toInput() inventory.ItemInput {
	return inventory.ItemInput{
		Name:          p.Name,
		Description:   p.Description,
		LocationLat:   p.LocationLat,
		LocationLng:   p.LocationLng,
		LocationLabel: p.LocationLabel,
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

// handleUploadItemImage accepts a multipart "file" field and stores it on disk.
func (s *Server) handleUploadItemImage(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	id, ok := s.pathID(r)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		writeAPIError(w, http.StatusBadRequest, "file too large or invalid upload")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedImageExt[ext] {
		writeAPIError(w, http.StatusBadRequest, "unsupported image type")
		return
	}

	name, err := randomFileName(ext)
	if err != nil {
		s.writeInventoryError(w, r, err, "upload image")
		return
	}
	dest := filepath.Join(s.cfg.UploadsDir(), name)
	out, err := os.Create(dest)
	if err != nil {
		s.writeInventoryError(w, r, err, "upload image")
		return
	}
	if _, err := io.Copy(out, file); err != nil {
		out.Close()
		_ = os.Remove(dest)
		s.writeInventoryError(w, r, err, "upload image")
		return
	}
	out.Close()

	prev, err := s.inventory.SetItemImage(r.Context(), user.ID, id, "/uploads/"+name)
	if err != nil {
		_ = os.Remove(dest)
		s.writeInventoryError(w, r, err, "upload image")
		return
	}
	// Best-effort cleanup of the replaced file.
	if prev != "" {
		_ = os.Remove(filepath.Join(s.cfg.UploadsDir(), filepath.Base(prev)))
	}

	item, err := s.inventory.GetItem(r.Context(), user.ID, id)
	if err != nil {
		s.writeInventoryError(w, r, err, "upload image")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item": item})
}

var allowedImageExt = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
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
	Kind       string  `json:"kind"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	Quantity   float64 `json:"quantity"`
	Note       string  `json:"note"`
	OccurredOn string  `json:"occurredOn"`
}

func (p entryPayload) toInput() inventory.EntryInput {
	return inventory.EntryInput{
		Kind:       p.Kind,
		Amount:     p.Amount,
		Currency:   p.Currency,
		Quantity:   p.Quantity,
		Note:       p.Note,
		OccurredOn: p.OccurredOn,
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
	stats, err := s.inventory.PortfolioStats(r.Context(), user.ID)
	if err != nil {
		s.writeInventoryError(w, r, err, "portfolio stats")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stats": stats})
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
