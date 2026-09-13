// Package fgpack reproduces the LZMA SDK 19.00 stream used to rebuild bundles.
package fgpack

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/lucasew/garotafitness/internal/wasmrun"
)

//go:embed fgpack.wasm
var guest []byte

// Encode uses the installer's d19, fb64, lc3, lp0, pb2 parameters. The stream
// includes the LZMA-alone header with the known uncompressed size and no EOS.
func Encode(ctx context.Context, data []byte) ([]byte, error) {
	if len(data) > 512<<20 {
		return nil, fmt.Errorf("fgpack: input exceeds memory limit")
	}
	return wasmrun.Bytes(ctx, guest, "fgpack_encode", len(data)+len(data)/3+65536, data)
}
