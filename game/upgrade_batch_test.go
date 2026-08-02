package game

import (
	"encoding/json"
	"math/rand"
	"testing"
)

func TestUpgradeActionsProcessBatchWithOneBroadcast(t *testing.T) {
	e := &Engine{
		gold:       amount("1e100"),
		diamonds:   3,
		facilities: defaultFacilities(),
		rng:        rand.New(rand.NewSource(1)),
	}
	e.facilities[0].Owned = 1
	e.initializeDerived()
	hub := NewHub(e)
	e.SetHub(hub)
	client := &Client{Name: "tester", Color: "red"}

	e.doBuy(client, FacilityDefs[0].ID, 5)
	if e.facilities[0].Owned != 6 {
		t.Fatalf("owned = %d, want 6", e.facilities[0].Owned)
	}
	assertBroadcastCount(t, hub, 2)

	e.doUpgradeClick(client, 5)
	if e.clickLevel != 5 {
		t.Fatalf("click level = %d, want 5", e.clickLevel)
	}
	assertBroadcastCount(t, hub, 2)

	e.doEnhance(client, FacilityDefs[0].ID, 20)
	if e.facilities[0].Enhance != 3 || e.diamonds != 0 {
		t.Fatalf("enhance = %d, diamonds = %d, want 3 and 0", e.facilities[0].Enhance, e.diamonds)
	}
	assertBroadcastCount(t, hub, 2)
}

func assertBroadcastCount(t *testing.T, hub *Hub, want int) {
	t.Helper()
	for i := 0; i < want; i++ {
		select {
		case data := <-hub.broadcast:
			var message struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(data, &message); err != nil {
				t.Fatal(err)
			}
			if message.Type != "notify" && message.Type != "state" {
				t.Fatalf("unexpected message type %q", message.Type)
			}
		default:
			t.Fatalf("broadcast count = %d, want %d", i, want)
		}
	}
	select {
	case <-hub.broadcast:
		t.Fatalf("broadcast count exceeds %d", want)
	default:
	}
}
