package inventory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aditya/capital-hub/internal/inventory"
)

// ---------- Cross-user isolation ----------

// TestOutsiderSeesNothing is the base invariant the whole service rests on: a
// user with no relationship to a collection can neither read nor change it or
// anything under it, and it never appears in their listings.
func TestOutsiderSeesNothing(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	svc, ownerID := e.svc, e.owner
	outsiderID := e.newUser("mallory", "mallory@example.com")

	colID := e.newCollection("Vehicles")
	item := e.newItem(colID, "Laptop")
	entry := e.newEntry(item.ID, "Purchase", 100, "debit")

	t.Run("reads", func(t *testing.T) {
		if _, err := svc.GetCollection(ctx, outsiderID, colID); !errors.Is(err, inventory.ErrNotFound) {
			t.Errorf("GetCollection = %v, want ErrNotFound", err)
		}
		if _, err := svc.GetItem(ctx, outsiderID, item.ID); !errors.Is(err, inventory.ErrNotFound) {
			t.Errorf("GetItem = %v, want ErrNotFound", err)
		}
		if _, err := svc.ListItems(ctx, outsiderID, colID); !errors.Is(err, inventory.ErrNotFound) {
			t.Errorf("ListItems = %v, want ErrNotFound", err)
		}
		if _, err := svc.ListEntries(ctx, outsiderID, item.ID); !errors.Is(err, inventory.ErrNotFound) {
			t.Errorf("ListEntries = %v, want ErrNotFound", err)
		}
		if _, err := svc.CollectionStats(ctx, outsiderID, colID); !errors.Is(err, inventory.ErrNotFound) {
			t.Errorf("CollectionStats = %v, want ErrNotFound", err)
		}
	})

	t.Run("writes", func(t *testing.T) {
		if _, err := svc.UpdateCollection(ctx, outsiderID, colID, inventory.CollectionInput{Name: "Taken", Currency: "EUR"}); err == nil {
			t.Error("UpdateCollection should fail for an outsider")
		}
		if err := svc.DeleteCollection(ctx, outsiderID, colID); err == nil {
			t.Error("DeleteCollection should fail for an outsider")
		}
		if _, err := svc.CreateItem(ctx, outsiderID, colID, inventory.ItemInput{Name: "Sneaky"}); err == nil {
			t.Error("CreateItem should fail for an outsider")
		}
		if _, err := svc.UpdateItem(ctx, outsiderID, item.ID, inventory.ItemInput{Name: "Renamed"}); err == nil {
			t.Error("UpdateItem should fail for an outsider")
		}
		if err := svc.DeleteItem(ctx, outsiderID, item.ID); err == nil {
			t.Error("DeleteItem should fail for an outsider")
		}
		if err := svc.DeleteEntry(ctx, outsiderID, entry.ID); err == nil {
			t.Error("DeleteEntry should fail for an outsider")
		}
	})

	t.Run("listing", func(t *testing.T) {
		cols, err := svc.ListCollections(ctx, outsiderID)
		if err != nil {
			t.Fatalf("ListCollections: %v", err)
		}
		if len(cols) != 0 {
			t.Errorf("outsider sees %d collections, want 0", len(cols))
		}
	})

	// Nothing above should have taken effect.
	got, err := svc.GetCollection(ctx, ownerID, colID)
	if err != nil {
		t.Fatalf("collection should survive: %v", err)
	}
	if got.Name != "Vehicles" {
		t.Errorf("name = %q, want it unchanged", got.Name)
	}
}

// ---------- Contents permissions by share level ----------

// TestContentsPermissionsByLevel covers items and entries, which are reached
// through the parent collection's access level rather than one of their own.
func TestContentsPermissionsByLevel(t *testing.T) {
	cases := []struct {
		access   string
		canWrite bool
	}{
		{inventory.AccessRead, false},
		{inventory.AccessWrite, true},
		{inventory.AccessFull, true},
	}

	for _, tc := range cases {
		t.Run(tc.access, func(t *testing.T) {
			ctx := context.Background()
			e := newEnv(t)
			svc, shareeID := e.svc, e.sharee
			colID := e.newCollection("Vehicles")
			e.share(colID, tc.access)

			seed := e.newItem(colID, "Laptop")

			// Reading is allowed at every level.
			if _, err := svc.GetItem(ctx, shareeID, seed.ID); err != nil {
				t.Fatalf("%s should read items: %v", tc.access, err)
			}

			_, createErr := svc.CreateItem(ctx, shareeID, colID, inventory.ItemInput{Name: "Added"})
			_, updateErr := svc.UpdateItem(ctx, shareeID, seed.ID, inventory.ItemInput{Name: "Renamed"})
			_, entryErr := svc.CreateEntry(ctx, shareeID, seed.ID, inventory.EntryInput{
				Name: "Purchase", Amount: 10, Kind: "debit", OccurredOn: "2026-02-01",
			})

			if tc.canWrite {
				for name, err := range map[string]error{"CreateItem": createErr, "UpdateItem": updateErr, "CreateEntry": entryErr} {
					if err != nil {
						t.Errorf("%s should allow %s: %v", tc.access, name, err)
					}
				}
				return
			}
			for name, err := range map[string]error{"CreateItem": createErr, "UpdateItem": updateErr, "CreateEntry": entryErr} {
				if !errors.Is(err, inventory.ErrForbidden) {
					t.Errorf("%s must not allow %s, got %v", tc.access, name, err)
				}
			}
		})
	}
}

// TestMoveItemRequiresWriteOnTarget guards the one operation that spans two
// collections: read access to the destination must not be enough to move an
// item into it.
func TestMoveItemRequiresWriteOnTarget(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	svc, ownerID, shareeID := e.svc, e.owner, e.sharee

	source := e.newCollection("Vehicles")
	e.share(source, inventory.AccessWrite)

	// A second collection the sharee can only read.
	target, err := svc.CreateCollection(ctx, ownerID, inventory.CollectionInput{Name: "Archive", Currency: "EUR"})
	if err != nil {
		t.Fatalf("create target: %v", err)
	}
	e.share(target.ID, inventory.AccessRead)

	item := e.newItem(source, "Laptop")

	if _, err := svc.MoveItem(ctx, shareeID, item.ID, target.ID); !errors.Is(err, inventory.ErrForbidden) {
		t.Fatalf("move into a read-only collection = %v, want ErrForbidden", err)
	}
	// The item must not have moved.
	got, err := svc.GetItem(ctx, ownerID, item.ID)
	if err != nil {
		t.Fatalf("get item: %v", err)
	}
	if got.CollectionID != source {
		t.Errorf("item moved to %d despite the error, want %d", got.CollectionID, source)
	}

	// The owner can move it, confirming the failure above was about permission.
	if _, err := svc.MoveItem(ctx, ownerID, item.ID, target.ID); err != nil {
		t.Fatalf("owner should move the item: %v", err)
	}
}

// ---------- Stats ----------

// TestStatsAggregation checks the debit/credit/net arithmetic and that totals
// are split per currency, since a collection's currency can change.
func TestStatsAggregation(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	svc, ownerID := e.svc, e.owner
	colID := e.newCollection("Vehicles")

	item := e.newItem(colID, "Laptop")
	seed := []struct {
		name   string
		amount float64
		kind   string
	}{
		{"Purchase", 1000, "debit"},
		{"Accessory", 250.50, "debit"},
		{"Rebate", 400, "credit"},
	}
	for _, s := range seed {
		e.newEntry(item.ID, s.name, s.amount, s.kind)
	}

	stats, err := svc.CollectionStats(ctx, ownerID, colID)
	if err != nil {
		t.Fatalf("collection stats: %v", err)
	}
	if stats.ItemCount != 1 {
		t.Errorf("itemCount = %d, want 1", stats.ItemCount)
	}
	if stats.EntryCount != 3 {
		t.Errorf("entryCount = %d, want 3", stats.EntryCount)
	}
	if len(stats.Totals) != 1 {
		t.Fatalf("totals = %d currencies, want 1", len(stats.Totals))
	}
	tot := stats.Totals[0]
	if tot.Currency != "EUR" {
		t.Errorf("currency = %q, want EUR", tot.Currency)
	}
	if tot.Debit != 1250.50 {
		t.Errorf("debit = %v, want 1250.50", tot.Debit)
	}
	if tot.Credit != 400 {
		t.Errorf("credit = %v, want 400", tot.Credit)
	}
	// Net is credit - debit, so a net expense is negative.
	if tot.Net != 400-1250.50 {
		t.Errorf("net = %v, want %v", tot.Net, 400-1250.50)
	}

	// Item-level stats should agree, since this collection has a single item.
	itemStats, err := svc.ItemStats(ctx, ownerID, item.ID)
	if err != nil {
		t.Fatalf("item stats: %v", err)
	}
	if itemStats.EntryCount != stats.EntryCount {
		t.Errorf("item entryCount = %d, want %d", itemStats.EntryCount, stats.EntryCount)
	}
	if len(itemStats.Totals) != 1 || itemStats.Totals[0].Net != tot.Net {
		t.Errorf("item totals = %+v, want net %v", itemStats.Totals, tot.Net)
	}
}

// TestEmptyCollectionStats makes sure an untouched collection reports zeroes
// and no currency rows, which is what lets the UI hide the money cards.
func TestEmptyCollectionStats(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	svc, ownerID := e.svc, e.owner
	colID := e.newCollection("Vehicles")

	stats, err := svc.CollectionStats(ctx, ownerID, colID)
	if err != nil {
		t.Fatalf("collection stats: %v", err)
	}
	if stats.ItemCount != 0 || stats.EntryCount != 0 {
		t.Errorf("counts = %d items / %d entries, want 0/0", stats.ItemCount, stats.EntryCount)
	}
	if len(stats.Totals) != 0 {
		t.Errorf("totals = %+v, want none", stats.Totals)
	}
}

// ---------- Sharing mechanics ----------

// TestResharingUpdatesLevel covers the upsert path: sharing with the same
// person twice must change their level rather than duplicate the row.
func TestResharingUpdatesLevel(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	svc, ownerID, shareeID := e.svc, e.owner, e.sharee
	colID := e.newCollection("Vehicles")

	e.share(colID, inventory.AccessRead)
	e.share(colID, inventory.AccessFull)

	shares, err := svc.ListShares(ctx, ownerID, colID)
	if err != nil {
		t.Fatalf("list shares: %v", err)
	}
	if len(shares) != 1 {
		t.Fatalf("shares = %d, want 1 (upsert, not duplicate)", len(shares))
	}
	if shares[0].Access != inventory.AccessFull {
		t.Errorf("access = %q, want %q", shares[0].Access, inventory.AccessFull)
	}

	got, err := svc.GetCollection(ctx, shareeID, colID)
	if err != nil {
		t.Fatalf("get as sharee: %v", err)
	}
	if got.AccessLevel != inventory.AccessFull {
		t.Errorf("accessLevel = %q, want %q", got.AccessLevel, inventory.AccessFull)
	}
}

// TestUnshareRevokesAccess confirms access actually disappears, rather than
// merely being hidden from the sharing dialog.
func TestUnshareRevokesAccess(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	svc, ownerID, shareeID := e.svc, e.owner, e.sharee
	colID := e.newCollection("Vehicles")
	e.share(colID, inventory.AccessWrite)

	if _, err := svc.GetCollection(ctx, shareeID, colID); err != nil {
		t.Fatalf("sharee should have access first: %v", err)
	}
	if err := svc.UnshareCollection(ctx, ownerID, colID, shareeID); err != nil {
		t.Fatalf("unshare: %v", err)
	}
	if _, err := svc.GetCollection(ctx, shareeID, colID); !errors.Is(err, inventory.ErrNotFound) {
		t.Errorf("after unshare = %v, want ErrNotFound", err)
	}
}

// TestCannotShareWithSelf guards against an owner demoting themselves by
// creating a share row against their own collection.
func TestCannotShareWithSelf(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	svc, ownerID := e.svc, e.owner
	colID := e.newCollection("Vehicles")

	if _, err := svc.ShareCollection(ctx, ownerID, colID, "owner", inventory.AccessRead); err == nil {
		t.Fatal("sharing with yourself should be rejected")
	}
	got, err := svc.GetCollection(ctx, ownerID, colID)
	if err != nil {
		t.Fatalf("get collection: %v", err)
	}
	if got.AccessLevel != inventory.AccessOwner {
		t.Errorf("accessLevel = %q, want %q", got.AccessLevel, inventory.AccessOwner)
	}
}

// TestSharedCollectionAppearsInListing checks the sharee's collection list
// includes it and reports the owner, which the UI shows on the card.
func TestSharedCollectionAppearsInListing(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	svc, shareeID := e.svc, e.sharee
	e.share(e.newCollection("Vehicles"), inventory.AccessWrite)

	cols, err := svc.ListCollections(ctx, shareeID)
	if err != nil {
		t.Fatalf("list collections: %v", err)
	}
	if len(cols) != 1 {
		t.Fatalf("sharee sees %d collections, want 1", len(cols))
	}
	if !cols[0].Shared {
		t.Error("collection should be marked shared for the sharee")
	}
	if cols[0].AccessLevel != inventory.AccessWrite {
		t.Errorf("accessLevel = %q, want %q", cols[0].AccessLevel, inventory.AccessWrite)
	}
	if cols[0].OwnerName == "" {
		t.Error("ownerName should be populated so the UI can attribute the collection")
	}
}
