package mpzz

import (
	"bytes"
	"errors"
	"io"
	"testing"
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
		{name: "short", in: bytes.NewReader([]byte("OGG")), want: io.ErrUnexpectedEOF},
		{name: "arc", in: bytes.NewReader([]byte("ArC\x01x")), want: errMagic},
		{name: "ogg", in: bytes.NewReader([]byte("OggS\x00")), want: errMagic},
		{name: "oggre", in: bytes.NewReader([]byte(oggre + "\x00\x09")), want: errPEOnly},
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
