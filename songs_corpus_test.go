package garotafitness

import (
	"bytes"
	"hash/crc32"
	"io"
	"io/fs"
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
	require.Contains(t, info.Encoders, "rzs")
	require.Contains(t, info.Encoders, "xt3u")
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

func TestSongsOfConquestPipelines(t *testing.T) {
	src := corpus.OpenEnv(t, socCorpus)
	want := map[string]Algo{
		"fg-01.bin": AlgoRZS,
		"fg-02.bin": AlgoRZS,
		"fg-03.bin": AlgoMagic2,
		"fg-04.bin": AlgoRZW,
	}
	for name, last := range want {
		t.Run(name, func(t *testing.T) {
			data, err := lewpath.New(name).ReadFile(src)
			require.NoError(t, err)
			v, err := parseVolume(name, data)
			require.NoError(t, err)
			found := false
			for _, m := range v.Members {
				if m.Dir || len(m.Pipeline) == 0 {
					continue
				}
				found = true
				if m.Pipeline.Last().Algo != last {
					t.Fatalf("%s last %s want %s (%s)", m.Path, m.Pipeline.Last().Algo, last, m.Pipeline)
				}
			}
			require.True(t, found, "no file members")
		})
	}
}

func TestExtractSongsOfConquestRZS(t *testing.T) {
	src := corpus.OpenEnv(t, socCorpus)
	name := "fg-02.bin"
	dst := &reconstruction{files: map[string][]byte{}, dirs: map[string]fs.FileMode{}}
	require.NoError(t, extractVolume(t.Context(), Extractor{Source: src, Dest: dst}, Volume{Name: name}))
	data, err := lewpath.New(name).ReadFile(src)
	require.NoError(t, err)
	v, err := parseVolume(name, data)
	require.NoError(t, err)
	checked := 0
	for _, m := range v.Members {
		if m.Dir {
			continue
		}
		b, err := dst.require(m.Path)
		require.NoError(t, err, m.Path)
		require.Equal(t, m.Size, uint64(len(b)), m.Path)
		table := m.crcTable
		if table == nil {
			table = crc32.IEEETable
		}
		require.Equal(t, m.CRC, crc32.Checksum(b, table), m.Path)
		checked++
	}
	require.Greater(t, checked, 0)
}
