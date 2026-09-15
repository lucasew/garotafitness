package pref

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func TestNewReader(t *testing.T) {
	t.Parallel()
	_, err := NewReader(nil)
	if !errors.Is(err, errNil) {
		t.Fatalf("nil: %v", err)
	}
	_, err = NewReader(bytes.NewReader(nil))
	if err == nil {
		t.Fatal("accepted empty")
	}
	_, err = NewReader(bytes.NewReader([]byte("ArC\x01xxxx")))
	if err == nil {
		t.Fatal("accepted non-PCF")
	}
}

func testdataPlain(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/plain.bin")
	if err != nil {
		t.Skip(err)
	}
	return b
}

func testdataPCF(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/plain.pcf")
	if err != nil {
		t.Skip(err)
	}
	return b
}

func TestRoundTrip(t *testing.T) {
	if len(guestWASM) < 8 {
		t.Skip("prefdec.wasm not built")
	}
	plain := testdataPlain(t)
	pcf := testdataPCF(t)
	rc, err := NewReader(bytes.NewReader(pcf))
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if err := rc.Close(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("got %d bytes %q; want %d %q", len(got), got, len(plain), plain)
	}
}

func TestInstantiateGuest(t *testing.T) {
	if len(guestWASM) < 8 {
		t.Skip("no wasm")
	}
	_, err := NewReader(bytes.NewReader([]byte("PCF\x00\x04\x08\x00")))
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "is not exported") {
		t.Fatal(err)
	}
}
