//go:build reconstruction

package garotafitness

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInspectRimWorld(t *testing.T) {
	for _, name := range []string{"fg-01.bin", "fg-02.bin", "fg-03.bin", "fg-04.bin", "fg-05.bin", "fg-06.bin", "fg-optional-bonus-soundtrack.bin"} {
		data, err := os.ReadFile(filepath.Join(rimworldCorpus, name))
		if err != nil {
			t.Fatal(err)
		}
		v, err := parseVolume(name, data)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range v.Members {
			t.Logf("%s: %s size %d crc %08x pipe %s", name, m.Path, m.Size, m.CRC, m.Pipeline)
		}
		if name == "fg-optional-bonus-soundtrack.bin" {
			continue
		}
		dst, err := OpenDirDest(filepath.Join("/tmp/gf-extract/outer", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := extractVolume(t.Context(), Extractor{Source: os.DirFS(rimworldCorpus), Dest: dst}, v); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInspectInner(t *testing.T) {
	data, err := os.ReadFile("/tmp/gf-extract/inner-patched.fgpack")
	if err != nil {
		t.Fatal(err)
	}
	v, err := parseVolume("inner-patched.fgpack", data)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range v.Members {
		t.Logf("%s size %d crc %08x pipe %s", m.Path, m.Size, m.CRC, m.Pipeline)
	}
	dst, err := OpenDirDest("/tmp/gf-extract/inner")
	if err != nil {
		t.Fatal(err)
	}
	if err := extractVolume(t.Context(), Extractor{Source: os.DirFS("/tmp/gf-extract"), Dest: dst}, v); err != nil {
		t.Fatal(err)
	}
}
