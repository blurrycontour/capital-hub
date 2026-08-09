package inventory_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/aditya/capital-hub/internal/database"
	"github.com/aditya/capital-hub/internal/inventory"
)

// testEnv is a migrated, throwaway SQLite database with an inventory service
// over it, plus two seeded accounts. Each test gets its own, so tests never
// share state.
type testEnv struct {
	t      *testing.T
	svc    *inventory.Service
	db     *sql.DB
	owner  int64
	sharee int64
}

func newEnv(t *testing.T) *testEnv {
	t.Helper()
	ctx := context.Background()

	db, err := database.Open(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	e := &testEnv{t: t, svc: inventory.NewService(db), db: db}
	e.owner = e.newUser("owner", "owner@example.com")
	e.sharee = e.newUser("sharee", "sharee@example.com")
	return e
}

// newUser seeds an account. The inventory package deliberately has no
// user-creation API, so tests insert directly.
func (e *testEnv) newUser(username, email string) int64 {
	e.t.Helper()
	res, err := e.db.ExecContext(context.Background(),
		`INSERT INTO users (username, email, display_name) VALUES (?, ?, ?)`,
		username, email, username,
	)
	if err != nil {
		e.t.Fatalf("insert user %s: %v", username, err)
	}
	id, _ := res.LastInsertId()
	return id
}

// newCollection creates a collection owned by the env's owner.
func (e *testEnv) newCollection(name string) int64 {
	e.t.Helper()
	c, err := e.svc.CreateCollection(context.Background(), e.owner, inventory.CollectionInput{
		Name: name, Description: "cars", Currency: "EUR",
	})
	if err != nil {
		e.t.Fatalf("create collection %s: %v", name, err)
	}
	return c.ID
}

// share grants the env's sharee the given access to a collection.
func (e *testEnv) share(collectionID int64, access string) {
	e.t.Helper()
	if _, err := e.svc.ShareCollection(context.Background(), e.owner, collectionID, "sharee", access); err != nil {
		e.t.Fatalf("share at %q: %v", access, err)
	}
}

// newItem creates an item in a collection as the owner.
func (e *testEnv) newItem(collectionID int64, name string) *inventory.Item {
	e.t.Helper()
	it, err := e.svc.CreateItem(context.Background(), e.owner, collectionID, inventory.ItemInput{Name: name})
	if err != nil {
		e.t.Fatalf("create item %s: %v", name, err)
	}
	return it
}

// newEntry records an entry against an item as the owner.
func (e *testEnv) newEntry(itemID int64, name string, amount float64, kind string) *inventory.Entry {
	e.t.Helper()
	en, err := e.svc.CreateEntry(context.Background(), e.owner, itemID, inventory.EntryInput{
		Name: name, Amount: amount, Kind: kind, OccurredOn: "2026-02-01",
	})
	if err != nil {
		e.t.Fatalf("create entry %s: %v", name, err)
	}
	return en
}
