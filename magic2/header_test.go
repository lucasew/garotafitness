package magic2

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"
)

func TestParseHeader(t *testing.T) {
	t.Parallel()
	h, err := ParseHeader(bytes.NewReader(testdata(t, "fg06.head")))
	if err != nil {
		t.Fatal(err)
	}
	if h.Ver != verV22c4b {
		t.Fatalf("ver %x; want %x", h.Ver, verV22c4b)
	}
}

func TestParseHeaderFG02(t *testing.T) {
	t.Parallel()
	h, err := ParseHeader(bytes.NewReader(testdata(t, "fg02.head")))
	if err != nil {
		t.Fatal(err)
	}
	if h.Ver != verV22c4b {
		t.Fatalf("ver %x; want %x", h.Ver, verV22c4b)
	}
}

func TestParseHeaderRejects(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   []byte
		want error
	}{
		{name: "empty", want: io.EOF},
		{name: "short", in: []byte("DH"), want: io.ErrUnexpectedEOF},
		{name: "arc", in: []byte("ArC\x01x"), want: errMagic},
		{name: "ver", in: []byte(lolzTag + "\x00"), want: errVersion},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseHeader(bytes.NewReader(tt.in))
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v; want %v", err, tt.want)
			}
		})
	}
}

func TestFG02SolidTag(t *testing.T) {
	t.Parallel()
	const corpus = `/media/downloads/TORRENTS/RimWorld [FitGirl Repack]/fg-02.bin`
	f, err := os.Open(corpus)
	if err != nil {
		t.Skip("corpus not mounted")
	}
	t.Cleanup(func() { f.Close() })
	if _, err := f.Seek(0x1F, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	want := testdata(t, "fg02.head")
	head := make([]byte, len(want))
	if _, err := io.ReadFull(f, head); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(head, want) {
		t.Fatalf("fg-02 solid at 0x1F = %x; want %x", head, want)
	}
}
