package game

import (
	"math/rand"
	"testing"
)

func TestEngineDerivedCacheTracksMutations(t *testing.T) {
	e := &Engine{
		gold:       amount("100000000000000000000000000000000000000000000000000"),
		diamonds:   2,
		clickLevel: 12,
		facilities: defaultFacilities(),
		rng:        rand.New(rand.NewSource(1)),
	}
	e.facilities[0] = FacilityState{ID: FacilityDefs[0].ID, Owned: 7, Enhance: 3}
	e.facilities[20] = FacilityState{ID: FacilityDefs[20].ID, Owned: 2, Enhance: 1}
	e.initializeDerived()
	assertDerivedCache(t, e)

	e.doBuy(&Client{}, FacilityDefs[0].ID, 1)
	assertDerivedCache(t, e)

	e.doBuy(&Client{}, FacilityDefs[20].ID, 1)
	assertDerivedCache(t, e)

	e.doEnhance(&Client{}, FacilityDefs[0].ID, 1)
	assertDerivedCache(t, e)

	e.doUpgradeClick(&Client{}, 1)
	assertDerivedCache(t, e)
}

func assertDerivedCache(t *testing.T, e *Engine) {
	t.Helper()
	if e.clickPower.Cmp(ClickPower(e.clickLevel)) != 0 {
		t.Fatalf("cached click power = %s", e.clickPower.String())
	}
	if e.clickCost.Cmp(ClickUpgradeCost(e.clickLevel)) != 0 {
		t.Fatalf("cached click cost = %s", e.clickCost.String())
	}

	total := zeroAmount()
	for i, def := range FacilityDefs {
		st := e.facilities[i]
		derived := e.derived[i]
		wantCost := FacilityCost(def, st.Owned)
		wantUnit := FacilityUnitCPS(def, st.Enhance)
		wantCPS := FacilityCPS(def, st)
		if derived.cost.Cmp(wantCost) != 0 || derived.costText != wantCost.String() {
			t.Fatalf("facility %d cached cost is stale", def.ID)
		}
		if derived.unitCPS.Cmp(wantUnit) != 0 || derived.unitCPSText != wantUnit.String() {
			t.Fatalf("facility %d cached unit CPS is stale", def.ID)
		}
		if derived.cps.Cmp(wantCPS) != 0 || derived.cpsText != wantCPS.String() {
			t.Fatalf("facility %d cached CPS is stale", def.ID)
		}
		total.Add(total, wantCPS)
	}
	if e.totalCPS.Cmp(total) != 0 || e.totalCPSText != total.String() {
		t.Fatal("cached total CPS is stale")
	}
	if e.clickPowerText != e.clickPower.String() || e.clickCostText != e.clickCost.String() {
		t.Fatal("cached click strings are stale")
	}
}
