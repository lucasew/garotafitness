// Package x2 applies the sparse replacement records used by the installer.
package x2

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Apply replaces byte ranges and sets the final length specified by patch.
// The format is a little-endian uint64 final length followed by records:
// uint64 offset, uint8 length (zero means 256), then the replacement bytes.
// This layout follows the static x2 patch loop at VA 0x401229..0x401354.
func Apply(old, patch []byte) ([]byte, error) {
	if len(patch) < 8 {
		return nil, io.ErrUnexpectedEOF
	}
	size := binary.LittleEndian.Uint64(patch)
	if size > 1<<30 {
		return nil, fmt.Errorf("x2: target exceeds memory limit")
	}
	out := make([]byte, int(size))
	copy(out, old)
	for p := patch[8:]; len(p) > 0; {
		if len(p) < 9 {
			return nil, io.ErrUnexpectedEOF
		}
		off := binary.LittleEndian.Uint64(p)
		n := int(p[8])
		if n == 0 {
			n = 256
		}
		p = p[9:]
		if len(p) < n {
			return nil, io.ErrUnexpectedEOF
		}
		if off > size || uint64(n) > size-off {
			return nil, fmt.Errorf("x2: replacement outside target")
		}
		copy(out[int(off):], p[:n])
		p = p[n:]
	}
	return out, nil
}
