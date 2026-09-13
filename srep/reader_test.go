package srep

import (
	"bytes"
	"encoding/binary"
	"io"
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
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	n, err := r.Read(make([]byte, 8))
	if n != 0 || err != io.EOF {
		t.Fatalf("empty solid: n=%d err=%v", n, err)
	}
}

func TestNewReaderLiterals(t *testing.T) {
	t.Parallel()
	plain := []byte("hello")
	r, err := NewReader(bytes.NewReader(literalSolid(plain)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("got %q want %q", got, plain)
	}
}

func TestNewReaderCorpus(t *testing.T) {
	t.Parallel()
	f := openCorpusFile(t, "fg-01.bin")
	if _, err := f.Seek(0x1F, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	r, err := NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	buf := make([]byte, 5)
	if _, err := io.ReadFull(r, buf); err != nil {
		t.Fatal(err)
	}
	if string(buf) != "OGGRE" {
		t.Fatalf("inner magic %q", buf)
	}
	// FreeArc trailer after the last literal block is not an SREP header.
	n, err := io.Copy(io.Discard, r)
	if err != nil {
		t.Fatal(err)
	}
	// 5 bytes already read + remainder = 216364145
	if n+5 != 216364145 {
		t.Fatalf("after-srep %d; want %d", n+5, 216364145)
	}
}

func literalSolid(plain []byte) []byte {
	var b bytes.Buffer
	b.Write(futureLZHead)
	var hdr [12]byte
	binary.LittleEndian.PutUint32(hdr[0:4], uint32(len(plain)))
	binary.LittleEndian.PutUint32(hdr[4:8], uint32(len(plain)))
	b.Write(hdr[:])
	b.Write(make([]byte, 16))
	b.Write(plain)
	return b.Bytes()
}
