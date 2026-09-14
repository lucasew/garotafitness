package mpz

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"
)

func TestParseHeader(t *testing.T) {
	t.Parallel()
	ok := frameHead(version5451, 16777216, 19968, 0)
	h, err := ParseHeader(bytes.NewReader(ok))
	if err != nil {
		t.Fatal(err)
	}
	if h.Version != version5451 || h.Orig != 16777216 || h.Frames != 19968 || h.Extra != 0 {
		t.Fatalf("got %+v", h)
	}
	old := frameHead(version5450, 100, 1, 0)[:4]
	h, err = ParseHeader(bytes.NewReader(old))
	if err != nil {
		t.Fatal(err)
	}
	if h.Version != version5450 || h.Orig != 0 {
		t.Fatalf("got %+v", h)
	}
}

func TestParseHeaderErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   []byte
		want error
	}{
		{name: "empty", want: io.EOF},
		{name: "short", in: []byte{0x05, 0x04}, want: io.ErrUnexpectedEOF},
		{name: "arc", in: pad16([]byte("ArC\x01")), want: errMagic},
		{name: "unknown", in: bytes.Repeat([]byte{0x01}, 16), want: errMagic},
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

func frameHead(ver, orig, frames, extra uint32) []byte {
	var b [headerLen]byte
	binary.LittleEndian.PutUint32(b[0:4], ver)
	binary.LittleEndian.PutUint32(b[4:8], orig)
	binary.LittleEndian.PutUint32(b[8:12], frames)
	binary.LittleEndian.PutUint32(b[12:16], extra)
	return b[:]
}

func pad16(p []byte) []byte {
	out := make([]byte, headerLen)
	copy(out, p)
	return out
}
