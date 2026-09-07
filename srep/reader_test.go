package srep

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"
)

var futureLZHead = []byte{
	0x17, 0x18, 0x35, 0x26,
	0x53, 0x52, 0x45, 0x50,
	0x03, 0x01, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00,
}

func TestNewReader(t *testing.T) {
	t.Parallel()
	if _, err := NewReader(nil); err == nil {
		t.Fatal("want nil reader error")
	}
	if _, err := NewReader(bytes.NewReader([]byte("ArC\x01"))); err == nil {
		t.Fatal("want header error")
	}
	r, err := NewReader(bytes.NewReader(futureLZHead))
	if errors.Is(err, errWASI) {
		if r != nil {
			t.Fatal("want nil reader")
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
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
	r, err := NewReader(f)
	if errors.Is(err, errWASI) {
		if r != nil {
			t.Fatal("want nil reader")
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
}
