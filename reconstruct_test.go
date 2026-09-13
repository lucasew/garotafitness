package garotafitness

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestInstalledChecksums(t *testing.T) {
	s := &reconstruction{files: map[string][]byte{"Data/a.txt": []byte("abc")}}
	manifest := "900150983cd24fb0d6963f7d28e17f72 *..\\Data\\a.txt\r\n"
	if err := s.verifyInstalled(t.Context(), manifest, "_Redist"); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{
		"", "not a checksum\n", strings.Replace(manifest, "9001", "0001", 1),
		strings.Replace(manifest, "a.txt", "missing", 1),
		strings.Replace(manifest, "Data\\a.txt", "..\\escape", 1),
		strings.Replace(manifest, "Data\\a.txt", "C:\\escape", 1),
	} {
		if err := s.verifyInstalled(t.Context(), bad, "_Redist"); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := s.verifyInstalled(ctx, manifest, "_Redist"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestStagingRejectsEscapes(t *testing.T) {
	s := &reconstruction{files: map[string][]byte{}}
	for _, name := range []string{"../escape", "/escape"} {
		if _, err := s.Create(name); err == nil {
			t.Fatalf("accepted %q", name)
		}
	}
	if len(s.files) != 0 {
		t.Fatal("wrote an invalid member")
	}
}
