package dispack

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestTagData(t *testing.T) {
	t.Parallel()
	plain := []byte("hello dispack")
	in := packData(16*1024, plain)
	rd, err := NewReader(bytes.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(rd)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(plain) {
		t.Fatalf("got %q", got)
	}
}

func TestOfficialCode(t *testing.T) {
	t.Parallel()
	assertEXE(t, "code.filt", "code.plain")
}

func TestOfficialJumpTable(t *testing.T) {
	t.Parallel()
	assertEXE(t, "jumptab.filt", "jumptab.plain")
}

func TestUnfilterDirect(t *testing.T) {
	t.Parallel()
	src, err := os.ReadFile(filepath.Join("testdata", "code.filt"))
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "code.plain"))
	if err != nil {
		t.Fatal(err)
	}
	got := make([]byte, len(want))
	if !unfilter(src, got, baseStart) {
		t.Fatal("unfilter failed")
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("got %x want %x", got, want)
	}
}

func TestNewReaderNil(t *testing.T) {
	t.Parallel()
	if _, err := NewReader(nil); err == nil {
		t.Fatal("want nil reader")
	}
}

func TestEmpty(t *testing.T) {
	t.Parallel()
	rd, err := NewReader(bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(rd)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got %q", got)
	}
}

func TestBadTag(t *testing.T) {
	t.Parallel()
	var b bytes.Buffer
	putU32(&b, 16*1024)
	putU32(&b, tagData+3)
	rd, err := NewReader(bytes.NewReader(b.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadAll(rd); err == nil {
		t.Fatal("want tag error")
	}
}

func assertEXE(t *testing.T, filt, plain string) {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("testdata", filt))
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", plain))
	if err != nil {
		t.Fatal(err)
	}
	in := packEXE(16*1024, src, uint32(len(want)))
	rd, err := NewReader(bytes.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(rd)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("got %x want %x", got, want)
	}
}

func packData(chunk uint32, data []byte) []byte {
	var b bytes.Buffer
	putU32(&b, chunk)
	putU32(&b, tagData)
	putU32(&b, uint32(len(data)))
	b.Write(data)
	return b.Bytes()
}

func packEXE(chunk uint32, filt []byte, outSize uint32) []byte {
	var b bytes.Buffer
	putU32(&b, chunk)
	putU32(&b, tagEXE)
	putU32(&b, outSize)
	putU32(&b, uint32(len(filt)))
	b.Write(filt)
	return b.Bytes()
}

func putU32(b *bytes.Buffer, v uint32) {
	var p [4]byte
	binary.LittleEndian.PutUint32(p[:], v)
	b.Write(p[:])
}
