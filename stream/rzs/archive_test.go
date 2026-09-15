package rzs

import (
	"io"
	"testing"

	"github.com/lucasew/garotafitness/internal/corpus"
)

func TestStdioIndexSize(t *testing.T) {
	for _, tt := range []struct {
		name   string
		packed uint64
		index  uint64
	}{
		{name: "fg-01.bin", packed: 0x372ad910, index: 0x372ad8fa},
		{name: "fg-02.bin", packed: 0x1dde4b95, index: 0x1dde4b7f},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := corpus.FileEnv(t, "GAROTAFITNESS_CORPUS_SOC", tt.name)
			if _, err := f.Seek(31+16, io.SeekStart); err != nil {
				t.Fatal(err)
			}
			off, n, err := parseCM(f)
			if err != nil {
				t.Fatal(err)
			}
			if off != tt.index || n != 20 {
				t.Fatalf("index=%#x consumed=%d", off, n)
			}
			idxN := tt.packed - tt.index
			if idxN != 22 {
				t.Fatalf("index size %d; want 22", idxN)
			}
			idx := make([]byte, idxN)
			if _, err := io.ReadFull(f, idx); err != nil {
				t.Fatal(err)
			}
			fn := int(idx[0]) | int(idx[1])<<8 | int(idx[2])<<16
			if 7+fn != int(idxN) {
				t.Fatalf("index frame %d+%d; want %d", 7, fn, idxN)
			}
		})
	}
}
