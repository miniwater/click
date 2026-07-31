package game

import "testing"

func TestClientIPPrefersForwardedHeaders(t *testing.T) {
	tests := []struct {
		remote string
		xff    string
		xri    string
		want   string
	}{
		{"127.0.0.1:1234", "203.0.113.10", "", "203.0.113.10"},
		{"[::1]:1234", "2001:db8::10", "", "2001:db8::10"},
		{"127.0.0.1:1234", "", "203.0.113.11", "203.0.113.11"},
		{"127.0.0.1:1234", "", "", "127.0.0.1"},
	}
	for _, tt := range tests {
		if got := ClientIP(tt.remote, tt.xff, tt.xri); got != tt.want {
			t.Errorf("ClientIP(%q) = %q, want %q", tt.remote, got, tt.want)
		}
	}
}
