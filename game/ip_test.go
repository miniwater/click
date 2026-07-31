package game

import "testing"

func TestClientIPDoesNotTrustForwardedHeadersByDefault(t *testing.T) {
	if got := ClientIP("10.0.0.1:1234", "203.0.113.10", "203.0.113.11", false); got != "10.0.0.1" {
		t.Fatalf("ClientIP = %q, want remote address", got)
	}
	if got := ClientIP("10.0.0.1:1234", "203.0.113.10", "", true); got != "203.0.113.10" {
		t.Fatalf("trusted ClientIP = %q, want forwarded address", got)
	}
}
