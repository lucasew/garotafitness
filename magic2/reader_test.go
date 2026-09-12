package magic2

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"testing"
)

func TestFG06(t *testing.T) {
	const corpus = "/media/downloads/TORRENTS/RimWorld [FitGirl Repack]/fg-06.bin"
	f, err := os.Open(corpus)
	if os.IsNotExist(err) {
		t.Skip("corpus not mounted")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	data := make([]byte, 93116)
	if _, err := f.ReadAt(data, 31); err != nil {
		t.Fatal(err)
	}
	r, err := NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	plain, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(plain) != 430889 {
		t.Fatalf("size %d", len(plain))
	}
	got := sha256.Sum256(plain)
	// Independent reconstruction: all five original archive CRCs match.
	if hex.EncodeToString(got[:]) != fg06SHA256 {
		t.Fatalf("output SHA256 %x", got)
	}
	for _, n := range []int{9, 14, 51, 55, len(data) - 1} {
		r, err := NewReader(bytes.NewReader(data[:n]))
		if err == nil {
			_, err = io.Copy(io.Discard, r)
			r.Close()
		}
		if err == nil {
			t.Errorf("accepted truncation at %d", n)
		}
	}
	corrupt := bytes.Clone(data)
	corrupt[55] ^= 0x80
	r, err = NewReader(bytes.NewReader(corrupt))
	if err == nil {
		_, err = io.Copy(io.Discard, r)
		r.Close()
	}
	if err == nil {
		t.Fatal("accepted corrupt frame length")
	}
}

const fg06SHA256 = "bc8242fbe8d79c30e3ccb08897f446ef95937c2ac5d5c38979182ba2a9417eab"
