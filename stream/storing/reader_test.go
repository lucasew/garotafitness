package storing

import (
	"io"
	"strings"
	"testing"
)

func TestNewReader(t *testing.T) {
	t.Parallel()
	r, err := NewReader(strings.NewReader("raw"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "raw" {
		t.Fatalf("got %q", got)
	}
	if _, err := NewReader(nil); err == nil {
		t.Fatal("want nil reader error")
	}
}
