package rzw

import (
	"bytes"
	"errors"
	"hash/crc32"
	"io"
	"testing"

	"github.com/lucasew/garotafitness/delta"
	"github.com/lucasew/garotafitness/dispack"
	"github.com/lucasew/garotafitness/internal/corpus"
	"github.com/lucasew/garotafitness/srep"
)

// fg-03.bin solid at 0x1F (rzwb). fg-04 4x4 inner packet is size+CM(.
var rzwbHead = []byte{
	0x6f, 0x04, 0x07, 0x01,
	'C', 'M', '(', 0x05, 0x06, 0x00, 0x00,
	0x33, 0xe3, 0x9c, 0x76, 0x53, 0x04, 0x07, 0x01, 0x00, 0x00,
}
var rzwHead = []byte{
	'C', 'M', '(', 0x05, 0x06, 0x00, 0x00,
	0xce, 0x2d, 0x9a, 0xe5, 0x45, 0xe2, 0x09, 0x00, 0x00, 0x00,
}

// fg-05 members after rzw → delta → dispack → srep, in archive order.
var fg05Members = []struct {
	size uint32
	crc  uint32
	path string
}{
	{613, 0xf2f30dfc, "mover/mover.bat"},
	{186, 0xf01e6380, "work/work/build01.bat"},
	{69660, 0xf1df1545, "work/work/fart.exe"},
	{108544, 0xf081f39d, "work/work/fgpack.exe"},
	{81920, 0xfd15612a, "work/work/run.exe"},
	{317952, 0xf7a6f4ee, "work/work/x.exe"},
	{38, 0xf82f659c, "work/Made by FitGirl.txt"},
	{712, 0xf01d85d3, "work/work/fitgirl01.txt"},
	{38, 0xf82f659c, "work/work/Made by FitGirl.txt"},
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
		{name: "short", in: bytes.NewReader([]byte("CM")), want: io.ErrUnexpectedEOF},
		{name: "shortCM", in: bytes.NewReader([]byte("CM(\x05\x06\x00\x00")), want: io.ErrUnexpectedEOF},
		{name: "arc", in: bytes.NewReader([]byte("ArC\x01xxxx")), want: errMagic},
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

func TestNewReaderTagged(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		in   []byte
	}{
		{name: "rzw", in: rzwHead},
		{name: "rzwb", in: rzwbHead},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rc, err := NewReader(bytes.NewReader(tt.in))
			if err != nil {
				t.Fatalf("NewReader(%s) err = %v", tt.name, err)
			}
			t.Cleanup(func() { rc.Close() })
			n, err := rc.Read(make([]byte, 8))
			if n != 0 || !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatalf("Read(%s) n=%d err=%v; want unexpected EOF", tt.name, n, err)
			}
		})
	}
}

func TestNewReaderVersion(t *testing.T) {
	t.Parallel()
	in := append([]byte(nil), rzwHead...)
	in[3] = 0
	rc, err := NewReader(bytes.NewReader(in))
	if rc != nil {
		t.Fatalf("reader = %T; want nil", rc)
	}
	if !errors.Is(err, errVersion) {
		t.Fatalf("err = %v; want %v", err, errVersion)
	}
}

func TestNewReaderCorpus(t *testing.T) {
	t.Parallel()
	f := corpus.File(t, "fg-03.bin")
	if _, err := f.Seek(0x1F, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	rc, err := NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rc.Close() })
	n, err := io.Copy(io.Discard, rc)
	if err != nil {
		t.Fatalf("Read n=%d err=%v", n, err)
	}
	if n == 0 {
		t.Fatal("empty corpus output")
	}
}

func TestFG05Header(t *testing.T) {
	t.Parallel()
	f := corpus.File(t, "fg-05.bin")
	if _, err := f.Seek(0x1F, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	rc, err := NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rc.Close() })
	rd, ok := rc.(*reader)
	if !ok {
		t.Fatalf("reader %T", rc)
	}
	if rd.hdr.prefix != 230566 || rd.hdr.indexOffset != 230542 {
		t.Fatalf("hdr %#v", rd.hdr)
	}
	buf := make([]byte, 8)
	n, err := rc.Read(buf)
	if err != nil {
		t.Fatalf("Read n=%d err=%v", n, err)
	}
	if n > 0 {
		t.Logf("decoded prefix %x", buf[:n])
	}
}

func TestFG05Pipeline(t *testing.T) {
	t.Parallel()
	raw := corpus.ReadFile(t, "fg-05.bin")
	solid, err := NewReader(bytes.NewReader(raw[31:]))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { solid.Close() })
	del, err := delta.NewReader(solid)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { del.Close() })
	dis, err := dispack.NewReader(del)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { dis.Close() })
	var dhead [16]byte
	dn, derr := io.ReadFull(dis, dhead[:])
	t.Logf("dispack head n=%d %x err=%v", dn, dhead[:dn], derr)
	sr, err := srep.NewReader(io.MultiReader(bytes.NewReader(dhead[:dn]), dis))
	if err != nil {
		if errors.Is(err, errCodec) {
			t.Fatalf("rzw kernel: %v", err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() { sr.Close() })
	plain, err := io.ReadAll(sr)
	if err != nil {
		t.Fatal(err)
	}
	off := 0
	matched := 0
	for _, m := range fg05Members {
		if off+int(m.size) > len(plain) {
			t.Fatalf("short solid at %s off=%d need=%d have=%d", m.path, off, m.size, len(plain)-off)
		}
		// Custom FreeArc polynomial recovered from the installer unarc DLL.
		got := crc32.Checksum(plain[off:off+int(m.size)], crc32.MakeTable(0x0895171b))
		if got != m.crc {
			t.Fatalf("%s crc=%08x want %08x", m.path, got, m.crc)
		}
		t.Logf("ok %8d %08x %s", m.size, m.crc, m.path)
		off += int(m.size)
		matched++
	}
	if matched != len(fg05Members) || off != len(plain) {
		t.Fatalf("matched %d members, consumed %d of %d bytes", matched, off, len(plain))
	}
}
