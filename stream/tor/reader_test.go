package tor

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

//go:embed testdata/*.tor
var samples embed.FS

func TestSamples(t *testing.T) {
	cases := []struct {
		name string
		n    int
		sha  string
	}{
		// BYTECODER: literals, a short match, and the long EOF match.
		{"byte_hello.tor", 23, "eac13dc78da3f79858440011e4db4107e9301314511e6d0073320b14c535b824"},
		// Same text with a 1MiB dictionary (output window is the dictionary, not 256KiB).
		{"byte_1mb.tor", 23, "eac13dc78da3f79858440011e4db4107e9301314511e6d0073320b14c535b824"},
		// Circular buffer: 320000 bytes through a 256KiB window.
		{"byte_wrap.tor", 320000, "85292f2ed0694debb297d38b0c437006fba742e4f967bb053cdb242cf68c5251"},
		// BITCODER, including a match.
		{"bit_hello.tor", 23, "eac13dc78da3f79858440011e4db4107e9301314511e6d0073320b14c535b824"},
		// Bit length code 7 (30 extra bits).
		{"bit_long.tor", 800, "0816ce95452b2213a15648a462086688f8929c9dd7c6ce9044f6b3ee411ed91d"},
		// HUFCODER presets -3 and -4.
		{"huf_hello.tor", 23, "eac13dc78da3f79858440011e4db4107e9301314511e6d0073320b14c535b824"},
		{"huf4_hello.tor", 23, "eac13dc78da3f79858440011e4db4107e9301314511e6d0073320b14c535b824"},
		{"huf_long.tor", 800, "0816ce95452b2213a15648a462086688f8929c9dd7c6ce9044f6b3ee411ed91d"},
		// Repeated distance / REPCHAR.
		{"huf_rep.tor", 1400, "9e9741f603ee30e4bfaa08f5831da487fe6c34ab76a5ce1d172dc427657da5a1"},
		// Two huffman rebuilds (EOB at 1250 and 6250 symbols).
		{"huf_rand.tor", 8000, "475fe19151c325be162357f61b6c2a73e7102b136470867b018db2167562a6ad"},
		// Diffed N=4 table.
		{"huf_table.tor", 1167, "a890fdc5fde5e61075cb0ea949e21e27a24b0b6c1d22658007d46a7b26b2bc80"},
		// Table row split across the 256KiB flush, then the circular shift.
		{"huf_span.tor", 264768, "436467e5af6b79798025340989c3758ffc776a301aeee6f0bb7f3f76a3b71e33"},
		// ARICODER, 30-bit extra bits, repdist, and a rescale.
		{"ari_hello.tor", 23, "eac13dc78da3f79858440011e4db4107e9301314511e6d0073320b14c535b824"},
		{"ari_long.tor", 800, "0816ce95452b2213a15648a462086688f8929c9dd7c6ce9044f6b3ee411ed91d"},
		{"ari_rep.tor", 1400, "9e9741f603ee30e4bfaa08f5831da487fe6c34ab76a5ce1d172dc427657da5a1"},
		{"ari_rand.tor", 4000, "546075653981249a1b6b4d6c81d6ec127828963e64219902ede9a899bb1bebd5"},
		{"ari_table.tor", 1167, "a890fdc5fde5e61075cb0ea949e21e27a24b0b6c1d22658007d46a7b26b2bc80"},
		// N=2 diffed table (minlen 2).
		{"ari_n2.tor", 862, "0481b78b30889db9e1729bea5b7444ac415c42ff08369d19e3dc5521113f5e80"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			in := readSample(t, tc.name)
			rc, err := NewReader(t.Context(), bytes.NewReader(in))
			require.NoError(t, err)
			got, err := io.ReadAll(rc)
			require.NoError(t, err)
			require.NoError(t, rc.Close())
			require.Len(t, got, tc.n)
			require.Equal(t, tc.sha, sha256hex(got))
		})
	}
}

func TestChunkDrop(t *testing.T) {
	t.Parallel()
	// byte_wrap flushes 256KiB then the tail. A cap that fits only the first
	// chunk must drop the tail whole, same as the guest write callback.
	in := readSample(t, "byte_wrap.tor")
	got, err := decompress(t.Context(), in, 262144)
	require.NoError(t, err)
	require.Len(t, got, 262144)
	require.Equal(t, "45392302e4a0844d5162e25bce073a22832e7939b77eecc052bd0b8dd614d1dc", sha256hex(got))
	got, err = decompress(t.Context(), in, 100)
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestEmptyInput(t *testing.T) {
	t.Parallel()
	out, err := decode(t.Context(), nil)
	require.NoError(t, err)
	require.Nil(t, out)
	rc, err := NewReader(t.Context(), bytes.NewReader(nil))
	require.NoError(t, err)
	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Empty(t, got)
	require.NoError(t, rc.Close())
}

func TestEmptyPlaintext(t *testing.T) {
	t.Parallel()
	in := readSample(t, "byte_empty.tor")
	_, err := NewReader(t.Context(), bytes.NewReader(in))
	require.ErrorIs(t, err, errGuest)
}

func TestBadInput(t *testing.T) {
	t.Parallel()
	_, err := NewReader(t.Context(), bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6}))
	require.ErrorIs(t, err, errGuest)
	_, err = NewReader(t.Context(), bytes.NewReader([]byte{9, 4, 0, 0, 1, 0, 1, 2, 3, 4}))
	require.ErrorIs(t, err, errGuest)
}

func TestNilReader(t *testing.T) {
	t.Parallel()
	_, err := NewReader(t.Context(), nil)
	require.EqualError(t, err, "tor: nil reader")
}

func TestReadError(t *testing.T) {
	t.Parallel()
	_, err := NewReader(t.Context(), boomReader{})
	require.EqualError(t, err, "tor: boom")
	require.ErrorIs(t, err, errBoom)
}

func TestCanceled(t *testing.T) {
	t.Parallel()
	in := readSample(t, "byte_hello.tor")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := NewReader(ctx, bytes.NewReader(in))
	require.ErrorIs(t, err, context.Canceled)

	ctx, cancel = context.WithDeadline(t.Context(), time.Unix(0, 0))
	defer cancel()
	_, err = NewReader(ctx, bytes.NewReader(in))
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestVLETables(t *testing.T) {
	t.Parallel()
	require.Equal(t, uint32(281), lcBase[7])
	require.Equal(t, uint32(307), lc2Base[15])
	require.Equal(t, uint32(11599872), dcBase[31])
	require.Equal(t, uint32(30), lcExtra[7])
	require.Equal(t, uint32(30), dcExtra[31])
}

func readSample(t *testing.T, name string) []byte {
	t.Helper()
	b, err := samples.ReadFile("testdata/" + name)
	require.NoError(t, err)
	return b
}

func sha256hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

var errBoom = errors.New("boom")

type boomReader struct{}

func (boomReader) Read([]byte) (int, error) { return 0, errBoom }
