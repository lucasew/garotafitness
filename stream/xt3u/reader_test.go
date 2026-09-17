package xt3u

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/lucasew/garotafitness/internal/corpus"
	"github.com/lucasew/garotafitness/stream/magic2"
	"github.com/lucasew/garotafitness/stream/srep"
)

func TestNewReaderNilShortBadMagic(t *testing.T) {
	t.Parallel()
	if _, err := NewReader(t.Context(), nil); !errors.Is(err, errNil) {
		t.Fatalf("nil: %v", err)
	}
	if _, err := NewReader(t.Context(), bytes.NewReader(nil)); err == nil {
		t.Fatal("want short error")
	}
	if _, err := NewReader(t.Context(), bytes.NewReader([]byte("XXXX"))); !errors.Is(err, errBadMagic) {
		t.Fatalf("magic: %v", err)
	}
	if _, err := NewReader(t.Context(), bytes.NewReader([]byte("XTL"))); err == nil {
		t.Fatal("want short magic")
	}
}

func TestNewReaderTailOnly(t *testing.T) {
	t.Parallel()
	plain := []byte("hello xt3u")
	r, err := NewReader(t.Context(), bytes.NewReader(tailSolid(plain)))
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

func TestNewReaderLZ4HC(t *testing.T) {
	t.Parallel()
	if len(guestWASM) == 0 {
		t.Skip("xt3udec.wasm not built")
	}
	raw := bytes.Repeat([]byte("Songs of Conquest "), 64)
	g, err := openGuest(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { g.Close() })
	comp, err := g.compressHC(raw, 12, 0)
	if err != nil {
		t.Fatal(err)
	}
	opt := int32(1 | (12 << 3))
	r, err := NewReader(t.Context(), bytes.NewReader(lz4hcSolid(raw, comp, opt)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, comp) {
		t.Fatalf("restored %d want %d", len(got), len(comp))
	}
}

func TestParseHeaderSOC(t *testing.T) {
	src := afterMagic2SREP(t)
	h, err := parseHeader(src)
	if err != nil {
		t.Fatal(err)
	}
	if h.Method != "unity:lz4hc:l12" {
		t.Fatalf("method %q", h.Method)
	}
	if h.Depth != 3 || h.Compressed != 0 || h.StoreDD != -1 {
		t.Fatalf("hdr %+v", h)
	}
	if len(h.Resources) != 1 || h.Resources[0].Name != "gk.key" || len(h.Resources[0].Data) != 32 {
		t.Fatalf("resources %+v", h.Resources)
	}
	if len(h.Dups) != 7 {
		t.Fatalf("dups %d", len(h.Dups))
	}
}

func TestNewReaderCorpusHead(t *testing.T) {
	src := afterMagic2SREP(t)
	r, err := NewReader(t.Context(), src)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	head := make([]byte, 32)
	if _, err := io.ReadFull(r, head); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(head, []byte("<configuration>\n")) {
		t.Fatalf("first bytes %q", head)
	}
}

func afterMagic2SREP(t *testing.T) io.Reader {
	t.Helper()
	f := corpus.FileEnv(t, "GAROTAFITNESS_CORPUS_SOC", "fg-03.bin")
	if _, err := f.Seek(31, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	m, err := magic2.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	s, err := srep.NewReader(t.Context(), m)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func tailSolid(plain []byte) []byte {
	var b bytes.Buffer
	b.WriteString(XTL0)
	putI32(&b, 1)  // depth
	b.WriteByte(0) // method
	putI32(&b, 0)  // resources
	putI32(&b, -2) // storeDD
	b.WriteByte(0) // compressed
	putI32(&b, 0)  // extra resources
	putI32(&b, 0)  // streamCount
	putI64(&b, 0)  // blockSize
	putU32(&b, uint32(len(plain)))
	b.Write(plain)
	putI32(&b, 0) // extra resources before terminator
	putI32(&b, int32(-1<<31))
	return b.Bytes()
}

func lz4hcSolid(raw, comp []byte, opt int32) []byte {
	var b bytes.Buffer
	b.WriteString(XTL0)
	putI32(&b, 1)
	b.WriteByte(byte(len("lz4hc:l12")))
	b.WriteString("lz4hc:l12")
	putI32(&b, 0)
	putI32(&b, -2)
	b.WriteByte(0)
	putI32(&b, 0)
	putI32(&b, 1) // one stream
	putI64(&b, int64(len(raw)))
	// TStreamHeader packed 18 bytes
	var h [streamHeaderSize]byte
	h[0] = kindDefault
	binary.LittleEndian.PutUint32(h[1:5], uint32(len(comp)))
	binary.LittleEndian.PutUint32(h[5:9], uint32(len(raw)))
	h[13] = codecLZ4
	binary.LittleEndian.PutUint32(h[14:18], uint32(opt))
	b.Write(h[:])
	b.Write(raw)
	putU32(&b, 0) // per-stream tail
	putU32(&b, 0) // final tail
	putI32(&b, 0) // extra resources before terminator
	putI32(&b, int32(-1<<31))
	return b.Bytes()
}

func putI32(b *bytes.Buffer, v int32) {
	var p [4]byte
	binary.LittleEndian.PutUint32(p[:], uint32(v))
	b.Write(p[:])
}

func putU32(b *bytes.Buffer, v uint32) {
	var p [4]byte
	binary.LittleEndian.PutUint32(p[:], v)
	b.Write(p[:])
}

func putI64(b *bytes.Buffer, v int64) {
	var p [8]byte
	binary.LittleEndian.PutUint64(p[:], uint64(v))
	b.Write(p[:])
}

func TestContainsToken(t *testing.T) {
	t.Parallel()
	if !containsToken("unity:lz4hc:l12", "lz4hc") {
		t.Fatal("lz4hc")
	}
	if containsToken("unity:lz4hc:l12", "lz4h") {
		t.Fatal("partial")
	}
	if !strings.Contains("unity:lz4hc:l12", "lz4") {
		t.Fatal("setup")
	}
}
