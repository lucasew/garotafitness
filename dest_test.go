package garotafitness

import (
	"io"
	"testing"

	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestMemberPathRejectsEscape(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"../x", "/etc/passwd", ""} {
		_, err := memberName(name)
		require.Error(t, err, "accepted %q", name)
	}
}

func TestDirDestCreateAndOverwrite(t *testing.T) {
	t.Parallel()
	d, err := OpenDirDest(t.TempDir())
	require.NoError(t, err)
	test.CloseOnCleanup(t, d)
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
	got, err := lewpath.New("a/b.txt").ReadFile(d)
	require.NoError(t, err)
	require.Equal(t, "two", string(got))
	_, err = d.Create("../escape")
	require.ErrorContains(t, err, "leaves dest")
}
