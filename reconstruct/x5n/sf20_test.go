package x5n

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSF20Copy(t *testing.T) {
	t.Parallel()
	cover := concat(packTag(0, 1, 0), packTag(0, 0, 0), packTag(5, 0, 0))
	body := step(cover, packTag(5, 0, 0), nil)
	diff := sf20(5, 5, 1, uint64(len(cover)+1), body)
	got, err := applySF20(t.Context(), []byte("hello"), diff)
	require.NoError(t, err)
	require.Equal(t, []byte("hello"), got)
}

func TestSF20LiteralRLEAndTerminator(t *testing.T) {
	t.Parallel()
	// "XY" + ("hello"+1) + "!" via a zero-length cover.
	cover := concat(
		packTag(0, 1, 0), packTag(2, 0, 0), packTag(5, 0, 0),
		packTag(0, 1, 0), packTag(1, 0, 0), packTag(0, 0, 0),
	)
	rle := concat(packTag(0, 0, 0), packTag(5, 0, 0), []byte{1, 1, 1, 1, 1})
	body := step(cover, rle, []byte("XY!"))
	diff := sf20(8, 5, 2, uint64(len(cover)+len(rle)), body)
	got, err := applySF20(t.Context(), []byte("hello"), diff)
	require.NoError(t, err)
	require.Equal(t, []byte("XYifmmp!"), got)
}

func TestSF20TwoSteps(t *testing.T) {
	t.Parallel()
	c1 := concat(packTag(0, 1, 0), packTag(0, 0, 0), packTag(2, 0, 0))
	c2 := concat(packTag(0, 1, 0), packTag(0, 0, 0), packTag(2, 0, 0))
	body := concat(step(c1, packTag(2, 0, 0), nil), step(c2, packTag(2, 0, 0), nil))
	stepMem := uint64(len(c1) + 1)
	diff := sf20(4, 4, 2, stepMem, body)
	got, err := applySF20(t.Context(), []byte("wxyz"), diff)
	require.NoError(t, err)
	require.Equal(t, []byte("wxyz"), got)
}

func TestSF20Rejects(t *testing.T) {
	t.Parallel()
	_, err := applySF20(t.Context(), nil, []byte("HDIFF13&"))
	require.ErrorIs(t, err, errSF20)

	cover := concat(packTag(0, 1, 0), packTag(0, 0, 0), packTag(1, 0, 0))
	body := step(cover, packTag(1, 0, 0), nil)
	diff := sf20(1, 1, 1, uint64(len(cover)+1), body)
	_, err = applySF20(t.Context(), []byte("ab"), diff)
	require.ErrorIs(t, err, errSF20)

	// compressedSize != 0
	diff = sf20Compressed(1, 0, 0, 0, 0, 1)
	_, err = applySF20(t.Context(), nil, diff)
	require.ErrorIs(t, err, errSF20)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = applySF20(ctx, nil, diff)
	require.ErrorIs(t, err, context.Canceled)
}

func sf20(newSize, oldSize, covers, stepMem uint64, body []byte) []byte {
	return sf20Compressed(newSize, oldSize, covers, stepMem, uint64(len(body)), 0, body)
}

func sf20Compressed(newSize, oldSize, covers, stepMem, uncompressed, compressed uint64, body ...[]byte) []byte {
	var b []byte
	b = append(b, "HDIFFSF20&\x00"...)
	for _, v := range []uint64{newSize, oldSize, covers, stepMem, uncompressed, compressed} {
		b = append(b, packTag(v, 0, 0)...)
	}
	for _, p := range body {
		b = append(b, p...)
	}
	return b
}

func step(cover, rle, lit []byte) []byte {
	var b []byte
	b = append(b, packTag(uint64(len(cover)), 0, 0)...)
	b = append(b, packTag(uint64(len(rle)), 0, 0)...)
	b = append(b, cover...)
	b = append(b, rle...)
	b = append(b, lit...)
	return b
}

func packTag(value uint64, tagBits, highTag uint) []byte {
	maxFirst := uint64(1<<(7-tagBits)) - 1
	var tail []byte
	for value > maxFirst {
		tail = append(tail, byte(value&0x7f))
		value >>= 7
	}
	first := byte(value) | byte(highTag<<(8-tagBits))
	if len(tail) > 0 {
		first |= 1 << (7 - tagBits)
	}
	out := []byte{first}
	for i := len(tail) - 1; i >= 0; i-- {
		b := tail[i]
		if i != 0 {
			b |= 0x80
		}
		out = append(out, b)
	}
	return out
}

func concat(parts ...[]byte) []byte {
	var b []byte
	for _, p := range parts {
		b = append(b, p...)
	}
	return b
}
