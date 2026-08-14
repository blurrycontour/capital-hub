package inventory_test

import (
	"context"
	"testing"

	"github.com/aditya/capital-hub/internal/inventory"
)

// TestEntryFromToRoundTrip covers the optional counterparty fields: they
// persist, they survive an update, and they are trimmed rather than stored raw.
func TestEntryFromToRoundTrip(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	colID := e.newCollection("Vehicles")
	item := e.newItem(colID, "Estate car")

	created, err := e.svc.CreateEntry(ctx, e.owner, item.ID, inventory.EntryInput{
		Name: "Purchase", Amount: 100, Kind: "debit", OccurredOn: "2026-02-01",
		From: "  Alice  ", To: "\tBob\n",
	})
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}
	if created.From != "Alice" || created.To != "Bob" {
		t.Errorf("from/to = %q/%q, want Alice/Bob (trimmed)", created.From, created.To)
	}

	// Values must come back from the database, not just the create response.
	fetched, err := e.svc.GetEntry(ctx, e.owner, created.ID)
	if err != nil {
		t.Fatalf("get entry: %v", err)
	}
	if fetched.From != "Alice" || fetched.To != "Bob" {
		t.Errorf("persisted from/to = %q/%q, want Alice/Bob", fetched.From, fetched.To)
	}

	updated, err := e.svc.UpdateEntry(ctx, e.owner, created.ID, inventory.EntryInput{
		Name: "Purchase", Amount: 100, Kind: "debit", OccurredOn: "2026-02-01",
		From: "Carol", To: "",
	})
	if err != nil {
		t.Fatalf("update entry: %v", err)
	}
	if updated.From != "Carol" || updated.To != "" {
		t.Errorf("after update from/to = %q/%q, want Carol/empty", updated.From, updated.To)
	}
}

// TestEntryFromToAreOptional keeps them optional: omitting both must not fail
// or leave nulls that break scanning.
func TestEntryFromToAreOptional(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	item := e.newItem(e.newCollection("Vehicles"), "Estate car")

	created, err := e.svc.CreateEntry(ctx, e.owner, item.ID, inventory.EntryInput{
		Name: "Purchase", Amount: 10, Kind: "debit", OccurredOn: "2026-02-01",
	})
	if err != nil {
		t.Fatalf("create entry without from/to: %v", err)
	}
	if created.From != "" || created.To != "" {
		t.Errorf("from/to = %q/%q, want empty", created.From, created.To)
	}

	list, err := e.svc.ListEntries(ctx, e.owner, item.ID)
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("entries = %d, want 1", len(list))
	}
}

// TestEntryFromToAreSearchable checks the full-text triggers were rebuilt to
// index the new fields, for entries written after the migration.
func TestEntryFromToAreSearchable(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	item := e.newItem(e.newCollection("Vehicles"), "Estate car")

	if _, err := e.svc.CreateEntry(ctx, e.owner, item.ID, inventory.EntryInput{
		Name: "Purchase", Amount: 10, Kind: "debit", OccurredOn: "2026-02-01",
		From: "Zenith Motors", To: "Household",
	}); err != nil {
		t.Fatalf("create entry: %v", err)
	}

	for _, term := range []string{"Zenith", "Household"} {
		results, err := e.svc.Search(ctx, e.owner, term)
		if err != nil {
			t.Fatalf("search %q: %v", term, err)
		}
		found := false
		for _, r := range results {
			if r.Type == "entry" {
				found = true
			}
		}
		if !found {
			t.Errorf("searching %q found no entry; from/to are not indexed", term)
		}
	}
}

// ---------- Suggestions ----------

// TestEntrySuggestionsRankAndDedupe checks the ordering the form relies on:
// most-used first, each value once, blanks excluded.
func TestEntrySuggestionsRankAndDedupe(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	item := e.newItem(e.newCollection("Vehicles"), "Estate car")

	seed := []struct{ name, from, to string }{
		{"Service", "Garage", "Household"},
		{"Service", "Garage", ""},
		{"Service", "Dealer", "Household"},
		{"Fuel", "Station", "Household"},
		{"Insurance", "", "Household"},
	}
	for _, row := range seed {
		if _, err := e.svc.CreateEntry(ctx, e.owner, item.ID, inventory.EntryInput{
			Name: row.name, Amount: 1, Kind: "debit", OccurredOn: "2026-02-01",
			From: row.from, To: row.to,
		}); err != nil {
			t.Fatalf("create entry %+v: %v", row, err)
		}
	}

	cases := []struct {
		field string
		want  []string
	}{
		// "Service" appears three times, so it leads; the rest tie and sort
		// alphabetically.
		{"name", []string{"Service", "Fuel", "Insurance"}},
		{"from", []string{"Garage", "Dealer", "Station"}},
		// The one blank `to` must not become a suggestion.
		{"to", []string{"Household"}},
	}
	for _, tc := range cases {
		t.Run(tc.field, func(t *testing.T) {
			got, err := e.svc.EntrySuggestions(ctx, e.owner, tc.field)
			if err != nil {
				t.Fatalf("suggestions: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("position %d = %q, want %q (full: %v)", i, got[i], tc.want[i], got)
				}
			}
		})
	}
}

// TestEntrySuggestionsRejectsUnknownField keeps the field name closed, since it
// is interpolated into the query.
func TestEntrySuggestionsRejectsUnknownField(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	for _, bad := range []string{"note", "amount", "", "name; DROP TABLE entries"} {
		if _, err := e.svc.EntrySuggestions(ctx, e.owner, bad); err == nil {
			t.Errorf("field %q should be rejected", bad)
		}
	}
}

// TestEntrySuggestionsRespectAccess is the important one: suggestions are drawn
// from other people's entries too, so they must stop at the sharing boundary.
func TestEntrySuggestionsRespectAccess(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	outsider := e.newUser("mallory", "mallory@example.com")

	colID := e.newCollection("Vehicles")
	item := e.newItem(colID, "Estate car")
	if _, err := e.svc.CreateEntry(ctx, e.owner, item.ID, inventory.EntryInput{
		Name: "Confidential", Amount: 1, Kind: "debit", OccurredOn: "2026-02-01",
		From: "Secret Counterparty", To: "Secret Destination",
	}); err != nil {
		t.Fatalf("create entry: %v", err)
	}

	for _, field := range []string{"name", "from", "to"} {
		got, err := e.svc.EntrySuggestions(ctx, outsider, field)
		if err != nil {
			t.Fatalf("suggestions for outsider: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("outsider sees %s suggestions %v, want none", field, got)
		}
	}

	// A read-only sharee legitimately sees them, since they can read the entries.
	e.share(colID, inventory.AccessRead)
	got, err := e.svc.EntrySuggestions(ctx, e.sharee, "from")
	if err != nil {
		t.Fatalf("suggestions for sharee: %v", err)
	}
	if len(got) != 1 || got[0] != "Secret Counterparty" {
		t.Errorf("sharee sees %v, want the one shared value", got)
	}
}
