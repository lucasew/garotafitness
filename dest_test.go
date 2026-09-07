package garotafitness

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMemberPathRejectsEscape(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for _, name := range []string{"../x", "/etc/passwd", ""} {
		if _, err := memberPath(root, name); err == nil {
			t.Fatalf("accepted %q", name)
		}
	}
}

func TestDirDestCreateAndOverwrite(t *testing.T) {
	t.Parallel()
	d, err := OpenDirDest(filepath.Join(t.TempDir(), "out"))
	if err != nil {
		t.Fatal(err)
	}
	w, err := d.Create("a/b.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(w, "one"); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	w, err = d.Create("a/b.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(w, "two"); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(d.Root, "a", "b.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "two" {
		t.Fatalf("got %q", got)
	}
	_, err = d.Create("../escape")
	if err == nil || !strings.Contains(err.Error(), "leaves dest") {
		t.Fatalf("got %v", err)
	}
}
