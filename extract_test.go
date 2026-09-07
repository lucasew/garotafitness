package garotafitness

import (
	"os"
	"path/filepath"
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

func TestExtractStoringSolid(t *testing.T) {
	t.Parallel()
	d, err := OpenDirDest(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("helloworld")
	s := solid{
		pipe: ParsePipeline("storing"),
		off:  0,
		csz:  10,
		files: []Member{
			{Path: "a.txt", Size: 5, Pipeline: ParsePipeline("storing")},
			{Path: "b.txt", Size: 5, Pipeline: ParsePipeline("storing")},
		},
	}
	if err := extractSolid(Extractor{Dest: d}, data, s); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(d.Root, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Fatalf("got %q", got)
	}
	got, err = os.ReadFile(filepath.Join(d.Root, "b.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "world" {
		t.Fatalf("got %q", got)
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
