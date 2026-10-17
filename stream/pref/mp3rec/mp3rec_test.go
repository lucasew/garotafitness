package mp3rec

import (
	_ "embed"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed testdata/tone25.pmp
var tone25PMP []byte

//go:embed testdata/tone25.mp3
var tone25MP3 []byte

//go:embed testdata/stereo.pmp
var stereoPMP []byte

//go:embed testdata/stereo.mp3
var stereoMP3 []byte

func TestRestoreTone25(t *testing.T) {
	got, err := Restore(tone25PMP)
	require.NoError(t, err)
	requireSame(t, tone25MP3, got)
}

func TestRestoreStereo(t *testing.T) {
	got, err := Restore(stereoPMP)
	require.NoError(t, err)
	requireSame(t, stereoMP3, got)
}

func TestRestoreMemoryLimit(t *testing.T) {
	n := mp3MaxMemorySize/minFrameBytes + 1
	p := []byte{'M', 'S', appVersion, 0x31, 0, 0, 0, byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n)}
	_, err := Restore(p)
	require.Error(t, err)
	require.Contains(t, err.Error(), "MP3_MAX_MEMORY_SIZE")
}

func requireSame(t *testing.T, want, got []byte) {
	t.Helper()
	require.Equal(t, len(want), len(got))
	for i := range want {
		if want[i] != got[i] {
			require.Equalf(t, want[i], got[i], "byte %d", i)
		}
	}
}
