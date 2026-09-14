package lzma

import (
	"bytes"
	"io"
	"testing"

	ulzma "github.com/ulikunitz/xz/lzma"
)

func TestNewReaderRoundTrip(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w, err := ulzma.NewWriter(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(w, "hello lzma"); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := NewReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello lzma" {
		t.Fatalf("got %q", got)
	}
}
