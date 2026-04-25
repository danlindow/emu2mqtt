package main

import "testing"

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "b", "c"); got != "b" {
		t.Errorf("got %q, want b", got)
	}
	if got := firstNonEmpty("", "", ""); got != "" {
		t.Errorf("got %q, want empty string", got)
	}
	if got := firstNonEmpty("a", "b"); got != "a" {
		t.Errorf("got %q, want a", got)
	}
	if got := firstNonEmpty(); got != "" {
		t.Errorf("got %q, want empty string for no args", got)
	}
}
