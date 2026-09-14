// Package xdelta applies VCDIFF patches with the official xdelta3 decoder.
package xdelta

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"

	"github.com/lucasew/garotafitness/internal/wasmrun"
)

//go:embed xdelta.wasm
var guest []byte

// Apply reconstructs a file and checks each VCDIFF window's embedded checksum.
func Apply(ctx context.Context, old, diff []byte) ([]byte, error) {
	n, err := targetSize(diff)
	if err != nil {
		return nil, err
	}
	out, err := wasmrun.Bytes(ctx, guest, "xdelta_apply", n, old, diff)
	if err != nil {
		return nil, err
	}
	if len(out) != n {
		return nil, fmt.Errorf("xdelta: target size mismatch")
	}
	return out, nil
}

// Read the window envelopes to bound allocation; the official decoder validates
// instructions, source ranges, addresses, and Adler32 checksums inside them.
func targetSize(diff []byte) (int, error) {
	if len(diff) < 5 || !bytes.Equal(diff[:4], []byte{0xd6, 0xc3, 0xc4, 0}) || diff[4]&^4 != 0 {
		return 0, fmt.Errorf("xdelta: unsupported VCDIFF header")
	}
	app := diff[4]&4 != 0
	diff = diff[5:]
	read := func(p *[]byte) (int, error) {
		n := 0
		for i := 0; i < 5; i++ {
			if len(*p) == 0 {
				return 0, io.ErrUnexpectedEOF
			}
			b := (*p)[0]
			*p = (*p)[1:]
			n = n<<7 | int(b&127)
			if n > 1<<30 {
				return 0, fmt.Errorf("xdelta: size exceeds memory limit")
			}
			if b < 128 {
				return n, nil
			}
		}
		return 0, fmt.Errorf("xdelta: invalid integer")
	}
	if app {
		n, err := read(&diff)
		if err != nil {
			return 0, err
		}
		if n > len(diff) {
			return 0, io.ErrUnexpectedEOF
		}
		diff = diff[n:]
	}
	total := 0
	for len(diff) > 0 {
		flags := diff[0]
		diff = diff[1:]
		if flags&^7 != 0 || flags&3 == 3 {
			return 0, fmt.Errorf("xdelta: invalid window flags")
		}
		if flags&3 != 0 {
			for i := 0; i < 2; i++ {
				if _, err := read(&diff); err != nil {
					return 0, err
				}
			}
		}
		n, err := read(&diff)
		if err != nil {
			return 0, err
		}
		if n > len(diff) {
			return 0, io.ErrUnexpectedEOF
		}
		window := diff[:n]
		diff = diff[n:]
		size, err := read(&window)
		if err != nil {
			return 0, err
		}
		if size > (1<<30)-total {
			return 0, fmt.Errorf("xdelta: target exceeds memory limit")
		}
		total += size
	}
	if total == 0 {
		return 0, fmt.Errorf("xdelta: empty target")
	}
	return total, nil
}
