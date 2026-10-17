// Package x5n applies HDIFF19 directory patches used by the installer's x5n step.
// The inner stream is uncompressed HDIFFSF20, applied in Go.
package x5n

import (
	"bytes"
	"context"
	"fmt"
)

// Apply reconstructs new files from old files and an HDIFF19 directory patch.
func Apply(ctx context.Context, oldFiles map[string][]byte, diff []byte) (map[string][]byte, error) {
	info, err := Parse(diff)
	if err != nil {
		return nil, err
	}
	var old bytes.Buffer
	old.Grow(int(info.OldRefSize))
	for _, idx := range info.OldRefs {
		if idx < 0 || idx >= len(info.OldPaths) {
			return nil, fmt.Errorf("x5n: old ref %d", idx)
		}
		name := info.OldPaths[idx]
		if isDirPath(name) {
			return nil, fmt.Errorf("x5n: old ref is a directory %q", name)
		}
		b, ok := oldFiles[name]
		if !ok {
			return nil, fmt.Errorf("x5n: missing old file %s", name)
		}
		old.Write(b)
	}
	if uint64(old.Len()) != info.OldRefSize {
		return nil, fmt.Errorf("x5n: old refs %d want %d", old.Len(), info.OldRefSize)
	}
	plain, err := applySF20(ctx, old.Bytes(), info.Inner)
	if err != nil {
		return nil, err
	}
	if uint64(len(plain)) != info.NewRefSize {
		return nil, fmt.Errorf("x5n: new refs %d want %d", len(plain), info.NewRefSize)
	}
	out := map[string][]byte{}
	off := 0
	for i, idx := range info.NewRefs {
		if idx < 0 || idx >= len(info.NewPaths) {
			return nil, fmt.Errorf("x5n: new ref %d", idx)
		}
		name := info.NewPaths[idx]
		n := int(info.NewRefSizes[i])
		if isDirPath(name) || n < 0 || off+n > len(plain) {
			return nil, fmt.Errorf("x5n: new file %q", name)
		}
		out[name] = bytes.Clone(plain[off : off+n])
		off += n
	}
	if off != len(plain) {
		return nil, fmt.Errorf("x5n: new file sizes %d want %d", off, len(plain))
	}
	return out, nil
}
