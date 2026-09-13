package rzw

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"io"
	"testing"
)

func framed(b []byte) []byte {
	h := []byte{byte(len(b)), byte(len(b) >> 8), byte(len(b) >> 16), 0, 0, 0, 0}
	crc := crc32.Update(crc32.ChecksumIEEE(h[:3]), crc32.IEEETable, b)
	binary.LittleEndian.PutUint32(h[3:], crc)
	return append(h, b...)
}

func indexUint(v uint64) []byte {
	var out []byte
	for v >= 128 {
		out = append(out, byte(v<<1)|1)
		v >>= 7
	}
	return append(out, byte(v<<1))
}

func TestArchiveIndex(t *testing.T) {
	// Stream 2 occupies two runs; the first has two independently framed
	// packets and is shorter than its later run, exercising a negative delta.
	p := bytes.Repeat([]byte{0xa5}, 8)
	q := bytes.Repeat([]byte{0x5a}, 24)
	body := append(framed(p), framed(p)...)
	body = append(body, framed(p)...)
	body = append(body, framed(q)...)
	idx := indexUint(3)
	idx = append(idx, indexUint(31<<4|2)...)
	idx = append(idx, indexUint(15<<4|0)...)
	idx = append(idx, indexUint(0<<4|8|2)...)
	src := append(append([]byte{}, body...), framed(idx)...)
	r := bytes.NewReader(src)
	a, err := readArchive(r, uint64(cmLen+len(body)))
	if err != nil {
		t.Fatal(err)
	}
	if len(a.streams[2]) != 3 || !bytes.Equal(a.streams[2][2], q) || len(a.streams[0]) != 1 || r.Len() != 0 {
		t.Fatalf("incorrect stream reconstruction: %#v, remaining=%d", a, r.Len())
	}
	for _, where := range []int{3, 7, len(body) + 3, len(src) - 1} {
		bad := bytes.Clone(src)
		bad[where] ^= 1
		if _, err := readArchive(bytes.NewReader(bad), uint64(cmLen+len(body))); err == nil {
			t.Fatalf("accepted corruption at %d", where)
		}
	}
	for cut := 0; cut < len(src); cut++ {
		if _, err := readArchive(bytes.NewReader(src[:cut]), uint64(cmLen+len(body))); err == nil {
			t.Fatalf("accepted truncation at %d", cut)
		}
	}
}

func TestIndexIntegerAcrossFrames(t *testing.T) {
	want := uint64(1)<<63 | 12345
	b := indexUint(want)
	r := indexReader{src: bytes.NewReader(append(framed(b[:2]), framed(b[2:])...))}
	got, err := r.uint()
	if err != nil || got != want {
		t.Fatalf("got %d, %v; want %d", got, err, want)
	}
	r = indexReader{src: bytes.NewReader(framed(bytes.Repeat([]byte{255}, 10)))}
	if _, err := r.uint(); err == nil {
		t.Fatal("accepted overflow")
	}
}

func TestRimWorldArchiveFrames(t *testing.T) {
	f := openCorpusFile(t, "fg-05.bin")
	if _, err := f.Seek(31, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	h, err := parseHeader(f)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readArchive(f, h.indexOffset)
	if err != nil {
		t.Fatal(err)
	}
	want := [8]int{50, 190, 992, 452, 228806}
	for i, n := range want {
		if n == 0 {
			if len(a.streams[i]) != 0 {
				t.Fatalf("unexpected stream %d", i)
			}
			continue
		}
		if len(a.streams[i]) != 1 || len(a.streams[i][0]) != n {
			t.Fatalf("stream %d: wrong packet lengths", i)
		}
	}
	pos, err := f.Seek(0, io.SeekCurrent)
	if err != nil || pos != 31+int64(h.prefix) {
		t.Fatalf("archive end %d, %v; want %d", pos, err, 31+h.prefix)
	}
}
