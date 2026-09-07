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
	raw := testdata(t, "fg06.head")
	h, err := ParseHeader(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if string(h.Tag[:]) != lolzTag {
		t.Fatalf("tag %q; want %q", h.Tag, lolzTag)
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
	if string(h.Tag[:]) != lolzTag {
		t.Fatalf("tag %q; want %q", h.Tag, lolzTag)
	}
	if h.Ver != verV22c4b {
		t.Fatalf("ver %x; want %x", h.Ver, verV22c4b)
	}
}

func TestParseHeaderExactPrefix(t *testing.T) {
	t.Parallel()
	in := append([]byte(lolzTag), verV22c4b)
	if len(in) != headerLen {
		t.Fatalf("prefix %d; want %d", len(in), headerLen)
	}
	h, err := ParseHeader(bytes.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if h.Ver != verV22c4b {
		t.Fatalf("ver %x; want %x", h.Ver, verV22c4b)
	}
}

func TestParseHeaderLeavesBitstream(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		file string
		rest []byte
	}{
		{name: "fg06", file: "fg06.head", rest: []byte{0x20, 0x00, 0x00, 0x00, 0x02, 0x00, 0x25, 0x00, 0x00, 0x00, 0xfa}},
		{name: "fg02", file: "fg02.head", rest: []byte{0xc0, 0x03, 0x77, 0x00, 0xac, 0x03, 0x04, 0x61, 0x00, 0x00, 0x07}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			raw := testdata(t, tt.file)
			if len(raw) < headerLen+len(tt.rest) {
				t.Fatalf("head %d; want at least %d", len(raw), headerLen+len(tt.rest))
			}
			r := bytes.NewReader(raw)
			if _, err := ParseHeader(r); err != nil {
				t.Fatal(err)
			}
			got := make([]byte, len(tt.rest))
			if _, err := io.ReadFull(r, got); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, tt.rest) {
				t.Fatalf("leftover %x; want %x", got, tt.rest)
			}
			if !bytes.Equal(got, raw[headerLen:headerLen+len(tt.rest)]) {
				t.Fatalf("leftover != testdata[%d:]", headerLen)
			}
		})
	}
}

func TestHeadsShareOnlyTagVersion(t *testing.T) {
	t.Parallel()
	a := testdata(t, "fg06.head")
	b := testdata(t, "fg02.head")
	if len(a) < headerLen || len(b) < headerLen {
		t.Fatal("short testdata")
	}
	if !bytes.Equal(a[:headerLen], b[:headerLen]) {
		t.Fatalf("proved prefix %x vs %x", a[:headerLen], b[:headerLen])
	}
	if bytes.Equal(a[headerLen:], b[headerLen:]) {
		t.Fatal("unproved tail identical; field map needs a third sample")
	}
}

func TestNotV20OptionsBlob(t *testing.T) {
	t.Parallel()
	// A LE store of dict<<24 is 00 00 00 XX. Neither live dword is.
	for _, name := range []string{"fg06.head", "fg02.head"} {
		raw := testdata(t, name)
		if len(raw) < headerLen+4 {
			t.Fatalf("%s short", name)
		}
		word := raw[headerLen : headerLen+4]
		if word[0] == 0 && word[1] == 0 && word[2] == 0 {
			t.Fatalf("%s +5 %x looks like LE dict<<24; v22 was not supposed to be that blob", name, word)
		}
		if bytes.HasPrefix(raw[headerLen:], optionTable32) {
			t.Fatalf("%s leftover is optionTable32; that blob is in-memory only", name)
		}
	}
}

func TestNoPlainSizesInHead(t *testing.T) {
	t.Parallel()
	// Sizes known from the ArC directory / member table. None of them
	// are a field in the 16-byte testdata prefix (proved by absence).
	sizes := []uint32{93116, 430889, 2895, 6, 190, 13622, 414176, 5}
	for _, name := range []string{"fg06.head", "fg02.head"} {
		raw := testdata(t, name)
		for _, v := range sizes {
			if v > 0xffff {
				le := []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)}
				be := []byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}
				if bytes.Contains(raw, le) || bytes.Contains(raw, be) {
					t.Fatalf("%s contains size %d", name, v)
				}
				continue
			}
			le := []byte{byte(v), byte(v >> 8)}
			if v != 0 && bytes.Contains(raw, le) {
				// 6 and 5 can appear as payload noise; only fail
				// if they sit on a 2-byte boundary in the proved
				// prefix (they do not: prefix is DH(n 1f).
				if bytes.Contains(raw[:headerLen], le) {
					t.Fatalf("%s prefix contains size %d", name, v)
				}
			}
		}
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
	if _, err := f.Seek(0x1F, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseHeader(f); err != nil {
		t.Fatal(err)
	}
	rest := make([]byte, len(want)-headerLen)
	if _, err := io.ReadFull(f, rest); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rest, want[headerLen:]) {
		t.Fatalf("fg-02 leftover %x; want %x", rest, want[headerLen:])
	}
}
