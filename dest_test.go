package garotafitness

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMemberPathRejectsEscape(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for _, name := range []string{"../x", "/etc/passwd", ""} {
		_, err := memberPath(root, name)
		require.Error(t, err, "accepted %q", name)
	}
}

func TestDirDestCreateAndOverwrite(t *testing.T) {
	t.Parallel()
	d, err := OpenDirDest(filepath.Join(t.TempDir(), "out"))
	require.NoError(t, err)
	w, err := d.Create("a/b.txt")
	require.NoError(t, err)
	_, err = io.WriteString(w, "one")
	require.NoError(t, err)
	require.NoError(t, w.Close())
	w, err = d.Create("a/b.txt")
	require.NoError(t, err)
	_, err = io.WriteString(w, "two")
	require.NoError(t, err)
	require.NoError(t, w.Close())
	got, err := os.ReadFile(filepath.Join(d.Root, "a", "b.txt"))
	require.NoError(t, err)
	require.Equal(t, "two", string(got))
	_, err = d.Create("../escape")
	require.ErrorContains(t, err, "leaves dest")
}
