package garotafitness

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/test"
	"github.com/lucasew/garotafitness/internal/corpus"
	"github.com/stretchr/testify/require"
)

// TestExtractFarmingSimulator25 is the FS25 pair of the RimWorld and
// Songs of Conquest installed-MD5 extracts. Extract verifies the manifest
// before writing the tree; this re-reads every installed file.
func TestExtractFarmingSimulator25(t *testing.T) {
	if os.Getenv("GAROTAFITNESS_FULL_EXTRACT") == "" {
		t.Skip("set GAROTAFITNESS_FULL_EXTRACT=all or required for full corpus extraction")
	}
	out := os.Getenv("GAROTAFITNESS_FS25_DEST")
	if out == "" {
		t.Fatal("set GAROTAFITNESS_FS25_DEST")
	}
	src := corpus.OpenEnv(t, fs25Corpus)
	require.NoError(t, os.MkdirAll(out, 0o755))
	dst, err := OpenDirDest(out)
	require.NoError(t, err)
	test.CloseOnCleanup(t, dst)
	require.NoError(t, (Extractor{Source: src, Dest: dst}).Extract(t.Context()))
	manifest, err := lewpath.New("_Redist/fitgirl.md5").ReadFile(dst)
	require.NoError(t, err)
	count := 0
	for line := range strings.SplitSeq(strings.TrimSpace(string(manifest)), "\n") {
		want, name, ok := strings.Cut(strings.TrimSuffix(line, "\r"), " *..\\")
		require.True(t, ok, "invalid manifest line %q", line)
		f, err := lewpath.New(strings.ReplaceAll(name, "\\", "/")).Open(dst)
		require.NoError(t, err)
		h := md5.New()
		_, err = io.Copy(h, f)
		require.NoError(t, f.Close())
		require.NoError(t, err)
		require.Equal(t, want, fmt.Sprintf("%x", h.Sum(nil)), name)
		count++
	}
	require.Equal(t, 39796, count)
}
