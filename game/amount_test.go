package game

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAmountScientificRoundTrip(t *testing.T) {
	for _, input := range []string{
		"0",
		"0.1",
		"123.456",
		"3.3228893783800823026439473159191827305e76",
		strings.Repeat("9", 5000),
	} {
		value := amount(input)
		serialized := value.String()
		if len(serialized) > 50 {
			t.Fatalf("serialized %q has length %d", input[:min(len(input), 20)], len(serialized))
		}
		if amount(serialized).Cmp(value) != 0 {
			t.Fatalf("round trip changed %q to %q", input[:min(len(input), 20)], serialized)
		}
	}
}

func TestHighLevelEconomyStringsStayBounded(t *testing.T) {
	values := []*Amount{
		ClickPower(100000),
		ClickUpgradeCost(100000),
		FacilityCost(FacilityDefs[59], 100000),
		FacilityUnitCPS(FacilityDefs[59], 100000),
	}
	for _, value := range values {
		if value.Sign() <= 0 || len(value.String()) > 50 {
			t.Fatalf("invalid high-level amount %q", value.String())
		}
	}
}

func TestEngineLoadsLegacyGoldIntoBoundedSnapshot(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	legacy := strings.Repeat("8", 6000) + ".123456789"
	state := defaultPersisted()
	state.Gold = legacy
	state.ClickLevel = 100000
	if err := store.Save(state); err != nil {
		t.Fatal(err)
	}

	engine, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	if !engine.dirty {
		t.Fatal("legacy amount was not marked for rewrite")
	}
	snapshot := engine.Snapshot()
	for name, value := range map[string]string{
		"gold": snapshot.Gold, "click power": snapshot.ClickPower,
		"click cost": snapshot.ClickCost, "CPS": snapshot.CPS,
	} {
		if len(value) > 50 {
			t.Fatalf("%s length = %d", name, len(value))
		}
	}
	for _, facility := range snapshot.Facilities {
		if len(facility.Cost) > 50 || len(facility.CPS) > 50 || len(facility.UnitCPS) > 50 {
			t.Fatalf("facility %d contains an unbounded amount", facility.ID)
		}
	}
	engine.persist()
	reloaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Gold) > 50 {
		t.Fatalf("persisted migrated gold length = %d", len(reloaded.Gold))
	}
}
