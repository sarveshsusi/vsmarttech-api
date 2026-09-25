package service

import "testing"

func TestSanitizeOtherEngineers(t *testing.T) {
	got, err := sanitizeOtherEngineers("  Ajithkumar ,  Guest  tech  ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Ajithkumar, Guest tech" {
		t.Fatalf("got %q", got)
	}
	empty, err := sanitizeOtherEngineers("  ,  ")
	if err != nil || empty != "" {
		t.Fatalf("empty got %q err=%v", empty, err)
	}
}
