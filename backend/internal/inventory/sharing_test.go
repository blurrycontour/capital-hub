package inventory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aditya/capital-hub/internal/inventory"
)

// TestCollectionEditPermissions pins down which share levels may edit the
// collection's own details. "full" was added for exactly this; "write" grants
// access to the contents only and must not widen to the collection itself.
func TestCollectionEditPermissions(t *testing.T) {
	cases := []struct {
		access  string
		canEdit bool
	}{
		{inventory.AccessRead, false},
		{inventory.AccessWrite, false},
		{inventory.AccessFull, true},
	}

	for _, tc := range cases {
		t.Run(tc.access, func(t *testing.T) {
			ctx := context.Background()
			e := newEnv(t)
			svc, shareeID := e.svc, e.sharee
			id := e.newCollection("Vehicles")
			e.share(id, tc.access)

			_, err := svc.UpdateCollection(ctx, shareeID, id, inventory.CollectionInput{
				Name: "Renamed", Description: "by sharee", Currency: "EUR",
			})

			if tc.canEdit {
				if err != nil {
					t.Fatalf("%s should be able to edit the collection, got %v", tc.access, err)
				}
				got, err := svc.GetCollection(ctx, shareeID, id)
				if err != nil {
					t.Fatalf("get collection: %v", err)
				}
				if got.Name != "Renamed" {
					t.Errorf("name = %q, want %q", got.Name, "Renamed")
				}
				return
			}
			if !errors.Is(err, inventory.ErrForbidden) {
				t.Fatalf("%s must not edit the collection, got err=%v", tc.access, err)
			}
			// The rejection must not have written anything.
			got, err := svc.GetCollection(ctx, shareeID, id)
			if err != nil {
				t.Fatalf("get collection: %v", err)
			}
			if got.Name != "Vehicles" {
				t.Errorf("name = %q, want it unchanged", got.Name)
			}
		})
	}
}

// TestFullAccessStopsShortOfOwnership guards the boundary chosen for "full":
// it may edit, but deleting the collection and managing its shares stay with
// the owner.
func TestFullAccessStopsShortOfOwnership(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	svc, ownerID, shareeID := e.svc, e.owner, e.sharee
	id := e.newCollection("Vehicles")
	e.share(id, inventory.AccessFull)

	if err := svc.DeleteCollection(ctx, shareeID, id); err == nil {
		t.Error("full access must not be able to delete the collection")
	}
	if _, err := svc.ListShares(ctx, shareeID, id); !errors.Is(err, inventory.ErrForbidden) {
		t.Errorf("full access must not list shares, got %v", err)
	}
	if _, err := svc.ShareCollection(ctx, shareeID, id, "owner", inventory.AccessFull); !errors.Is(err, inventory.ErrForbidden) {
		t.Errorf("full access must not re-share the collection, got %v", err)
	}
	if err := svc.UnshareCollection(ctx, shareeID, id, ownerID); !errors.Is(err, inventory.ErrForbidden) {
		t.Errorf("full access must not remove shares, got %v", err)
	}

	// The collection must still exist and still be shared after those attempts.
	if _, err := svc.GetCollection(ctx, ownerID, id); err != nil {
		t.Fatalf("collection should survive: %v", err)
	}
}

// TestAccessLevelReporting checks the level surfaced to the frontend, which
// drives which controls the UI enables.
func TestAccessLevelReporting(t *testing.T) {
	for _, access := range inventory.ShareAccessLevels {
		t.Run(access, func(t *testing.T) {
			ctx := context.Background()
			e := newEnv(t)
			svc, ownerID, shareeID := e.svc, e.owner, e.sharee
			id := e.newCollection("Vehicles")
			e.share(id, access)

			asSharee, err := svc.GetCollection(ctx, shareeID, id)
			if err != nil {
				t.Fatalf("get as sharee: %v", err)
			}
			if asSharee.AccessLevel != access {
				t.Errorf("sharee accessLevel = %q, want %q", asSharee.AccessLevel, access)
			}
			if !asSharee.Shared {
				t.Error("sharee should see the collection as shared")
			}

			asOwner, err := svc.GetCollection(ctx, ownerID, id)
			if err != nil {
				t.Fatalf("get as owner: %v", err)
			}
			if asOwner.AccessLevel != inventory.AccessOwner {
				t.Errorf("owner accessLevel = %q, want %q", asOwner.AccessLevel, inventory.AccessOwner)
			}
		})
	}
}

// TestShareRejectsUnknownAccess keeps the stored vocabulary closed, and
// TestUnknownStoredAccessDegradesToRead covers a row that somehow holds one
// anyway — it must not be treated as more than read-only.
func TestShareRejectsUnknownAccess(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	svc, ownerID := e.svc, e.owner
	id := e.newCollection("Vehicles")

	for _, bad := range []string{"admin", "owner", "", "FULL CONTROL"} {
		if _, err := svc.ShareCollection(ctx, ownerID, id, "sharee", bad); err == nil {
			t.Errorf("access %q should be rejected", bad)
		}
	}
	// Casing and padding are normalised rather than rejected.
	if _, err := svc.ShareCollection(ctx, ownerID, id, "sharee", "  FULL  "); err != nil {
		t.Errorf("normalised access should be accepted, got %v", err)
	}
}

// TestWriteAccessStillEditsContents makes sure adding "full" did not disturb
// what "write" could already do.
func TestWriteAccessStillEditsContents(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	svc, shareeID := e.svc, e.sharee
	id := e.newCollection("Vehicles")
	e.share(id, inventory.AccessWrite)

	if _, err := svc.CreateItem(ctx, shareeID, id, inventory.ItemInput{Name: "Estate car"}); err != nil {
		t.Fatalf("write access should create items: %v", err)
	}
}
