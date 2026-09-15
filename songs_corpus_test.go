package garotafitness

import (
	"bytes"
	"io"
	"strings"
	"testing"

	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lucasew/garotafitness/internal/corpus"
	"github.com/lucasew/garotafitness/setupdata"
	"github.com/stretchr/testify/require"
)

const socCorpus = "GAROTAFITNESS_CORPUS_SOC"

func TestSongsOfConquestSetup(t *testing.T) {
	f := corpus.FileEnv(t, socCorpus, "setup.exe")
	info, err := setupdata.Scan(f)
	require.NoError(t, err)
	require.Equal(t, "{app}\\_Redist\\fitgirl.md5", info.ManifestPath)
	require.Equal(t, 674, strings.Count(info.InstalledMD5, "\n"))
	require.Contains(t, info.Encoders, "pref")
	require.Len(t, info.Operations, 12)
}

func TestSongsOfConquestPrefHeader(t *testing.T) {
	src := corpus.OpenEnv(t, socCorpus)
	data, err := lewpath.New("fg-optional-bonus-content.bin").ReadFile(src)
	require.NoError(t, err)
	v, err := parseVolume("fg-optional-bonus-content.bin", data)
	require.NoError(t, err)
	for _, s := range groupSolids(v.Members) {
		if s.pipe.String() != "pref+srep:m3yf+magic2" {
			continue
		}
		r, err := Decode(bytes.NewReader(data[s.off:s.off+int64(s.csz)]), s.pipe[len(s.pipe)-1])
		require.NoError(t, err)
		r2, err := Decode(r, s.pipe[len(s.pipe)-2])
		require.NoError(t, err)
		var head [16]byte
		n, err := io.ReadFull(r2, head[:])
		require.NoError(t, r2.Close())
		require.NoError(t, r.Close())
		require.GreaterOrEqual(t, n, 7)
		require.Equal(t, "PCF", string(head[:3]))
		require.Equal(t, []byte{0, 4, 8, 0}, head[3:7])
		return
	}
	t.Fatal("no pref solid")
}
