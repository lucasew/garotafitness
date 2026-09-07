package garotafitness

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestExtractNilDeps(t *testing.T) {
	t.Parallel()
	var e Extractor
	if err := e.Extract(t.Context()); err == nil {
		t.Fatal("want nil source")
	}
	e.Source = fstest.MapFS{}
	if err := e.Extract(t.Context()); err == nil {
		t.Fatal("want nil dest")
	}
}

func TestExtractNoVolume(t *testing.T) {
	t.Parallel()
	d, err := OpenDirDest(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	e := Extractor{Source: fstest.MapFS{"readme.txt": {Data: []byte("x")}}, Dest: d}
	if err := e.Extract(t.Context()); err == nil || !strings.Contains(err.Error(), "no fg-*.bin") {
		t.Fatalf("got %v", err)
	}
}

func TestExtractUnknownEncoder(t *testing.T) {
	t.Parallel()
	d, err := OpenDirDest(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	e := Extractor{
		Source: fstest.MapFS{"fg-01.bin": {Data: []byte(arcMagic + "storing\x00SREP")}},
		Dest:   d,
	}
	err = e.Extract(t.Context())
	if err == nil {
		t.Fatal("want error")
	}
}
