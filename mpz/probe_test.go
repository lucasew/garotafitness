//go:build reconstruction

package mpz

import (
	"bytes"
	"context"
	"github.com/lucasew/garotafitness/internal/wasmrun"
	"github.com/lucasew/garotafitness/srep"
	"hash/crc32"
	"io"
	"os"
	"testing"
)

func TestLiftedBlock(t *testing.T) {
	code, err := os.ReadFile("/tmp/gf-re/mpz-lifted.wasm")
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile("/tmp/gf-re/ost-first.mpz")
	if err != nil {
		t.Fatal(err)
	}
	out, err := wasmrun.Bytes(context.Background(), code, "mpz_decode", 16<<20, src)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 16<<20 {
		t.Fatalf("decoded %d", len(out))
	}
	if err := os.WriteFile("/tmp/gf-re/ost-wasm.srep", out, 0600); err != nil {
		t.Fatal(err)
	}
	checkFirstMP3(t, out)
}
func TestNativeBlock(t *testing.T) {
	b, err := os.ReadFile("/tmp/gf-re/ost-first.srep")
	if err != nil {
		t.Fatal(err)
	}
	checkFirstMP3(t, b)
}
func checkFirstMP3(t *testing.T, b []byte) {
	t.Helper()
	r, err := srep.NewReader(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	first := make([]byte, firstMP3Size)
	if _, err := io.ReadFull(r, first); err != nil {
		t.Fatal(err)
	}
	got := crc32.Checksum(first, crc32.MakeTable(0x0895171b))
	if got != firstMP3CRC {
		t.Fatalf("CRC %08x want %08x", got, firstMP3CRC)
	}
	t.Logf("first MP3 CRC verified: %08x", got)
}
