package srep

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFutureMatchBlock(t *testing.T) {
	t.Parallel()
	// One in-block match: literals "ab", then two bytes copied from offset 1.
	stat := make([]byte, 16)
	binary.LittleEndian.PutUint32(stat[0:], 1)
	binary.LittleEndian.PutUint32(stat[4:], 1)
	binary.LittleEndian.PutUint32(stat[12:], 2)
	var in bytes.Buffer
	in.Write(futureLZHead)
	writeFutureBlock(&in, []byte("ab"), stat, 4)
	rc, err := NewReader(t.Context(), &in)
	require.NoError(t, err)
	defer rc.Close()
	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, []byte("abbb"), got)
}

func TestFutureMatchAcrossBlocks(t *testing.T) {
	t.Parallel()
	stat := make([]byte, 16)
	binary.LittleEndian.PutUint32(stat[4:], 4)
	binary.LittleEndian.PutUint32(stat[12:], 4)
	var in bytes.Buffer
	in.Write(futureLZHead)
	writeFutureBlock(&in, []byte("abcd"), stat, 4)
	writeFutureBlock(&in, nil, nil, 4)
	rc, err := NewReader(t.Context(), &in)
	require.NoError(t, err)
	defer rc.Close()
	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, []byte("abcdabcd"), got)
}

func TestFutureRejectsZeroOffset(t *testing.T) {
	t.Parallel()
	stat := make([]byte, 16)
	binary.LittleEndian.PutUint32(stat[12:], 1)
	var in bytes.Buffer
	in.Write(futureLZHead)
	writeFutureBlock(&in, []byte("a"), stat, 1)
	rc, err := NewReader(t.Context(), &in)
	require.NoError(t, err)
	defer rc.Close()
	_, err = io.ReadAll(rc)
	require.ErrorIs(t, err, errBroken)
}

func TestFutureBaseLen(t *testing.T) {
	t.Parallel()
	stat := make([]byte, 16)
	binary.LittleEndian.PutUint32(stat[4:], 1)
	binary.LittleEndian.PutUint32(stat[12:], 1)
	k := newKernel(2)
	out := make([]byte, 4)
	require.True(t, k.block(stat, []byte("a"), out, 0))
	require.Equal(t, []byte("aaaa"), out)
}

func TestFutureSpillAgrees(t *testing.T) {
	t.Parallel()
	const n = 50
	type blk struct {
		lits, stat []byte
		orig       int
	}
	var blocks []blk
	for i := 0; i < 4; i++ {
		lits := bytes.Repeat([]byte{'A' + byte(i)}, n)
		stat := make([]byte, 16)
		binary.LittleEndian.PutUint32(stat[4:], 8*n)
		binary.LittleEndian.PutUint32(stat[12:], n)
		blocks = append(blocks, blk{lits, stat, n})
	}
	for i := 0; i < 4; i++ {
		blocks = append(blocks, blk{bytes.Repeat([]byte{'E' + byte(i)}, n), nil, n})
	}
	for i := 0; i < 4; i++ {
		blocks = append(blocks, blk{nil, nil, n})
	}
	run := func(k *kernel) []byte {
		t.Helper()
		var out []byte
		var start uint64
		for _, b := range blocks {
			buf := make([]byte, b.orig)
			require.True(t, k.block(b.stat, b.lits, buf, start))
			out = append(out, buf...)
			start += uint64(b.orig)
		}
		return out
	}
	huge := run(newKernel(0))
	smallK := newKernelLimited(0, 1<<20, 128)
	smallK.mm.useful = 200
	small := run(smallK)
	require.Equal(t, huge, small)
	require.Equal(t, bytes.Repeat([]byte{'A'}, n), huge[:n])
	require.Equal(t, bytes.Repeat([]byte{'A'}, n), huge[8*n:9*n])
	require.Equal(t, bytes.Repeat([]byte{'D'}, n), huge[11*n:12*n])
}

func writeFutureBlock(b *bytes.Buffer, lits, stat []byte, orig uint32) {
	var hdr [12]byte
	binary.LittleEndian.PutUint32(hdr[0:], uint32(len(lits)))
	binary.LittleEndian.PutUint32(hdr[4:], orig)
	binary.LittleEndian.PutUint32(hdr[8:], uint32(len(stat)))
	b.Write(hdr[:])
	b.Write(make([]byte, 16))
	b.Write(stat)
	b.Write(lits)
}
