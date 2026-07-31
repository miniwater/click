package game

import "testing"

func TestHubLimitsConnectionsPerIP(t *testing.T) {
	h := NewHub(nil)
	for i := 0; i < maxConnectionsPerIP; i++ {
		if !h.Admit("203.0.113.1") {
			t.Fatalf("connection %d was unexpectedly rejected", i)
		}
	}
	if h.Admit("203.0.113.1") {
		t.Fatal("connection above per-IP limit was admitted")
	}
	h.Release("203.0.113.1")
	if !h.Admit("203.0.113.1") {
		t.Fatal("connection slot was not released")
	}
}
