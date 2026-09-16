package garotafitness

import (
	"fmt"
	"hash/crc32"
	"slices"
	"testing"
	"testing/fstest"

	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestExtractNilDeps(t *testing.T) {
	t.Parallel()
	var e Extractor
	require.Error(t, e.Extract(t.Context()))
	e.Source = fstest.MapFS{}
	require.Error(t, e.Extract(t.Context()))
}

func TestScanSetup(t *testing.T) {
	t.Parallel()
	names, err := scanSetup(fstest.MapFS{})
	require.NoError(t, err)
	require.Nil(t, names)
	src := fstest.MapFS{
		"setup.exe": {Data: []byte("[External compressor:srep]\r\nunpackcmd = srep d\r\n")},
	}
	names, err = scanSetup(src)
	require.NoError(t, err)
	require.Contains(t, names, "srep")
}

func TestExtractNoVolume(t *testing.T) {
	t.Parallel()
	d, err := OpenDirDest(t.TempDir())
	require.NoError(t, err)
	test.CloseOnCleanup(t, d)
	e := Extractor{Source: fstest.MapFS{"readme.txt": {Data: []byte("x")}}, Dest: d}
	err = e.Extract(t.Context())
	require.ErrorContains(t, err, "no fg-*.bin")
}

func TestExtractStoringSolid(t *testing.T) {
	t.Parallel()
	d, err := OpenDirDest(t.TempDir())
	require.NoError(t, err)
	test.CloseOnCleanup(t, d)
	data := []byte("helloworld")
	s := solid{
		pipe: ParsePipeline("storing"),
		off:  0,
		csz:  10,
		files: []Member{
			{Path: "a.txt", Size: 5, Pipeline: ParsePipeline("storing")},
			{Path: "b.txt", Size: 5, Pipeline: ParsePipeline("storing")},
		},
	}
	require.NoError(t, extractSolid(t.Context(), Extractor{Dest: d}, data, s))
	got, err := lewpath.New("a.txt").ReadFile(d)
	require.NoError(t, err)
	require.Equal(t, "hello", string(got))
	got, err = lewpath.New("b.txt").ReadFile(d)
	require.NoError(t, err)
	require.Equal(t, "world", string(got))
}

func TestExtractStackedStoring(t *testing.T) {
	t.Parallel()
	d, err := OpenDirDest(t.TempDir())
	require.NoError(t, err)
	test.CloseOnCleanup(t, d)
	data := []byte("hello")
	s := solid{
		pipe: ParsePipeline("storing+storing"),
		off:  0,
		csz:  5,
		files: []Member{
			{Path: "a.txt", Size: 5, Pipeline: ParsePipeline("storing+storing")},
		},
	}
	require.NoError(t, extractSolid(t.Context(), Extractor{Dest: d}, data, s))
	got, err := lewpath.New("a.txt").ReadFile(d)
	require.NoError(t, err)
	require.Equal(t, "hello", string(got))
}

func TestExtractCRCMismatch(t *testing.T) {
	t.Parallel()
	d, err := OpenDirDest(t.TempDir())
	require.NoError(t, err)
	test.CloseOnCleanup(t, d)
	data := []byte("hello")
	s := solid{
		pipe: ParsePipeline("storing"),
		off:  0,
		csz:  5,
		files: []Member{
			{Path: "a.txt", Size: 5, CRC: crc32.ChecksumIEEE(data) ^ 1, Pipeline: ParsePipeline("storing")},
		},
	}
	require.ErrorContains(t, extractSolid(t.Context(), Extractor{Dest: d}, data, s), "crc")
}

func TestExtractIndependentSolids(t *testing.T) {
	t.Parallel()
	d, err := OpenDirDest(t.TempDir())
	require.NoError(t, err)
	test.CloseOnCleanup(t, d)
	data := []byte("helloworld")
	solids := []solid{
		{pipe: ParsePipeline("storing"), off: 0, csz: 5, files: []Member{{Path: "a.txt", Size: 5, Pipeline: ParsePipeline("storing")}}},
		{pipe: ParsePipeline("storing"), off: 5, csz: 5, files: []Member{{Path: "b.txt", Size: 5, Pipeline: ParsePipeline("storing")}}},
	}
	require.NoError(t, extractSolids(t.Context(), Extractor{Dest: d}, data, slices.Values(solids)))
	got, err := lewpath.New("a.txt").ReadFile(d)
	require.NoError(t, err)
	require.Equal(t, "hello", string(got))
	got, err = lewpath.New("b.txt").ReadFile(d)
	require.NoError(t, err)
	require.Equal(t, "world", string(got))
}

func TestExtractIndependentSolidsStaging(t *testing.T) {
	t.Parallel()
	s := newStaging()
	data := make([]byte, 0, 16*5)
	solids := make([]solid, 16)
	want := map[string][]byte{}
	for i := range solids {
		chunk := []byte(fmt.Sprintf("file%02d", i))[:5]
		data = append(data, chunk...)
		name := fmt.Sprintf("f%02d.txt", i)
		solids[i] = solid{
			pipe:  ParsePipeline("storing"),
			off:   int64(i * 5),
			csz:   5,
			files: []Member{{Path: name, Size: 5, Pipeline: ParsePipeline("storing")}},
		}
		want[name] = append([]byte(nil), chunk...)
	}
	require.NoError(t, extractSolids(t.Context(), Extractor{Dest: s}, data, slices.Values(solids)))
	require.Equal(t, want, s.files)
}

func TestExtractUnknownEncoder(t *testing.T) {
	t.Parallel()
	d, err := OpenDirDest(t.TempDir())
	require.NoError(t, err)
	test.CloseOnCleanup(t, d)
	e := Extractor{
		Source: fstest.MapFS{"fg-01.bin": {Data: []byte(arcMagic + "storing\x00SREP")}},
		Dest:   d,
	}
	require.Error(t, e.Extract(t.Context()))
}
