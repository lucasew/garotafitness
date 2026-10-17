package xt2png

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"

	"github.com/lucasew/garotafitness/internal/xtl"
	"github.com/stretchr/testify/require"
)

func TestFS25FirstChunkRestore(t *testing.T) {
	raw, err := os.ReadFile("testdata/fg01.head.bin")
	require.NoError(t, err)
	h, err := parseHeader(bytes.NewReader(raw))
	require.NoError(t, err)
	rest := raw[headerLen(h):]
	sc := i32le(rest)
	bs := i64le(rest[4:])
	require.Equal(t, int32(3), sc)
	require.Equal(t, int64(91), bs)
	rest = rest[12:]
	r := &reader{ctx: t.Context(), hdr: h}
	var got int
	off := 0
	wantSHA := []string{
		"c8dc8082021c0cde2ce0cd8a75f4cc952da4d04c54c9068b0560287594d7b333",
		"a68099ca9aacf11f41aafc7800a078405ff7e1a2bf59b21bdccde45be5e95251",
		"5c4766be88d2f9fe992be9232bf5b0a4045469c29fb8605696c2a9f211825e2f",
	}
	wantLen := []int{28, 17, 23}
	for i := 0; i < 3; i++ {
		sh, err := xtl.ReadStreamHeader(bytes.NewReader(rest[i*18:]))
		require.NoError(t, err)
		payload := rest[3*18+off : 3*18+off+int(sh.NewSize)]
		off += int(sh.NewSize)
		rawp, ext, err := r.takeStreamFrom(sh, payload)
		require.NoError(t, err)
		out, err := r.restore(sh, rawp, ext)
		require.NoError(t, err)
		require.Equal(t, sh.OldSize, int32(len(out)))
		require.Equal(t, wantLen[i], len(out))
		sum := sha256.Sum256(out)
		require.Equal(t, wantSHA[i], hex.EncodeToString(sum[:]))
		got += len(out)
	}
	require.Equal(t, 28+17+23, got)
}

func (r *reader) takeStreamFrom(h streamHeader, payload []byte) (raw, ext []byte, err error) {
	r.block = payload
	r.blockOff = 0
	return r.takeStream(h)
}
