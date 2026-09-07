package rzw

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"
)

// fg-03.bin solid at 0x1F (rzwb). fg-04 4x4 inner block starts at CM(.
var rzwbHead = []byte{0x6f, 0x04, 0x07, 0x01, 'C', 'M', '('}
var rzwHead = []byte{'C', 'M', '(', 0x05, 0x06, 0x00, 0x00}

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
		{name: "arc", in: bytes.NewReader([]byte("ArC\x01xxx")), want: errMagic},
		{name: "rzw", in: bytes.NewReader(rzwHead), want: errPEOnly},
		{name: "rzwb", in: bytes.NewReader(rzwbHead), want: errPEOnly},
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
	if rc != nil {
		t.Fatalf("reader = %T; want nil", rc)
	}
	if !errors.Is(err, errPEOnly) {
		t.Fatalf("err = %v; want %v", err, errPEOnly)
	}
}
