package magic2

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"
)

// fg-06.bin solid at 0x1F (method magic2).
var fg06Head = []byte{
	0x44, 0x48, 0x28, 0x6e, 0x1f, 0x20, 0x00, 0x00,
	0x00, 0x02, 0x00, 0x25, 0x00, 0x00, 0x00, 0xfa,
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
		{name: "short", in: bytes.NewReader([]byte("DH")), want: io.ErrUnexpectedEOF},
		{name: "arc", in: bytes.NewReader([]byte("ArC\x01x")), want: errMagic},
		{name: "srep", in: bytes.NewReader([]byte("SREP\x03")), want: errMagic},
		{name: "lolz", in: bytes.NewReader(fg06Head), want: errPEOnly},
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

func TestFG06SolidTag(t *testing.T) {
	t.Parallel()
	const corpus = `/media/downloads/TORRENTS/RimWorld [FitGirl Repack]/fg-06.bin`
	f, err := os.Open(corpus)
	if err != nil {
		t.Skip("corpus not mounted")
	}
	t.Cleanup(func() { f.Close() })
	if _, err := f.Seek(0x1F, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	head := make([]byte, len(fg06Head))
	if _, err := io.ReadFull(f, head); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(head, fg06Head) {
		t.Fatalf("fg-06 solid at 0x1F = %x; want %x", head, fg06Head)
	}
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
