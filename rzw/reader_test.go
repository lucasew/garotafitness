package rzw

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/lucasew/garotafitness/fourx4"
)

// fg-03.bin solid at 0x1F (rzwb). fg-04 4x4 inner packet is size+CM(.
var rzwbHead = []byte{
	0x6f, 0x04, 0x07, 0x01,
	'C', 'M', '(', 0x05, 0x06, 0x00, 0x00,
	0x33, 0xe3, 0x9c, 0x76, 0x53, 0x04, 0x07, 0x01, 0x00, 0x00,
}
var rzwHead = []byte{
	'C', 'M', '(', 0x05, 0x06, 0x00, 0x00,
	0xce, 0x2d, 0x9a, 0xe5, 0x45, 0xe2, 0x09, 0x00, 0x00, 0x00,
}

func TestNewReader(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   io.Reader
		want error
	}{
		{name: "nil", want: errNil},
		{name: "empty", in: bytes.NewReader(nil), want: io.EOF},
		{name: "short", in: bytes.NewReader([]byte("CM")), want: io.ErrUnexpectedEOF},
		{name: "shortCM", in: bytes.NewReader([]byte("CM(\x05\x06\x00\x00")), want: io.ErrUnexpectedEOF},
		{name: "arc", in: bytes.NewReader([]byte("ArC\x01xxx")), want: errMagic},
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
	for _, tt := range []struct {
		name string
		in   []byte
	}{
		{name: "rzw", in: rzwHead},
		{name: "rzwb", in: rzwbHead},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rc, err := NewReader(bytes.NewReader(tt.in))
			if err != nil {
				t.Fatalf("NewReader(%s) err = %v", tt.name, err)
			}
			t.Cleanup(func() { rc.Close() })
			n, err := rc.Read(make([]byte, 8))
			if n != 0 || !errors.Is(err, errCodec) {
				t.Fatalf("Read(%s) n=%d err=%v; want errCodec", tt.name, n, err)
			}
		})
	}
}

func TestNewReaderVersion(t *testing.T) {
	t.Parallel()
	in := append([]byte(nil), rzwHead...)
	binary.LittleEndian.PutUint32(in[3:7], 0x00000100)
	rc, err := NewReader(bytes.NewReader(in))
	if rc != nil {
		t.Fatalf("reader = %T; want nil", rc)
	}
	if !errors.Is(err, errVersion) {
		t.Fatalf("err = %v; want %v", err, errVersion)
	}
}

func TestNewReaderCorpus(t *testing.T) {
	t.Parallel()
	f, err := os.Open("/media/downloads/TORRENTS/RimWorld [FitGirl Repack]/fg-03.bin")
	if err != nil {
		t.Skip("corpus not mounted")
	}
	t.Cleanup(func() { f.Close() })
	if _, err := f.Seek(0x1F, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	rc, err := NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rc.Close() })
	n, err := rc.Read(make([]byte, 8))
	if n != 0 || !errors.Is(err, errCodec) {
		t.Fatalf("Read n=%d err=%v; want errCodec", n, err)
	}
}

func TestFourX4InnerCorpus(t *testing.T) {
	t.Parallel()
	f, err := os.Open("/media/downloads/TORRENTS/RimWorld [FitGirl Repack]/fg-04.bin")
	if err != nil {
		t.Skip("corpus not mounted")
	}
	t.Cleanup(func() { f.Close() })
	if _, err := f.Seek(0x1F, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	var gotName string
	rd, err := fourx4.NewReader(f, "b128mb:rzw", func(r io.Reader, name, _ string) (io.ReadCloser, error) {
		gotName = name
		return NewReader(r)
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rd.Close() })
	_, err = io.ReadAll(rd)
	if gotName != "rzw" {
		t.Fatalf("inner name %q", gotName)
	}
	if !errors.Is(err, errCodec) {
		t.Fatalf("4x4 inner err = %v; want errCodec", err)
	}
}
