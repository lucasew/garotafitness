package main

import "testing"

func TestParseArgs(t *testing.T) {
	t.Parallel()
	_, _, err := parseArgs(nil)
	if err == nil {
		t.Fatal("want error for empty argv")
	}
	_, _, err = parseArgs([]string{"extract", "only"})
	if err == nil {
		t.Fatal("want error for one operand")
	}
	src, dst, err := parseArgs([]string{"extract", "/src", "/dst"})
	if err != nil {
		t.Fatal(err)
	}
	if src != "/src" || dst != "/dst" {
		t.Fatalf("got %q %q", src, dst)
	}
}
