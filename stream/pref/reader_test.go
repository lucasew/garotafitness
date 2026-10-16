package pref

import (
	"bytes"
	"embed"
	"encoding/hex"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ulikunitz/xz"
)

//go:embed testdata
var oracleFS embed.FS

func TestNewReader(t *testing.T) {
	t.Parallel()
	_, err := NewReader(t.Context(), nil)
	require.ErrorIs(t, err, errNil)
	_, err = NewReader(t.Context(), bytes.NewReader(nil))
	require.Error(t, err)
	_, err = NewReader(t.Context(), bytes.NewReader([]byte("ArC\x01xxxx")))
	require.Error(t, err)
}

func TestVlint(t *testing.T) {
	t.Parallel()
	for _, n := range []uint64{0, 1, 127, 128, 255, 256, 1000, 1 << 20, (1 << 32) - 1} {
		c := &cursor{b: putVlint(n)}
		got, err := c.vlint()
		require.NoError(t, err, n)
		require.Equal(t, n, got)
		require.Equal(t, len(c.b), c.i)
	}
}

func TestRestoreStreams(t *testing.T) {
	t.Parallel()
	raw, err := hex.DecodeString("68656c6c6f20707265666c617465")
	require.NoError(t, err)
	diff, err := hex.DecodeString("8c48108e0304")
	require.NoError(t, err)
	enc, err := hex.DecodeString("cb48cdc9c95728284a4dcb492c490500")
	require.NoError(t, err)
	hdr := []byte{0x78, 0x9c}

	cases := []struct {
		name string
		in   []byte
		want []byte
	}{
		{"copy", pcf(0, uncChunk([]byte("hello"))), []byte("hello")},
		{"raw", pcf(0, deflateChunk(3, dRaw, hdr, diff, raw, enc, true, nil)), append(append([]byte{}, hdr...), enc...)},
		{"zip", pcf(0, deflateChunk(3, dZIP, hdr, diff, raw, enc, false, nil)), append(append([]byte{'P', 'K', 3, 4}, hdr...), enc...)},
		{"gzip", pcf(0, deflateChunk(3, dGZIP, hdr, diff, raw, enc, false, nil)), append(append([]byte{31, 139}, hdr...), enc...)},
		{"pdf", pcf(0, deflateChunk(3, dPDF, hdr, diff, raw, enc, false, nil)), append(append([]byte("/FlateDecode"), hdr...), enc...)},
		{"png", pcf(0, deflateChunk(3, dPNG, hdr, diff, raw, enc, true, nil)), append(append([]byte("IDAT"), hdr...), enc...)},
		{"swf", pcf(0, deflateChunk(3, dSWF, hdr, diff, raw, enc, true, nil)), append(append([]byte("CWS"), hdr...), enc...)},
		{"brute", pcf(0, deflateChunk(3, dBrute, hdr, diff, raw, enc, false, nil)), append(append([]byte{}, hdr...), enc...)},
		{"recur", pcf(0, deflateChunk(3|128, dRaw, hdr, diff, raw, enc, true, uncChunk(raw))), append(append([]byte{}, hdr...), enc...)},
		{"b64", pcf(0, b64Chunk("A", 0, []int{4}, []byte("abc"), 4)), []byte("AYWJj")},
		{"b64crlf", pcf(0, b64Chunk("A", 0, []int{4}, []byte("abc"), 6)), []byte("AYWJj\r\n")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rc, err := NewReader(t.Context(), bytes.NewReader(tc.in))
			require.NoError(t, err)
			got, err := io.ReadAll(rc)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
			require.NoError(t, rc.Close())
		})
	}
}

func TestMultiPNG(t *testing.T) {
	t.Parallel()
	raw, err := hex.DecodeString("68656c6c6f20707265666c617465")
	require.NoError(t, err)
	diff, err := hex.DecodeString("8c48108e0304")
	require.NoError(t, err)
	enc, err := hex.DecodeString("cb48cdc9c95728284a4dcb492c490500")
	require.NoError(t, err)
	hdr := []byte{0x78, 0x9c}
	body := deflateHdr(3, dMultiPNG, hdr, true)
	body = append(body, putVlint(1)...)         // count-1 = 1 → two IDATs
	body = append(body, putVlint(2+2)...)       // first data length + zlib header
	body = append(body, 0x01, 0x02, 0x03, 0x04) // crc of the second chunk
	body = append(body, putVlint(uint64(len(enc)-2))...)
	body = append(body, putVlint(uint64(len(diff)))...)
	body = append(body, diff...)
	body = append(body, putVlint(uint64(len(enc)))...)
	body = append(body, putVlint(uint64(len(raw)))...)
	body = append(body, raw...)

	rc, err := NewReader(t.Context(), bytes.NewReader(pcf(0, body)))
	require.NoError(t, err)
	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	want := append([]byte("IDAT"), hdr...)
	want = append(want, enc[:2]...)
	want = append(want, 0x01, 0x02, 0x03, 0x04)
	var be [4]byte
	n := len(enc) - 2
	be[0] = byte(uint32(n) >> 24)
	be[1] = byte(uint32(n) >> 16)
	be[2] = byte(uint32(n) >> 8)
	be[3] = byte(n)
	want = append(want, be[:]...)
	want = append(want, "IDAT"...)
	want = append(want, enc[2:]...)
	require.Equal(t, want, got)
}

func TestOTFXZ(t *testing.T) {
	t.Parallel()
	body := uncChunk([]byte("xz-body"))
	var buf bytes.Buffer
	w, err := xz.NewWriter(&buf)
	require.NoError(t, err)
	_, err = w.Write(body)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	rc, err := NewReader(t.Context(), bytes.NewReader(pcf(otfXZ, buf.Bytes())))
	require.NoError(t, err)
	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, []byte("xz-body"), got)
}

func TestUnknownStream(t *testing.T) {
	t.Parallel()
	_, err := NewReader(t.Context(), bytes.NewReader(pcf(0, []byte{1, 42})))
	require.ErrorIs(t, err, errGuest)
	require.ErrorContains(t, err, "unsupported stream")
}

func TestOraclePCF(t *testing.T) {
	t.Parallel()
	pairs := []struct{ pcf, want string }{
		{"testdata/plain.txt.pcf", "testdata/plain.txt"},
		{"testdata/box.gif.pcf", "testdata/box.gif"},
		{"testdata/anim.gif.pcf", "testdata/anim.gif"},
		{"testdata/noise.gif.pcf", "testdata/noise.gif"},
		{"testdata/tiny.bz2.pcf", "testdata/tiny.bz2"},
		{"testdata/lvl1.bz2.pcf", "testdata/lvl1.bz2"},
		{"testdata/tiny.jpg.pcf", "testdata/tiny.jpg"},
		{"testdata/tiny.jpg.packjpg.pcf", "testdata/tiny.jpg"},
		{"testdata/g16.jpg.pcf", "testdata/g16.jpg"},
		{"testdata/g16.packjpg.pcf", "testdata/g16.jpg"},
		{"testdata/tone.mp3.pcf", "testdata/tone.mp3"},
	}
	for _, tc := range pairs {
		t.Run(tc.pcf, func(t *testing.T) {
			t.Parallel()
			in, err := oracleFS.ReadFile(tc.pcf)
			require.NoError(t, err)
			want, err := oracleFS.ReadFile(tc.want)
			require.NoError(t, err)
			rc, err := NewReader(t.Context(), bytes.NewReader(in))
			require.NoError(t, err)
			got, err := io.ReadAll(rc)
			require.NoError(t, err)
			require.Equal(t, want, got)
		})
	}
}

func pcf(method byte, body []byte) []byte {
	b := []byte{'P', 'C', 'F', 0, 4, 8, method, 'n', 0}
	return append(b, body...)
}

func uncChunk(p []byte) []byte {
	b := []byte{0}
	b = append(b, putVlint(uint64(len(p)))...)
	return append(b, p...)
}

func deflateHdr(flags, kind byte, hdr []byte, incLast bool) []byte {
	stored := append([]byte{}, hdr...)
	if incLast {
		stored[len(stored)-1]++
	}
	b := []byte{flags, kind}
	b = append(b, putVlint(uint64(len(hdr)))...)
	return append(b, stored...)
}

func deflateChunk(flags, kind byte, hdr, recon, raw, enc []byte, incLast bool, nested []byte) []byte {
	b := deflateHdr(flags, kind, hdr, incLast)
	b = append(b, putVlint(uint64(len(recon)))...)
	b = append(b, recon...)
	b = append(b, putVlint(uint64(len(enc)))...)
	b = append(b, putVlint(uint64(len(raw)))...)
	if flags&128 != 0 {
		b = append(b, putVlint(uint64(len(nested)))...)
		return append(b, nested...)
	}
	return append(b, raw...)
}

func b64Chunk(header string, lineCase int, lines []int, raw []byte, outN int) []byte {
	flags := byte(1 + (lineCase << 2))
	b := []byte{flags, dBase64}
	hb := []byte(header)
	stored := append([]byte{}, hb...)
	stored[0]--
	b = append(b, putVlint(uint64(len(stored)))...)
	b = append(b, stored...)
	b = append(b, putVlint(uint64(len(lines)))...)
	if lineCase == 2 {
		for _, n := range lines {
			b = append(b, byte(n))
		}
	} else {
		b = append(b, byte(lines[0]))
		if lineCase == 1 {
			b = append(b, byte(lines[len(lines)-1]))
		}
	}
	b = append(b, putVlint(uint64(len(raw)))...)
	b = append(b, putVlint(uint64(outN))...)
	return append(b, raw...)
}
