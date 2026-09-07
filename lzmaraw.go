package garotafitness

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/lucasew/garotafitness/lzma"
)

// rawLZMA1 decodes a FreeArc lzma:mfbt4:d1m block (no .lzma header on disk).
func rawLZMA1(blob []byte, orig int) ([]byte, error) {
	var hdr [13]byte
	hdr[0] = 0x5d // lc=3 lp=0 pb=2
	binary.LittleEndian.PutUint32(hdr[1:5], 1<<20)
	binary.LittleEndian.PutUint64(hdr[5:13], uint64(orig))
	r, err := lzma.NewReader(io.MultiReader(bytes.NewReader(hdr[:]), bytes.NewReader(blob)))
	if err != nil {
		return nil, fmt.Errorf("lzma raw: %w", err)
	}
	defer r.Close()
	out := make([]byte, orig)
	if _, err := io.ReadFull(r, out); err != nil {
		return nil, fmt.Errorf("lzma raw: %w", err)
	}
	return out, nil
}
