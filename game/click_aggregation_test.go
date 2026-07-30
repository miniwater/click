package game

import (
	"encoding/json"
	"math/rand"
	"testing"
)

func TestClickResultsAreAggregatedUntilFlush(t *testing.T) {
	e := &Engine{
		gold:       zeroAmount(),
		facilities: defaultFacilities(),
		rng:        rand.New(rand.NewSource(1)),
	}
	e.initializeDerived()
	hub := NewHub(e)
	e.SetHub(hub)
	first := &Client{Name: "first", Color: "red"}
	second := &Client{Name: "second", Color: "blue"}

	e.doClick(first, 2)
	e.doClick(first, 3)
	e.doClick(second, 4)
	select {
	case <-hub.broadcast:
		t.Fatal("click result was broadcast before the aggregation flush")
	default:
	}

	e.flushClickResults()
	var message struct {
		Type    string        `json:"type"`
		Results []clickResult `json:"results"`
		State   PublicState   `json:"state"`
	}
	select {
	case data := <-hub.broadcast:
		if err := json.Unmarshal(data, &message); err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatal("aggregated click result was not broadcast")
	}
	if message.Type != "click_result" || len(message.Results) != 2 {
		t.Fatalf("unexpected aggregation message: %#v", message)
	}
	results := make(map[string]clickResult, len(message.Results))
	for _, result := range message.Results {
		results[result.Name] = result
	}
	if results["first"].Gain != "5e0" || results["second"].Gain != "4e0" {
		t.Fatalf("aggregated gains = %#v", results)
	}
	if message.State.Gold != "9e0" {
		t.Fatalf("state gold = %q, want 9e0", message.State.Gold)
	}

	e.flushClickResults()
	select {
	case <-hub.broadcast:
		t.Fatal("empty aggregation window was broadcast")
	default:
	}
}
