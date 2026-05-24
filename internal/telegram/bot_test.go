package telegram

import "testing"

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"this is a very long string", 10, "this is a ..."},
		{"exact", 5, "exact"},
		{"café 🍺", 4, "café..."},
	}

	for _, tt := range tests {
		got := truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestParseUserID(t *testing.T) {
	id, err := ParseUserID("123456789")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 123456789 {
		t.Errorf("expected 123456789, got %d", id)
	}

	_, err = ParseUserID("invalid")
	if err == nil {
		t.Error("expected error for invalid user ID")
	}
}
