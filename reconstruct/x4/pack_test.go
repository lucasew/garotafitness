package x4

import (
	"archive/zip"
	"bytes"
	"io"
	"testing"
)

func TestPackZip(t *testing.T) {
	t.Parallel()
	out, err := Pack([]File{{Name: "n/a.txt", Data: []byte("hi")}}, "02", "01")
	if err != nil {
		t.Fatal(err)
	}
	r, err := zip.NewReader(bytes.NewReader(out), int64(len(out)))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.File) != 1 || r.File[0].Name != "n/a.txt" {
		t.Fatalf("files %+v", r.File)
	}
	f, err := r.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(f)
	f.Close()
	if err != nil || string(b) != "hi" {
		t.Fatalf("got %q", b)
	}
}
func TestPackRejectsVersion(t *testing.T) {
	t.Parallel()
	if _, err := Pack(nil, "01", "01"); err == nil {
		t.Fatal("accepted")
	}
}
