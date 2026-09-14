package mpzz

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestReadSize(t *testing.T) {
	for _, tt := range []struct {
		in   []byte
		want uint32
	}{
		{[]byte{0}, 0}, {[]byte{0xfc}, 63}, {[]byte{1, 1}, 64},
		{[]byte{0xf9, 0x12}, 1214}, {[]byte{0xff, 0xff, 0xff, 0xff}, 1<<30 - 1},
	} {
		got, err := readSize(bytes.NewReader(tt.in))
		if err != nil || got != tt.want {
			t.Fatalf("%x: %d, %v; want %d", tt.in, got, err, tt.want)
		}
	}
	for _, in := range [][]byte{{1}, {2, 0}, {3, 0, 0}} {
		if _, err := readSize(bytes.NewReader(in)); !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("%x: expected truncation, got %v", in, err)
		}
	}
}
