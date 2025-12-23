package main

import (
	"testing"
)

func TestNormalizeHost(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{":8080", "localhost:8080"},
		{"localhost:8080", "localhost:8080"},
		{"127.0.0.1:8080", "127.0.0.1:8080"},
		{"0.0.0.0:8080", "0.0.0.0:8080"},
		{"[::1]:8080", "[::1]:8080"},
		{"[::]:8080", "[::]:8080"},
		{"8080", "8080"},
		{"localhost", "localhost"},
		{"", ""},
	}

	for _, tt := range tests {
		if got := normalizeHost(tt.in); got != tt.want {
			t.Fatalf("normalizeHost(%q)=%q want=%q", tt.in, got, tt.want)
		}
	}
}
