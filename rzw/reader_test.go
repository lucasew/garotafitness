package rzw

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"testing"

	"github.com/lucasew/garotafitness/delta"
	"github.com/lucasew/garotafitness/dispack"
	"github.com/lucasew/garotafitness/fourx4"
	"github.com/lucasew/garotafitness/srep"
)

const rimworld = `/media/downloads/TORRENTS/RimWorld [FitGirl Repack]`

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
		{name: "arc", in: bytes.NewReader([]byte("ArC\x01xxx")), want: errMagic},
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
	binary.LittleEndian.PutUint32(in[3:7], 0x00000100)
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
	f, err := os.Open(rimworld + "/fg-03.bin")
	if err != nil {
		t.Skip("corpus not mounted")
	}
	t.Cleanup(func() { f.Close() })
	if _, err := f.Seek(0x1F, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	rc, err := NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rc.Close() })
	n, err := rc.Read(make([]byte, 8))
	if err != nil && !errors.Is(err, errCodec) && !errors.Is(err, io.EOF) {
		t.Fatalf("Read n=%d err=%v", n, err)
	}
}

func TestFourX4InnerCorpus(t *testing.T) {
	t.Parallel()
	f, err := os.Open(rimworld + "/fg-04.bin")
	if err != nil {
		t.Skip("corpus not mounted")
	}
	t.Cleanup(func() { f.Close() })
	if _, err := f.Seek(0x1F, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	var gotName string
	rd, err := fourx4.NewReader(f, "b128mb:rzw", func(r io.Reader, name, _ string) (io.ReadCloser, error) {
		gotName = name
		return NewReader(r)
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rd.Close() })
	_, err = io.ReadAll(rd)
	if gotName != "rzw" {
		t.Fatalf("inner name %q", gotName)
	}
	if err != nil && !errors.Is(err, errCodec) {
		t.Fatalf("4x4 inner err = %v", err)
	}
}

func TestFG05Header(t *testing.T) {
	t.Parallel()
	f, err := os.Open(rimworld + "/fg-05.bin")
	if err != nil {
		t.Skip("corpus not mounted")
	}
	t.Cleanup(func() { f.Close() })
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
	if rd.hdr.prefix != 230566 || rd.hdr.packed != 230542 || rd.hdr.crc != 0xe77aa130 {
		t.Fatalf("hdr prefix=%d packed=%d crc=%#x", rd.hdr.prefix, rd.hdr.packed, rd.hdr.crc)
	}
	buf := make([]byte, 8)
	n, err := rc.Read(buf)
	if err != nil && !errors.Is(err, errCodec) && !errors.Is(err, io.EOF) {
		t.Fatalf("Read n=%d err=%v", n, err)
	}
	if n > 0 {
		t.Logf("decoded prefix %x", buf[:n])
	}
}

func TestFG05LegalChunkPipeline(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(rimworld + "/fg-05.bin")
	if err != nil {
		t.Skip("corpus not mounted")
	}
	packed := raw[0x1F+4+17 : 0x1F+4+17+230542]
	alts := []struct {
		name string
		fn   func([]byte, int) ([]byte, error)
	}{
		{"copy8", decodeAltCopy8},
		{"bin0", decodeAltBin0},
		{"s1u16", decodeAltS1u16},
		{"s1lane", decodeAltS1lane},
		{"t15d8", decodeAltT15Delta},
		{"t15d32", decodeAltT15Delta4},
		{"nochk", decodeAltNoChunk},
	}
	for _, a := range alts {
		dest, err := a.fn(packed, 1<<20)
		if err != nil {
			t.Logf("%s: %v", a.name, err)
			continue
		}
		if len(dest) < 16 {
			t.Logf("%s short %d", a.name, len(dest))
			continue
		}
		ch := uint32(dest[8]) | uint32(dest[9])<<8 | uint32(dest[10])<<16 | uint32(dest[11])<<24
		tag := uint32(dest[12]) | uint32(dest[13])<<8 | uint32(dest[14])<<16 | uint32(dest[15])<<24
		t.Logf("%s len=%d crc=%08x dest8=%08x tag=%08x dispack=%v head=%x",
			a.name, len(dest), crc32.ChecksumIEEE(dest), ch, tag, ch >= 1 && ch <= 8<<20, dest[:16])
		if len(dest) >= 28 {
			t.Logf("%s dest[12:28]=%x", a.name, dest[12:28])
		}
		if ch < 1 || ch > 8<<20 {
			continue
		}
		del, err := delta.NewReader(io.NopCloser(bytes.NewReader(dest)))
		if err != nil {
			t.Logf("%s delta: %v", a.name, err)
			continue
		}
		dis, err := dispack.NewReader(del)
		if err != nil {
			t.Logf("%s dispack: %v", a.name, err)
			continue
		}
		var dhead [16]byte
		dn, derr := io.ReadFull(dis, dhead[:])
		t.Logf("%s dispack head n=%d %x err=%v", a.name, dn, dhead[:dn], derr)
		if derr != nil {
			continue
		}
		sr, err := srep.NewReader(io.MultiReader(bytes.NewReader(dhead[:dn]), dis))
		if err != nil {
			t.Logf("%s srep: %v", a.name, err)
			continue
		}
		plain, err := io.ReadAll(sr)
		if err != nil {
			t.Logf("%s read: %v", a.name, err)
			continue
		}
		if len(plain) >= 613 {
			got := crc32.ChecksumIEEE(plain[:613])
			t.Logf("%s mover crc=%08x want f2f30dfc plain=%d", a.name, got, len(plain))
		} else {
			t.Logf("%s plain %d", a.name, len(plain))
		}
	}
}

func TestFG05Pipeline(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(rimworld + "/fg-05.bin")
	if err != nil {
		t.Skip("corpus not mounted")
	}
	packOff := 0x1F + 4 + 17
	packed := raw[packOff : packOff+230542]
	want := uint32(0xe77aa130)
	dest, err := decodeNative(packed, 1<<20)
	if err != nil {
		t.Fatalf("decodeNative: %v", err)
	}
	got := crc32.ChecksumIEEE(dest)
	nhead := 8
	if len(dest) < nhead {
		nhead = len(dest)
	}
	t.Logf("rzw dest %d crc %08x want %08x head=%x", len(dest), got, want, dest[:nhead])
	if len(dest) >= 8 {
		ds := uint32(dest[0]) | uint32(dest[1])<<8 | uint32(dest[2])<<16 | uint32(dest[3])<<24
		ts := uint32(dest[4]) | uint32(dest[5])<<8 | uint32(dest[6])<<16 | uint32(dest[7])<<24
		t.Logf("delta dataSize=%#x (%d) tableSize=%#x span=%d", ds, ds, ts, 8+3*int(ts)+int(ds))
	}
	if len(dest) >= 12 {
		ch := uint32(dest[8]) | uint32(dest[9])<<8 | uint32(dest[10])<<16 | uint32(dest[11])<<24
		t.Logf("dest[8:12]=%08x (%d) dispack=%v", ch, ch, ch >= 1 && ch <= 8<<20)
	}
	if len(dest) >= 28 {
		t.Logf("dest[12:28]=%x", dest[12:28])
	}
	if len(dest) >= 40 {
		t.Logf("dest[8:40]=%x", dest[8:40])
	}
	solid := io.NopCloser(bytes.NewReader(dest))
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
		got := crc32.ChecksumIEEE(plain[off : off+int(m.size)])
		if got != m.crc {
			t.Fatalf("%s crc=%08x want %08x", m.path, got, m.crc)
		}
		t.Logf("ok %8d %08x %s", m.size, m.crc, m.path)
		off += int(m.size)
		matched++
	}
	if matched < 2 {
		t.Fatalf("matched %d members", matched)
	}
}
