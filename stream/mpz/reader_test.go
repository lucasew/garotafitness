package mpz

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"testing"

	"github.com/lucasew/garotafitness/internal/corpus"
	"github.com/lucasew/garotafitness/stream/fourx4"
	"github.com/lucasew/garotafitness/stream/srep"
)

// First member in the optional solid (FreeArc custom CRC32).
const (
	firstMP3Path = "Soundtrack/1 RimWorld Trailer Music.mp3"
	firstMP3Size = 3468604
	firstMP3CRC  = 0xf7a85e73
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
		{name: "short", in: bytes.NewReader([]byte{0x01, 0x02}), want: io.ErrUnexpectedEOF},
		{name: "arc", in: bytes.NewReader([]byte("ArC\x01xxxx")), want: errMagic},
		{name: "srep", in: bytes.NewReader([]byte("SREP\x03\x01\x00\x00")), want: errMagic},
		{name: "ziganshin", in: bytes.NewReader([]byte{0x17, 0x18, 0x35, 0x26, 0x53, 0x52, 0x45, 0x50}), want: errMagic},
		{name: "oggre", in: bytes.NewReader([]byte("OGGRE\x00\x09\x00")), want: errMagic},
		{name: "razor", in: bytes.NewReader([]byte("CM(\x05\x06\x00\x00\x00")), want: errMagic},
		{name: "lolz", in: bytes.NewReader([]byte("DH(n\x1f\x20\x00\x00")), want: errMagic},
		{name: "id3", in: bytes.NewReader([]byte("ID3\x04\x00\x00\x00\x00")), want: errMagic},
		{name: "mp3", in: bytes.NewReader([]byte{0xff, 0xfb, 0x90, 0x00, 0x00, 0x00, 0x00, 0x00}), want: errMagic},
		{name: "unknown", in: bytes.NewReader(bytes.Repeat([]byte{0x01}, 16)), want: errMagic},
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

func TestNewReaderTagged(t *testing.T) {
	t.Parallel()
	in := frameHead(version5451, 64, 1, 0)
	rc, err := NewReader(bytes.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rc.Close() })
	n, err := rc.Read(make([]byte, 8))
	if n != 0 || err == nil {
		t.Fatalf("Read n=%d err=%v; want error", n, err)
	}
}

func TestFourx4Inner(t *testing.T) {
	t.Parallel()
	payload := frameHead(version5451, 8, 1, 0)
	in := frame4x4(8, payload)
	inner := func(r io.Reader, name, params string) (io.ReadCloser, error) {
		if name != "mpz" || params != "" {
			t.Fatalf("inner %q %q", name, params)
		}
		return NewReader(r)
	}
	rd, err := fourx4.NewReader(bytes.NewReader(in), "b16mb:mpz", inner)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rd.Close() })
	_, err = io.ReadAll(rd)
	if err == nil {
		t.Fatal("want inner decode error")
	}
}

func TestOptionalOST(t *testing.T) {
	t.Parallel()
	f := corpus.File(t, "fg-optional-bonus-soundtrack.bin")
	if _, err := f.Seek(0x1F, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	var ver [4]byte
	if _, err := io.ReadFull(f, ver[:]); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint32(ver[:]) != 0 {
		t.Fatalf("4x4 version %x", ver)
	}
	var hdr [8]byte
	if _, err := io.ReadFull(f, hdr[:]); err != nil {
		t.Fatal(err)
	}
	outSize := binary.LittleEndian.Uint32(hdr[0:4])
	inSize := binary.LittleEndian.Uint32(hdr[4:8])
	if outSize != 16<<20 {
		t.Fatalf("first out %d", outSize)
	}
	head := make([]byte, headerLen)
	if _, err := io.ReadFull(f, head); err != nil {
		t.Fatal(err)
	}
	h, err := ParseHeader(bytes.NewReader(head))
	if err != nil {
		t.Fatal(err)
	}
	if h.Version != version5451 {
		t.Fatalf("mpz version %#x", h.Version)
	}
	if h.Orig != outSize {
		t.Fatalf("mpz orig %d want %d", h.Orig, outSize)
	}
	if inSize < headerLen {
		t.Fatalf("in %d", inSize)
	}
}

func TestOptionalOSTFirstMP3(t *testing.T) {
	if os.Getenv("GAROTAFITNESS_CORPUS_TESTS") == "" {
		t.Skip("set GAROTAFITNESS_CORPUS_TESTS=1 for the full MPZ block check")
	}
	t.Parallel()
	f := corpus.File(t, "fg-optional-bonus-soundtrack.bin")
	if _, err := f.Seek(0x1F, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	var blk [12]byte
	if _, err := io.ReadFull(f, blk[:]); err != nil {
		t.Fatal(err)
	}
	inSize := binary.LittleEndian.Uint32(blk[8:12])
	if _, err := f.Seek(0x1F, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	inner := func(r io.Reader, name, params string) (io.ReadCloser, error) {
		if name != "mpz" {
			t.Fatalf("inner %q", name)
		}
		return NewReader(r)
	}
	// One 4x4 member (version + sizes + payload). Full solid is 178MiB.
	fx, err := fourx4.NewReader(io.LimitReader(f, int64(12+inSize)), "b16mb:mpz", inner)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { fx.Close() })
	got, err := io.ReadAll(fx)
	if err != nil || len(got) == 0 {
		t.Fatalf("mpz guest: %v n=%d", err, len(got))
	}
	if !bytes.HasPrefix(got, []byte{0x17, 0x18, 0x35, 0x26}) && !bytes.HasPrefix(got, []byte("SREP")) {
		t.Fatalf("mpz guest: no srep prefix (%d bytes)", len(got))
	}
	sr, err := srep.NewReader(bytes.NewReader(got))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sr.Close() })
	first := make([]byte, firstMP3Size)
	if _, err := io.ReadFull(sr, first); err != nil {
		t.Fatal(err)
	}
	sum := crc32.Checksum(first, crc32.MakeTable(0x0895171b))
	if sum != firstMP3CRC {
		t.Fatalf("%s crc %08x want %08x", firstMP3Path, sum, firstMP3CRC)
	}
}

func frame4x4(outSize uint32, data []byte) []byte {
	var b bytes.Buffer
	var ver [4]byte
	b.Write(ver[:])
	putU32(&b, outSize)
	putU32(&b, uint32(len(data)))
	b.Write(data)
	return b.Bytes()
}

func putU32(b *bytes.Buffer, v uint32) {
	var p [4]byte
	binary.LittleEndian.PutUint32(p[:], v)
	b.Write(p[:])
}
