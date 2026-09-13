package mpz

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"testing"
)

func TestRangeCodedLiterals(t *testing.T) {
	// Independently encoded literal run: selector=1, a/b/c, escape=256,
	// exhausted-output selector=1, then the five range-coder flush bytes.
	data, err := hex.DecodeString("05040501030000000000000000000000009856c49e560800")
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	got, err := io.ReadAll(r)
	if err != nil || string(got) != "abc" {
		t.Fatalf("%q %v", got, err)
	}
	for n := 16; n < len(data); n++ {
		r, err := NewReader(bytes.NewReader(data[:n]))
		if err != nil {
			t.Fatal(err)
		}
		_, err = io.ReadAll(r)
		r.Close()
		if err == nil {
			t.Fatalf("accepted truncation at %d", n)
		}
	}
}

func TestComplementedLiterals(t *testing.T) {
	data := append(frameHead(version5450, 0, 0, 0)[:4], []byte{0, 255, 128, 127}...)
	r, err := NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(r)
	if err != nil || !bytes.Equal(got, []byte{255, 0, 127, 128}) {
		t.Fatalf("%x %v", got, err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Read(make([]byte, 1)); !errors.Is(err, errClosed) {
		t.Fatal(err)
	}
}

func TestOutputBound(t *testing.T) {
	for _, n := range []uint32{0, maxBlock + 1, ^uint32(0)} {
		if _, err := NewReader(bytes.NewReader(frameHead(version5451, n, 0, 0))); err == nil {
			t.Fatalf("accepted size %d", n)
		}
	}
}

func TestZeroLengthReadDoesNotDecode(t *testing.T) {
	r, err := NewReader(bytes.NewReader(frameHead(version5451, 3, 0, 0)))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if n, err := r.Read(nil); n != 0 || err != nil {
		t.Fatalf("%d %v", n, err)
	}
	if _, err := r.Read(make([]byte, 1)); err == nil {
		t.Fatal("accepted missing payload")
	}
}
