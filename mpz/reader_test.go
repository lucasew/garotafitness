package mpz

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/lucasew/garotafitness/fourx4"
)

const optionalOST = `/media/downloads/TORRENTS/RimWorld [FitGirl Repack]/fg-optional-bonus-soundtrack.bin`

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
		{name: "unknown", in: bytes.NewReader([]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}), want: errPEOnly},
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

func TestFourx4Inner(t *testing.T) {
	t.Parallel()
	payload := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	in := frame4x4(uint32(len(payload)+8), payload)
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
	if !errors.Is(err, errPEOnly) {
		t.Fatalf("err = %v; want %v", err, errPEOnly)
	}
}

func TestOptionalOST(t *testing.T) {
	t.Parallel()
	f, err := os.Open(optionalOST)
	if err != nil {
		t.Skip("corpus not mounted")
	}
	t.Cleanup(func() { f.Close() })
	// Method is srep+4x4:b16mb:mpz. Solid at 0x1F has no SREP tag, so
	// the inner mpz stream is not reachable from this package.
	if _, err := f.Seek(0x1F, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	head := make([]byte, 8)
	if _, err := io.ReadFull(f, head); err != nil {
		t.Fatal(err)
	}
	if string(head[:4]) == "SREP" || bytes.Equal(head[:4], []byte{0x17, 0x18, 0x35, 0x26}) {
		t.Fatal("optional solid grew an SREP tag; re-sample mpz magic")
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
