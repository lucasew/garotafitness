package rzs

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"testing"

	"github.com/lucasew/garotafitness/internal/corpus"
)

func TestNewReader(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   io.Reader
		want error
	}{
		{name: "nil", want: errNil},
		{name: "empty", in: bytes.NewReader(nil), want: io.EOF},
		{name: "short", in: bytes.NewReader([]byte{1, 2, 3}), want: io.ErrUnexpectedEOF},
		{name: "zero", in: bytes.NewReader(make([]byte, headerSize)), want: errTooLarge},
		{name: "arc", in: bytes.NewReader(append(sizeHdr(8, 8), []byte("ArC\x01xxxx")...)), want: errMagic},
		{name: "shortCM", in: bytes.NewReader(append(sizeHdr(8, 8), []byte("CM(\x05\x06\x00")...)), want: io.ErrUnexpectedEOF},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rc, err := NewReader(tt.in)
			if rc != nil {
				t.Fatalf("NewReader(%s) reader = %T; want nil", tt.name, rc)
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("NewReader(%s) err = %v; want %v", tt.name, err, tt.want)
			}
		})
	}
}

func TestHeaderSizes(t *testing.T) {
	t.Parallel()
	var hdr [headerSize]byte
	binary.LittleEndian.PutUint64(hdr[0:8], 100)
	binary.LittleEndian.PutUint64(hdr[8:16], 4<<30+1)
	_, err := NewReader(bytes.NewReader(hdr[:]))
	if !errors.Is(err, errTooLarge) {
		t.Fatalf("got %v; want %v", err, errTooLarge)
	}
}

func TestNewReaderVersion(t *testing.T) {
	t.Parallel()
	in := append(sizeHdr(8, 8), []byte("CM(\x00\x06\x00\x00")...)
	rc, err := NewReader(bytes.NewReader(in))
	if rc != nil {
		t.Fatalf("reader = %T; want nil", rc)
	}
	if !errors.Is(err, errVersion) {
		t.Fatalf("err = %v; want %v", err, errVersion)
	}
}

func TestCorpusHeaders(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		plain  uint64
		packed uint64
		index  uint64
	}{
		{name: "fg-01.bin", plain: 0x37e068cb, packed: 0x372ad910, index: 0x372ad8fa},
		{name: "fg-02.bin", plain: 0x1e6165d5, packed: 0x1dde4b95, index: 0x1dde4b7f},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := corpus.FileEnv(t, "GAROTAFITNESS_CORPUS_SOC", tt.name)
			if _, err := f.Seek(31, io.SeekStart); err != nil {
				t.Fatal(err)
			}
			var hdr [headerSize]byte
			if _, err := io.ReadFull(f, hdr[:]); err != nil {
				t.Fatal(err)
			}
			plain := binary.LittleEndian.Uint64(hdr[0:8])
			packed := binary.LittleEndian.Uint64(hdr[8:16])
			if plain != tt.plain || packed != tt.packed {
				t.Fatalf("plain=%#x packed=%#x; want %#x %#x", plain, packed, tt.plain, tt.packed)
			}
			off, n, err := parseCM(f)
			if err != nil {
				t.Fatal(err)
			}
			if off != tt.index || n != 20 {
				t.Fatalf("index=%#x consumed=%d; want %#x 20", off, n, tt.index)
			}
		})
	}
}

func TestParseRawHeader(t *testing.T) {
	t.Parallel()
	payload := []byte{0x45, 0xe2, 0x09, 0x00, 0x00, 0x00}
	in := append([]byte("CM(\x05\x06\x00\x00"), framed(payload)[3:]...)
	off, n, err := parseCM(bytes.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if off != 0x9e245 || n != 17 {
		t.Fatalf("index=%#x consumed=%d", off, n)
	}
}

func TestParseStdioHeader(t *testing.T) {
	t.Parallel()
	payload := []byte{0x7f, 0x4b, 0xde, 0x1d, 0x00, 0x00}
	in := append([]byte("CM(\x05\x06\x00\x00\x06\x00\x00"), framed(payload)[3:]...)
	off, n, err := parseCM(bytes.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if off != 0x1dde4b7f || n != 20 {
		t.Fatalf("index=%#x consumed=%d", off, n)
	}
}

func TestNewReaderCorpusHeader(t *testing.T) {
	t.Parallel()
	f := corpus.FileEnv(t, "GAROTAFITNESS_CORPUS_SOC", "fg-02.bin")
	if _, err := f.Seek(31, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	rc, err := NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rc.Close() })
}

func sizeHdr(plain, packed uint64) []byte {
	var hdr [headerSize]byte
	binary.LittleEndian.PutUint64(hdr[0:8], plain)
	binary.LittleEndian.PutUint64(hdr[8:16], packed)
	return hdr[:]
}

func framed(b []byte) []byte {
	h := []byte{byte(len(b)), byte(len(b) >> 8), byte(len(b) >> 16), 0, 0, 0, 0}
	crc := crc32.Update(crc32.ChecksumIEEE(h[:3]), crc32.IEEETable, b)
	binary.LittleEndian.PutUint32(h[3:], crc)
	return append(h, b...)
}
