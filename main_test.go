package main

import (
	"net/http/httptest"
	"testing"
)

func TestWebSocketOriginMustMatchHost(t *testing.T) {
	tests := []struct {
		origin string
		want   bool
	}{
		{"", true},
		{"https://game.example", true},
		{"https://GAME.EXAMPLE", true},
		{"https://evil.example", false},
		{"null", false},
	}
	for _, tt := range tests {
		r := httptest.NewRequest("GET", "http://game.example/ws", nil)
		if tt.origin != "" {
			r.Header.Set("Origin", tt.origin)
		}
		if got := upgrader.CheckOrigin(r); got != tt.want {
			t.Errorf("origin %q: got %v, want %v", tt.origin, got, tt.want)
		}
	}
}
