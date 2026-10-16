package xdelta

import (
	"bytes"
	"encoding/binary"
	"hash/adler32"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func literalPatch(data []byte) []byte {
	// VCDIFF, one checksummed window, ADD with explicit size, no source.
	b := []byte{0xd6, 0xc3, 0xc4, 0, 0, 4, byte(11 + len(data)), byte(len(data)), 0, byte(len(data)), 2, 0}
	b = binary.BigEndian.AppendUint32(b, adler32.Checksum(data))
	b = append(b, data...)
	return append(b, 1, byte(len(data)))
}

func TestApplyStreamMatchesMemory(t *testing.T) {
	want := []byte("hello, patch")
	patch := literalPatch(want)
	var buf bytes.Buffer
	err := ApplyStream(t.Context(), bytes.NewReader(nil), 0, bytes.NewReader(patch), int64(len(patch)), SeqWriter(&buf))
	require.NoError(t, err)
	require.Equal(t, want, buf.Bytes())
}

func TestApplyWindowChecksum(t *testing.T) {
	want := []byte("hello, patch")
	patch := literalPatch(want)
	got, err := Apply(t.Context(), nil, patch)
	require.NoError(t, err)
	require.Equal(t, want, got)
	bad := bytes.Clone(patch)
	bad[12] ^= 1
	_, err = Apply(t.Context(), nil, bad)
	require.Error(t, err)
	for n := 5; n < len(patch); n++ {
		_, err := targetSize(patch[:n])
		require.Error(t, err)
	}
}

func TestTargetSizeBounds(t *testing.T) {
	for _, b := range [][]byte{
		{}, {0xd6, 0xc3, 0xc4, 0, 1},
		{0xd6, 0xc3, 0xc4, 0, 4, 0xff, 0xff, 0xff, 0xff, 0x7f},
		{0xd6, 0xc3, 0xc4, 0, 0, 3},
	} {
		_, err := targetSize(b)
		require.Error(t, err)
	}
}

func TestCodeTableShape(t *testing.T) {
	require.Equal(t, byte(opRun), codeTable[0].t1)
	require.Equal(t, byte(opAdd), codeTable[1].t1)
	require.Equal(t, byte(0), codeTable[1].s1)
	// Mode SELF, direct size 6.
	require.Equal(t, byte(opCpy), codeTable[22].t1)
	require.Equal(t, byte(6), codeTable[22].s1)
	// Mode HERE, direct size 4.
	require.Equal(t, byte(opCpy+1), codeTable[36].t1)
	require.Equal(t, byte(4), codeTable[36].s1)
	// ADD2 + COPY6, mode SELF.
	require.Equal(t, byte(opAdd), codeTable[168].t1)
	require.Equal(t, byte(2), codeTable[168].s1)
	require.Equal(t, byte(opCpy), codeTable[168].t2)
	require.Equal(t, byte(6), codeTable[168].s2)
	require.Equal(t, byte(opCpy+8), codeTable[255].t1)
	require.Equal(t, byte(4), codeTable[255].s1)
	require.Equal(t, byte(opAdd), codeTable[255].t2)
	require.Equal(t, byte(1), codeTable[255].s2)
}

func TestApplySourceCopy(t *testing.T) {
	src := []byte("abcdefghij")
	want := []byte("XYabcdef")
	body := []byte{byte(len(want)), 0, 2, 1, 1}
	body = binary.BigEndian.AppendUint32(body, adler32.Checksum(want))
	body = append(body, 'X', 'Y', 168, 0)
	patch := append([]byte{0xd6, 0xc3, 0xc4, 0, 0, 5, 6, 0, byte(len(body))}, body...)
	got, err := Apply(t.Context(), src, patch)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestApplyHereAndRun(t *testing.T) {
	src := []byte("abcdefghij")
	here := []byte("ghij")
	body := []byte{byte(len(here)), 0, 0, 1, 1}
	body = binary.BigEndian.AppendUint32(body, adler32.Checksum(here))
	body = append(body, 36, 4)
	patch := append([]byte{0xd6, 0xc3, 0xc4, 0, 0, 5, 10, 0, byte(len(body))}, body...)
	got, err := Apply(t.Context(), src, patch)
	require.NoError(t, err)
	require.Equal(t, here, got)

	run := []byte("ZZZZ")
	body = []byte{byte(len(run)), 0, 1, 2, 0}
	body = binary.BigEndian.AppendUint32(body, adler32.Checksum(run))
	body = append(body, 'Z', 0, 4)
	patch = append([]byte{0xd6, 0xc3, 0xc4, 0, 0, 4, byte(len(body))}, body...)
	got, err = Apply(t.Context(), nil, patch)
	require.NoError(t, err)
	require.Equal(t, run, got)
}

func TestApplyOverlappingCopy(t *testing.T) {
	want := []byte("aaaa")
	body := []byte{byte(len(want)), 0, 1, 3, 1}
	body = binary.BigEndian.AppendUint32(body, adler32.Checksum(want))
	body = append(body, 'a', 2, 19, 3, 0)
	patch := append([]byte{0xd6, 0xc3, 0xc4, 0, 0, 4, byte(len(body))}, body...)
	got, err := Apply(t.Context(), nil, patch)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestApplyTwoWindows(t *testing.T) {
	a := literalPatch([]byte("one"))
	b := literalPatch([]byte("two"))
	patch := append(append([]byte{}, a...), b[5:]...)
	got, err := Apply(t.Context(), nil, patch)
	require.NoError(t, err)
	require.Equal(t, []byte("onetwo"), got)
}

func TestApplyStreamLargeSourceOffset(t *testing.T) {
	const off = 1 << 32
	payload := []byte("WXYZ")
	body := []byte{byte(len(payload)), 0, 0, 1, 1}
	body = binary.BigEndian.AppendUint32(body, adler32.Checksum(payload))
	body = append(body, 20, 0)
	// 1<<32, high 7-bit group first: 0x10, then four zero groups.
	cpyoff := []byte{0x90, 0x80, 0x80, 0x80, 0x00}
	patch := append([]byte{0xd6, 0xc3, 0xc4, 0, 0, 5, byte(len(payload))}, cpyoff...)
	patch = append(patch, byte(len(body)))
	patch = append(patch, body...)
	_, err := Apply(t.Context(), payload, patch)
	require.Error(t, err)

	src := offsetReader{base: off, data: payload}
	var buf bytes.Buffer
	err = ApplyStream(t.Context(), src, off+int64(len(payload)), bytes.NewReader(patch), int64(len(patch)), SeqWriter(&buf))
	require.NoError(t, err)
	require.Equal(t, payload, buf.Bytes())
}

type offsetReader struct {
	base int64
	data []byte
}

func (o offsetReader) ReadAt(p []byte, pos int64) (int, error) {
	rel := pos - o.base
	if rel < 0 || rel > int64(len(o.data)) {
		return 0, io.EOF
	}
	n := copy(p, o.data[rel:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}
