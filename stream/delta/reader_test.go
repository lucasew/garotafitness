package delta

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	lewpath "github.com/lewtec/lewkit/x/path"
)

func TestOfficialNoTable(t *testing.T) {
	t.Parallel()
	assertGolden(t, "small.delta", "small.plain")
}

func TestOfficialTable(t *testing.T) {
	t.Parallel()
	assertGolden(t, "table.delta", "table.plain")
}

func TestHandcraftedUndiff(t *testing.T) {
	t.Parallel()
	// N=2, both columns mutable, type=1<<2=4.
	// Original rows: 00 FF / 01 00. Diffed: 00 FF / 01 01.
	plain := []byte{0x00, 0xff, 0x01, 0x00}
	diffed := []byte{0x00, 0xff, 0x01, 0x01}
	in := packBlock(diffed, []tableDesc{{skip: 0, typ: 4, rows: 2}})
	rd, err := NewReader(bytes.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(rd)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("got %x want %x", got, plain)
	}
}

func TestUnreorder(t *testing.T) {
	t.Parallel()
	// N=3, immutable=[1,0,1], type=(1<<3)+1+4=13.
	// Original: A B C / D E F. Reordered: A C D F B E. No diff (doDiff false on imm).
	// doDiff is !immutable, so cols 1 is diffed. Two rows, first left as-is.
	// After reorder+diff of original A B C / D E F with doDiff [F,T,F]:
	//   diff row1: B stays... wait col1: E-B, others untouched -> A B C / D (E-B) F
	//   reorder: imm A,C then D,F then mut B, (E-B)
	// Easier: only unreorder, all immutable so no diff: type=(1<<3)+1+2+4=15
	// type 15: bits 0,1,2 set, sentinel bit 3. N=3 all immutable. unreorder no-op.
	plain := []byte{'A', 'B', 'C', 'D', 'E', 'F'}
	in := packBlock(plain, []tableDesc{{skip: 0, typ: 15, rows: 2}})
	rd, err := NewReader(bytes.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(rd)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("got %q", got)
	}

	// Mixed: immutable col0 only. type=(1<<3)+1=9. doDiff [F,T,T]
	// original: 10 01 02 / 10 03 05
	// diff:     10 01 02 / 10 02 03
	// reorder: imm 10,10 then mut 01 02, 02 03 -> 10 10 01 02 02 03
	reordered := []byte{0x10, 0x10, 0x01, 0x02, 0x02, 0x03}
	want := []byte{0x10, 0x01, 0x02, 0x10, 0x03, 0x05}
	in = packBlock(reordered, []tableDesc{{skip: 0, typ: 9, rows: 2}})
	rd, err = NewReader(bytes.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	got, err = io.ReadAll(rd)
	if err != nil {
		t.Fatal(err)
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

func assertGolden(t *testing.T, packed, plain string) {
	t.Helper()
	td := testdataRoot(t)
	in, err := lewpath.New(packed).ReadFile(td)
	if err != nil {
		t.Fatal(err)
	}
	want, err := lewpath.New(plain).ReadFile(td)
	if err != nil {
		t.Fatal(err)
	}
	rd, err := NewReader(bytes.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(rd)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("got %d bytes want %d", len(got), len(want))
	}
}

type tableDesc struct {
	skip, typ, rows uint32
}

func packBlock(data []byte, tabs []tableDesc) []byte {
	var b bytes.Buffer
	putU32(&b, uint32(len(data)))
	putU32(&b, uint32(len(tabs)*4))
	for _, t := range tabs {
		putU32(&b, t.skip)
	}
	for _, t := range tabs {
		putU32(&b, t.typ)
	}
	for _, t := range tabs {
		putU32(&b, t.rows)
	}
	b.Write(data)
	return b.Bytes()
}

func putU32(b *bytes.Buffer, v uint32) {
	var p [4]byte
	binary.LittleEndian.PutUint32(p[:], v)
	b.Write(p[:])
}
