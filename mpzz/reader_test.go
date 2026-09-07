package mpzz

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/lucasew/garotafitness/srep"
)

// fg-01 after SREP: testdata/header.hex.
var fg01Head = []byte{
	'O', 'G', 'G', 'R', 'E', 0x00, 0x09, 0xf9,
	0x12, 0x8c, 0xaa, 0xb3, 0xee, 0xae, 0x5c, 0xde,
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
		{name: "short", in: bytes.NewReader([]byte("OGG")), want: io.ErrUnexpectedEOF},
		{name: "arc", in: bytes.NewReader([]byte("ArC\x01x")), want: errMagic},
		{name: "ogg", in: bytes.NewReader([]byte("OggS\x00")), want: errMagic},
		{name: "oggre", in: bytes.NewReader(fg01Head), want: errPEOnly},
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

func TestHeaderHex(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("testdata/header.hex")
	if err != nil {
		t.Fatal(err)
	}
	var hexDigits strings.Builder
	for _, line := range strings.Split(string(raw), "\n") {
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		hexDigits.WriteString(strings.Map(func(r rune) rune {
			if strings.ContainsRune("0123456789abcdefABCDEF", r) {
				return r
			}
			return -1
		}, line))
	}
	got, err := hex.DecodeString(hexDigits.String())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, fg01Head) {
		t.Fatalf("testdata/header.hex = %x; want %x", got, fg01Head)
	}
}

func TestNewReaderCorpus(t *testing.T) {
	t.Parallel()
	f, err := os.Open("/media/downloads/TORRENTS/RimWorld [FitGirl Repack]/fg-01.bin")
	if err != nil {
		t.Skip("corpus not mounted")
	}
	t.Cleanup(func() { f.Close() })
	if _, err := f.Seek(0x1F, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	sr, err := srep.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sr.Close() })
	head := make([]byte, len(fg01Head))
	if _, err := io.ReadFull(sr, head); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(head, fg01Head) {
		t.Fatalf("fg-01 after SREP = %x; want %x", head, fg01Head)
	}
	rc, err := NewReader(bytes.NewReader(head))
	if rc != nil {
		t.Fatalf("reader = %T; want nil", rc)
	}
	if !errors.Is(err, errPEOnly) {
		t.Fatalf("err = %v; want %v", err, errPEOnly)
	}
}
