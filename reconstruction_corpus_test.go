package garotafitness

import (
	"crypto/md5"
	"fmt"
	"hash/crc32"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// This checks the public extraction flow and then reopens every installed file.
// It is opt-in because reconstructing the full corpus takes several minutes.
func TestExtractInstalledRimWorld(t *testing.T) {
	mode := os.Getenv("GAROTAFITNESS_FULL_EXTRACT")
	if mode == "" {
		t.Skip("set GAROTAFITNESS_FULL_EXTRACT=all or required for full corpus extraction")
	}
	if mode != "all" && mode != "required" {
		t.Fatal("GAROTAFITNESS_FULL_EXTRACT must be all or required")
	}
	source := os.DirFS(rimworldCorpus)
	if _, err := fs.Stat(source, "setup.exe"); err != nil {
		t.Skip("corpus not mounted")
	}
	if mode == "required" {
		dir := t.TempDir()
		for _, name := range []string{"setup.exe", "MD5", "fg-01.bin", "fg-02.bin", "fg-03.bin", "fg-04.bin", "fg-05.bin", "fg-06.bin"} {
			if err := os.Symlink(filepath.Join(rimworldCorpus, name), filepath.Join(dir, name)); err != nil {
				t.Fatal(err)
			}
		}
		source = os.DirFS(dir)
	}
	dst, err := OpenDirDest(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := (Extractor{Source: source, Dest: dst}).Extract(t.Context()); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(dst.Root, "_Redist/fitgirl.md5"))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, line := range strings.Split(strings.TrimSpace(string(manifest)), "\n") {
		want, name, ok := strings.Cut(strings.TrimSuffix(line, "\r"), " *..\\")
		if !ok {
			t.Fatalf("invalid manifest line %q", line)
		}
		f, err := os.Open(filepath.Join(dst.Root, filepath.FromSlash(strings.ReplaceAll(name, "\\", "/"))))
		if err != nil {
			t.Fatal(err)
		}
		h := md5.New()
		_, err = io.Copy(h, f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if got := fmt.Sprintf("%x", h.Sum(nil)); got != want {
			t.Fatalf("%s: %s want %s", name, got, want)
		}
		count++
	}
	if count != 1712 {
		t.Fatalf("verified %d files, want 1712", count)
	}
	tracks, err := filepath.Glob(filepath.Join(dst.Root, "Soundtrack/*.mp3"))
	if err != nil {
		t.Fatal(err)
	}
	want := 0
	if mode == "all" {
		want = 31
	}
	if len(tracks) != want {
		t.Fatalf("wrote %d tracks, want %d", len(tracks), want)
	}
	if mode == "all" {
		name := "fg-optional-bonus-soundtrack.bin"
		data, err := fs.ReadFile(source, name)
		if err != nil {
			t.Fatal(err)
		}
		volume, err := parseVolume(name, data)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range volume.Members {
			if m.Dir {
				continue
			}
			f, err := os.Open(filepath.Join(dst.Root, filepath.FromSlash(m.Path)))
			if err != nil {
				t.Fatal(err)
			}
			h := crc32.New(m.crcTable)
			size, err := io.Copy(h, f)
			f.Close()
			if err != nil || uint64(size) != m.Size || h.Sum32() != m.CRC {
				t.Fatalf("%s: size %d, CRC %08x, error %v; want %d, %08x", m.Path, size, h.Sum32(), err, m.Size, m.CRC)
			}
		}
	}
	for _, name := range []string{"inner.fgpack", "rimworld.x3", "temp", "work", "mover"} {
		if _, err := os.Stat(filepath.Join(dst.Root, name)); !os.IsNotExist(err) {
			t.Fatalf("intermediate %s remains: %v", name, err)
		}
	}
}
