package game

import (
	"database/sql"
	"math/big"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestFacilityCostGrowth(t *testing.T) {
	oldCost := FacilityCost(FacilityDefs[19], 2)
	wantOld := newRat(t, "27562500000000000000000")
	if oldCost.Cmp(wantOld) != 0 {
		t.Fatalf("facility 20 cost = %s, want %s", oldCost.RatString(), wantOld.RatString())
	}

	newCost := FacilityCost(FacilityDefs[20], 2)
	wantNew := newRat(t, "4687500000000000000000000")
	if newCost.Cmp(wantNew) != 0 {
		t.Fatalf("facility 21 cost = %s, want %s", newCost.RatString(), wantNew.RatString())
	}
}

func TestDecimalStringIsExact(t *testing.T) {
	value := newRat(t, "123456789012345678901234567890123456789/100000000000000000000")
	want := "1234567890123456789.01234567890123456789"
	if got := decimalString(value); got != want {
		t.Fatalf("decimalString() = %q, want %q", got, want)
	}
}

func TestStoreMigratesLegacyGold(t *testing.T) {
	path := filepath.Join(t.TempDir(), "game.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
CREATE TABLE game_state (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  gold REAL NOT NULL DEFAULT 0,
  diamonds INTEGER NOT NULL DEFAULT 0,
  click_level INTEGER NOT NULL DEFAULT 0,
  facilities TEXT NOT NULL DEFAULT '[]',
  updated_at INTEGER NOT NULL DEFAULT 0
);
INSERT INTO game_state VALUES (1, 123456789.25, 3, 4, '[]', 0);
`)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	state, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if state.Gold != "123456789.250000" {
		t.Fatalf("migrated gold = %q", state.Gold)
	}
	state.Gold = "123456789012345678901234567890.125"
	if err := store.Save(state); err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Gold != state.Gold {
		t.Fatalf("reloaded gold = %q, want %q", reloaded.Gold, state.Gold)
	}
}

func newRat(t *testing.T, value string) *big.Rat {
	t.Helper()
	r, ok := new(big.Rat).SetString(value)
	if !ok {
		t.Fatalf("invalid rational %q", value)
	}
	return r
}
