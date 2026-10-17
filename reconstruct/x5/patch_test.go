package x5

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyLiteral(t *testing.T) {
	t.Parallel()
	diff := buildHDiff13(2, 0, 0, nil, packTag(1, rleTagBits, rleType0), nil, []byte("XY"))
	got, err := Apply(t.Context(), nil, diff)
	require.NoError(t, err)
	require.Equal(t, []byte("XY"), got)
}

func TestApplyCopy(t *testing.T) {
	t.Parallel()
	cover := concat(packTag(0, signTagBits, 0), packTag(0, 0, 0), packTag(5, 0, 0))
	diff := buildHDiff13(5, 5, 1, cover, packTag(4, rleTagBits, rleType0), nil, nil)
	got, err := Apply(t.Context(), []byte("hello"), diff)
	require.NoError(t, err)
	require.Equal(t, []byte("hello"), got)
}

func TestApplyLiteralThenCopy(t *testing.T) {
	t.Parallel()
	cover := concat(packTag(0, signTagBits, 0), packTag(2, 0, 0), packTag(5, 0, 0))
	diff := buildHDiff13(7, 5, 1, cover, packTag(6, rleTagBits, rleType0), nil, []byte("XY"))
	got, err := Apply(t.Context(), []byte("hello"), diff)
	require.NoError(t, err)
	require.Equal(t, []byte("XYhello"), got)
}

func TestApplyRLEAdd(t *testing.T) {
	t.Parallel()
	cover := concat(packTag(0, signTagBits, 0), packTag(0, 0, 0), packTag(5, 0, 0))
	// len0 stored as 4, value run of five 1s. RLE ctrl is one command of type "byte".
	ctrl := packTag(4, rleTagBits, rleTypeByte)
	code := []byte{1}
	diff := buildHDiff13(5, 5, 1, cover, ctrl, code, nil)
	got, err := Apply(t.Context(), []byte("hello"), diff)
	require.NoError(t, err)
	require.Equal(t, []byte("ifmmp"), got)
}

func TestApplyBackrefAndWideInc(t *testing.T) {
	t.Parallel()
	old := make([]byte, 70)
	for i := range old {
		old[i] = byte(i)
	}
	cover := concat(
		packTag(64, signTagBits, 0), packTag(0, 0, 0), packTag(6, 0, 0),
		packTag(3, signTagBits, 1), packTag(0, 0, 0), packTag(3, 0, 0),
	)
	diff := buildHDiff13(9, uint64(len(old)), 2, cover, packTag(8, rleTagBits, rleType0), nil, nil)
	got, err := Apply(t.Context(), old, diff)
	require.NoError(t, err)
	require.Equal(t, []byte{64, 65, 66, 67, 68, 69, 67, 68, 69}, got)
}

func TestApplyRejects(t *testing.T) {
	t.Parallel()
	_, err := Apply(t.Context(), nil, []byte("nope"))
	require.ErrorIs(t, err, errHeader)

	zlib := append([]byte("HDIFF13&zlib\x00"), packTag(1, 0, 0)...)
	_, err = Apply(t.Context(), nil, zlib)
	require.ErrorIs(t, err, errHeader)

	cover := concat(packTag(0, signTagBits, 0), packTag(0, 0, 0), packTag(1, 0, 0))
	diff := buildHDiff13(1, 1, 1, cover, packTag(0, rleTagBits, rleType0), nil, nil)
	_, err = Apply(t.Context(), []byte("ab"), diff)
	require.ErrorIs(t, err, errHeader)

	diff = append(buildHDiff13(1, 0, 0, nil, packTag(0, rleTagBits, rleType0), nil, []byte("Z")), 0xFF)
	_, err = Apply(t.Context(), nil, diff)
	require.ErrorIs(t, err, errCorrupt)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = Apply(ctx, nil, diff[:len(diff)-1])
	require.ErrorIs(t, err, context.Canceled)
}

func buildHDiff13(newSize, oldSize, covers uint64, cover, rleCtrl, rleCode, lit []byte) []byte {
	var b []byte
	b = append(b, "HDIFF13&\x00"...)
	for _, v := range []uint64{
		newSize, oldSize, covers,
		uint64(len(cover)), 0,
		uint64(len(rleCtrl)), 0,
		uint64(len(rleCode)), 0,
		uint64(len(lit)), 0,
	} {
		b = append(b, packTag(v, 0, 0)...)
	}
	b = append(b, cover...)
	b = append(b, rleCtrl...)
	b = append(b, rleCode...)
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
