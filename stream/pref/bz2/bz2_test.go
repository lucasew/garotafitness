package bz2

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompressHello(t *testing.T) {
	t.Parallel()
	got, err := Compress([]byte("hello"), 9)
	require.NoError(t, err)
	require.Equal(t, []byte{
		0x42, 0x5a, 0x68, 0x39, 0x31, 0x41, 0x59, 0x26, 0x53, 0x59,
		0x19, 0x31, 0x65, 0x3d, 0x00, 0x00, 0x00, 0x81, 0x00, 0x02,
		0x44, 0xa0, 0x00, 0x21, 0x9a, 0x68, 0x33, 0x4d, 0x07, 0x33,
		0x8b, 0xb9, 0x22, 0x9c, 0x28, 0x48, 0x0c, 0x98, 0xb2, 0x9e, 0x80,
	}, got)
}

func TestCompressEmpty(t *testing.T) {
	t.Parallel()
	got, err := Compress(nil, 1)
	require.NoError(t, err)
	require.Equal(t, []byte{
		0x42, 0x5a, 0x68, 0x31, 0x17, 0x72, 0x45, 0x38, 0x50, 0x90, 0x00, 0x00, 0x00, 0x00,
	}, got)
}

func TestPCFPayloads(t *testing.T) {
	t.Parallel()
	dir := filepath.Join("..", "testdata")
	cases := []struct {
		name  string
		level int
	}{
		{"tiny.bz2", 9},
		{"lvl1.bz2", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			pcf, err := os.ReadFile(filepath.Join(dir, tc.name+".pcf"))
			require.NoError(t, err)
			want, err := os.ReadFile(filepath.Join(dir, tc.name))
			require.NoError(t, err)
			body := pcfBody(t, pcf)
			require.Equal(t, []byte{0x01, 9, byte(tc.level)}, body[:3])
			c := body[3:]
			recomp, c := takeVlint(t, c)
			decomp, c := takeVlint(t, c)
			require.Equal(t, decomp, len(c))
			got, err := Compress(c, tc.level)
			require.NoError(t, err)
			if recomp < len(got) {
				got = got[:recomp]
			}
			require.Equal(t, want, got)
		})
	}
}

func pcfBody(t *testing.T, in []byte) []byte {
	t.Helper()
	require.GreaterOrEqual(t, len(in), 8)
	require.Equal(t, "PCF", string(in[:3]))
	i := 7
	for i < len(in) && in[i] != 0 {
		i++
	}
	require.Less(t, i, len(in))
	return in[i+1:]
}

func takeVlint(t *testing.T, b []byte) (int, []byte) {
	t.Helper()
	var v, o uint64
	var s uint
	for i, c := range b {
		if c < 128 {
			return int(v + o + (uint64(c) << s)), b[i+1:]
		}
		v += uint64(c&127) << s
		s += 7
		o = (o + 1) << 7
	}
	t.Fatal("truncated vlint")
	return 0, nil
}

func TestCompressRejectsLevel(t *testing.T) {
	t.Parallel()
	_, err := Compress([]byte("a"), 0)
	require.Error(t, err)
}

// TestMainSort locks the block sorter used once a block reaches 10000 bytes.
// The digest is libbzip2 1.0.6 at level 1 for byte(i*17+i/100).
func TestMainSort(t *testing.T) {
	t.Parallel()
	src := make([]byte, 100000)
	for i := range src {
		src[i] = byte(i*17 + i/100)
	}
	got, err := Compress(src, 1)
	require.NoError(t, err)
	require.Len(t, got, 1164)
	sum := sha256.Sum256(got)
	require.Equal(t, "3fa0e74f691a92449a77bb07bdeaafde1fb7b51828cd1fb8593e01517a439e9e", hex.EncodeToString(sum[:]))
}
