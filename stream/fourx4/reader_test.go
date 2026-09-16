package fourx4

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"testing"
)

func TestParseInner(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, name, params string
	}{
		{"b128mb:rzw", "rzw", ""},
		{"b16mb:mpz", "mpz", ""},
		{"t4:i2:b8mb:lzma:8mb", "lzma", "8mb"},
		{"rzw", "rzw", ""},
		{"r99:rzw", "rzw", ""},
	}
	for _, tc := range cases {
		name, params, err := parseInner(tc.in)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if name != tc.name || params != tc.params {
			t.Fatalf("%q: got %q %q want %q %q", tc.in, name, params, tc.name, tc.params)
		}
	}
	if _, _, err := parseInner(""); err == nil {
		t.Fatal("want missing inner")
	}
	if _, _, err := parseInner("b128mb"); err == nil {
		t.Fatal("want missing inner")
	}
}

func TestNewReaderNil(t *testing.T) {
	t.Parallel()
	if _, err := NewReader(t.Context(), nil, "rzw", ident); err == nil {
		t.Fatal("want nil reader")
	}
	if _, err := NewReader(t.Context(), bytes.NewReader(nil), "rzw", nil); err == nil {
		t.Fatal("want nil inner")
	}
}

func TestStoredRoundTrip(t *testing.T) {
	t.Parallel()
	plain := []byte("hello 4x4 stored")
	var called bool
	inner := func(_ context.Context, r io.Reader, name, params string) (io.ReadCloser, error) {
		called = true
		return ident(t.Context(), r, name, params)
	}
	in := frameStored(plain)
	rd, err := NewReader(t.Context(), bytes.NewReader(in), "b128mb:rzw", inner)
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
	if called {
		t.Fatal("stored block must not call inner")
	}
}

func TestInnerCompressed(t *testing.T) {
	t.Parallel()
	plain := []byte("inner payload")
	var gotName, gotParams string
	inner := func(_ context.Context, r io.Reader, name, params string) (io.ReadCloser, error) {
		gotName, gotParams = name, params
		b, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		for i := range b {
			b[i] ^= 0x5a
		}
		return io.NopCloser(bytes.NewReader(b)), nil
	}
	comp := append([]byte(nil), plain...)
	for i := range comp {
		comp[i] ^= 0x5a
	}
	in := frameComp(uint32(len(plain)), comp)
	rd, err := NewReader(t.Context(), bytes.NewReader(in), "b16mb:mpz:q1", inner)
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
	if gotName != "mpz" || gotParams != "q1" {
		t.Fatalf("inner %q %q", gotName, gotParams)
	}
}

func TestManyBlocksOrder(t *testing.T) {
	t.Parallel()
	var framed bytes.Buffer
	var ver [4]byte
	framed.Write(ver[:])
	var want bytes.Buffer
	for i := 0; i < 8; i++ {
		plain := []byte(fmt.Sprintf("block-%02d-payload", i))
		want.Write(plain)
		comp := append([]byte(nil), plain...)
		for j := range comp {
			comp[j] ^= 0x5a
		}
		putU32(&framed, uint32(len(plain)))
		putU32(&framed, uint32(len(comp)))
		framed.Write(comp)
	}
	inner := func(_ context.Context, r io.Reader, _, _ string) (io.ReadCloser, error) {
		b, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		for i := range b {
			b[i] ^= 0x5a
		}
		return io.NopCloser(bytes.NewReader(b)), nil
	}
	rd, err := NewReader(t.Context(), bytes.NewReader(framed.Bytes()), "t4:rzw", inner)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(rd)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want.String() {
		t.Fatalf("got %q want %q", got, want.String())
	}
}

func TestParseThreads(t *testing.T) {
	t.Parallel()
	if n := parseThreads("t4:b8mb:rzw"); n != 4 {
		t.Fatalf("got %d", n)
	}
	if n := parseThreads("rzw"); n < 1 {
		t.Fatalf("got %d", n)
	}
}

func TestEmptyStream(t *testing.T) {
	t.Parallel()
	rd, err := NewReader(t.Context(), bytes.NewReader(nil), "rzw", ident)
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

func TestBadVersion(t *testing.T) {
	t.Parallel()
	in := []byte{1, 0, 0, 0}
	if _, err := NewReader(t.Context(), bytes.NewReader(in), "rzw", ident); err == nil {
		t.Fatal("want version error")
	}
}

func TestInnerError(t *testing.T) {
	t.Parallel()
	boom := func(context.Context, io.Reader, string, string) (io.ReadCloser, error) {
		return nil, fmt.Errorf("nope")
	}
	in := frameComp(4, []byte("xxxx"))
	rd, err := NewReader(t.Context(), bytes.NewReader(in), "rzw", boom)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadAll(rd); err == nil {
		t.Fatal("want inner error")
	}
}

func ident(_ context.Context, r io.Reader, _, _ string) (io.ReadCloser, error) {
	return io.NopCloser(r), nil
}

func frameStored(data []byte) []byte {
	var b bytes.Buffer
	var ver [4]byte
	b.Write(ver[:])
	putU32(&b, storedOut)
	putU32(&b, uint32(len(data)))
	b.Write(data)
	return b.Bytes()
}

func frameComp(outSize uint32, data []byte) []byte {
	var b bytes.Buffer
	var ver [4]byte
	b.Write(ver[:])
	putU32(&b, outSize)
	putU32(&b, uint32(len(data)))
	b.Write(data)
	return b.Bytes()
}

func putU32(b *bytes.Buffer, v uint32) {
	var p [4]byte
	binary.LittleEndian.PutUint32(p[:], v)
	b.Write(p[:])
}
