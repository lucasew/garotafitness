package jpgrec

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed testdata/red.pjg
var redPJG []byte

//go:embed testdata/red.jpg
var redJPG []byte

//go:embed testdata/src32.pjg
var src32PJG []byte

//go:embed testdata/src32.jpg
var src32JPG []byte

//go:embed testdata/red.brn
var redBrunsli []byte

//go:embed testdata/src32.brn
var src32Brunsli []byte

func TestRestorePackJPGRed(t *testing.T) {
	got, err := RestorePackJPG(redPJG)
	require.NoError(t, err)
	require.Equal(t, redJPG, got)
}

func TestRestorePackJPGSrc32(t *testing.T) {
	got, err := RestorePackJPG(src32PJG)
	require.NoError(t, err)
	require.Equal(t, src32JPG, got)
}

func TestRestoreBrunsliRed(t *testing.T) {
	got, err := RestoreBrunsli(redBrunsli, false)
	require.NoError(t, err)
	require.Equal(t, redJPG, got)
}

func TestRestoreBrunsliSrc32(t *testing.T) {
	got, err := RestoreBrunsli(src32Brunsli, false)
	require.NoError(t, err)
	require.Equal(t, src32JPG, got)
}

func TestStripMotionDHTSplice(t *testing.T) {
	// Crafted buffer only locks the splice math. The official writer was not
	// observed emitting a motion-JPEG DHT on the oracle JPEGs.
	const gap = mjpgDHTLen
	prefix := bytes.Repeat([]byte{0x11}, 10)
	mid := bytes.Repeat([]byte{0x22}, gap)
	// FF is immediately before DA, so ffda_pos lands on DA.
	tail := append([]byte{0xFF, 0xDA, 0x33, 0x44}, bytes.Repeat([]byte{0x55}, 20)...)
	jpeg := append(append(prefix, mid...), tail...)
	// ffda_pos = len(prefix)+gap+1 (the DA byte). cut = ffda_pos-1-gap = len(prefix).
	// Output length is recompressedLen: prefix plus the bytes that begin at FF.
	recompressed := len(prefix) + len(tail)
	got, err := StripMotionDHT(jpeg, recompressed)
	require.NoError(t, err)
	want := append(append([]byte{}, prefix...), tail...)
	require.Equal(t, want, got)
	require.Equal(t, recompressed, len(got))
}
