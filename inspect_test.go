//go:build reconstruction

package garotafitness

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/lucasew/garotafitness/fgpack"
	"github.com/lucasew/garotafitness/fsb"
	"github.com/lucasew/garotafitness/x2"
	"github.com/lucasew/garotafitness/x3"
	"github.com/lucasew/garotafitness/x5"
	"github.com/lucasew/garotafitness/xdelta"
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

func TestReconstructInner(t *testing.T) {
	in, err := os.Open("/tmp/gf-extract/outer/fg-01.bin/inner.fgpack")
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	var out bytes.Buffer
	if err := fsb.Remux(t.Context(), &out, in); err != nil {
		t.Fatal(err)
	}
	t.Logf("FSB size %d", out.Len())
	diff, err := os.ReadFile("/tmp/gf-extract/outer/fg-04.bin/inner.fgpack.x5")
	if err != nil {
		t.Fatal(err)
	}
	patched, err := x5.Apply(t.Context(), out.Bytes(), diff)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("inner size %d", len(patched))
	if err := os.WriteFile("/tmp/gf-extract/inner-patched.fgpack", patched, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestReconstructBundles(t *testing.T) {
	for i := 1; i <= 4; i++ {
		name := fmt.Sprint(i)
		if selected := os.Getenv("RIMWORLD_BUNDLE"); selected != "" && selected != name {
			continue
		}
		old, err := os.ReadFile("/tmp/gf-extract/outer/fg-02.bin/temp/" + name + ".fgu")
		if err != nil {
			t.Fatal(err)
		}
		diff, err := os.ReadFile("/tmp/gf-extract/inner/temp/" + name + ".fgu.x2")
		if err != nil {
			t.Fatal(err)
		}
		raw, err := x2.Apply(old, diff)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s: x2 %d", name, len(raw))
		encoded, err := fgpack.Encode(t.Context(), raw)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s: LZMA %d", name, len(encoded))
		diff, err = os.ReadFile("/tmp/gf-extract/outer/fg-02.bin/temp/" + name + ".bundle.x")
		if err != nil {
			t.Fatal(err)
		}
		bundle, err := xdelta.Apply(t.Context(), encoded, diff)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s: bundle %d", name, len(bundle))
		if err := os.WriteFile("/tmp/gf-extract/"+name+".bundle", bundle, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReconstructUpdate(t *testing.T) {
	data, err := os.ReadFile("/tmp/gf-extract/outer/fg-06.bin/rimworld.x3")
	if err != nil {
		t.Fatal(err)
	}
	records, err := x3.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, rec := range records {
		var old []byte
		for _, dir := range []string{"update", "outer/fg-06.bin", "outer/fg-03.bin", "outer/fg-02.bin", "inner"} {
			old, err = os.ReadFile(filepath.Join("/tmp/gf-extract", dir, rec.Source))
			if err == nil {
				break
			}
		}
		if err != nil {
			t.Fatal(err)
		}
		out, err := rec.Apply(t.Context(), old)
		if err != nil {
			t.Fatal(err)
		}
		name := filepath.Join("/tmp/gf-extract/update", rec.Target)
		if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, out, 0600); err != nil {
			t.Fatal(err)
		}
		t.Logf("updated %s (%d bytes)", rec.Target, len(out))
	}
}
