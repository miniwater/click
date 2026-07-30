package game

import (
	"database/sql"
	"math/big"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestFacilityCostGrowth(t *testing.T) {
	oldCost := FacilityCost(FacilityDefs[19], 2)
	wantOld := amount("27562500000000000000000")
	if !amountsClose(oldCost, wantOld) {
		t.Fatalf("facility 20 cost = %s, want %s", oldCost.String(), wantOld.String())
	}

	newCost := FacilityCost(FacilityDefs[20], 2)
	wantNew := amount("4687500000000000000000000")
	if !amountsClose(newCost, wantNew) {
		t.Fatalf("facility 21 cost = %s, want %s", newCost.String(), wantNew.String())
	}

	for _, tc := range []struct {
		id   int
		want string
	}{
		{31, "1.5"},
		{41, "1.75"},
		{51, "2"},
	} {
		if got := facilityCostGrowth(FacilityDefs[tc.id-1]); got.String() != amount(tc.want).String() {
			t.Fatalf("facility %d growth = %s, want %s", tc.id, got.String(), tc.want)
		}
	}
}

func amountsClose(got, want *Amount) bool {
	difference := new(big.Float).SetPrec(amountPrecision).Sub(got.v, want.v)
	difference.Abs(difference)
	tolerance := new(big.Float).SetPrec(amountPrecision).Mul(
		new(big.Float).SetPrec(amountPrecision).Abs(want.v),
		amount("1e-35").v,
	)
	return difference.Cmp(tolerance) <= 0
}

func TestFacilityCatalogAndLegacyNormalization(t *testing.T) {
	if len(FacilityDefs) != 60 {
		t.Fatalf("facility count = %d, want 60", len(FacilityDefs))
	}
	previousCost := zeroAmount()
	legacy := make([]FacilityState, 30)
	for i, def := range FacilityDefs {
		if def.ID != i+1 {
			t.Fatalf("facility index %d has ID %d", i, def.ID)
		}
		cost := amount(def.BaseCost)
		cps := amount(def.BaseCPS)
		if cost.Sign() <= 0 || cps.Sign() <= 0 {
			t.Fatalf("facility %d has invalid economy values", def.ID)
		}
		if i > 0 && cost.Cmp(previousCost) <= 0 {
			t.Fatalf("facility %d base cost is not increasing", def.ID)
		}
		previousCost = cost
		if i < len(legacy) {
			legacy[i] = FacilityState{ID: def.ID, Owned: i + 1, Enhance: i}
		}
	}

	normalized := normalizeFacilities(legacy)
	if len(normalized) != 60 {
		t.Fatalf("normalized facility count = %d, want 60", len(normalized))
	}
	if normalized[29] != legacy[29] {
		t.Fatal("legacy facility state was not preserved")
	}
	if normalized[30] != (FacilityState{ID: 31}) || normalized[59] != (FacilityState{ID: 60}) {
		t.Fatal("new facility defaults were not appended")
	}
}

func TestAmountSerializationStaysBounded(t *testing.T) {
	legacy := strings.Repeat("9", 7000) + "." + strings.Repeat("8", 7000)
	serialized := amount(legacy).String()
	if len(serialized) > 50 {
		t.Fatalf("serialized amount length = %d", len(serialized))
	}
	if !strings.Contains(serialized, "e7000") {
		t.Fatalf("serialized amount = %q", serialized)
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

func TestStoreRewritesHugeLegacyGoldAsScientific(t *testing.T) {
	path := filepath.Join(t.TempDir(), "game.db")
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	legacy := strings.Repeat("7", 6000) + ".123456789"
	state := defaultPersisted()
	state.Gold = legacy
	if err := store.Save(state); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	loaded.Gold = amount(loaded.Gold).String()
	if err := store.Save(loaded); err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Gold) > 50 || !strings.Contains(reloaded.Gold, "e5999") {
		t.Fatalf("rewritten gold = %q", reloaded.Gold)
	}
}
