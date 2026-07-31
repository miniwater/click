package game

import "testing"

func TestAllowActionLimitsChatAndActions(t *testing.T) {
	h := NewHub(nil)
	c := &Client{hub: h, IP: "203.0.113.1"}
	for i := 0; i < 40; i++ {
		if !allowAction(c, "click", 1) {
			t.Fatalf("action %d was unexpectedly limited", i)
		}
	}
	if allowAction(c, "click", 1) {
		t.Fatal("41st action was not limited")
	}

	c = &Client{hub: NewHub(nil), IP: "203.0.113.1"}
	for i := 0; i < 5; i++ {
		if !allowAction(c, "chat", 0) {
			t.Fatalf("chat %d was unexpectedly limited", i)
		}
	}
	if allowAction(c, "chat", 0) {
		t.Fatal("6th chat was not limited")
	}

	c = &Client{hub: NewHub(nil), IP: "203.0.113.1"}
	if !allowAction(c, "click", 20) || !allowAction(c, "click", 20) {
		t.Fatal("two batched clicks should fit the action budget")
	}
	if allowAction(c, "click", 20) {
		t.Fatal("third batched click exceeded the action budget")
	}
}

func TestAllowActionIsSharedByIPAcrossConnections(t *testing.T) {
	h := NewHub(nil)
	first := &Client{hub: h, IP: "203.0.113.1"}
	second := &Client{hub: h, IP: "203.0.113.1"}
	if !allowAction(first, "click", 20) || !allowAction(second, "click", 20) {
		t.Fatal("shared IP budget rejected valid actions")
	}
	if allowAction(&Client{hub: h, IP: "203.0.113.1"}, "click", 1) {
		t.Fatal("new connection bypassed shared IP budget")
	}
	if !allowAction(&Client{hub: h, IP: "203.0.113.2"}, "click", 1) {
		t.Fatal("one IP consumed another IP's budget")
	}
}
