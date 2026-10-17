// Package tor decodes FreeArc Tornado streams (method tor).
package tor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
)

var (
	errNil    = errors.New("tor: nil reader")
	errGuest  = errors.New("tor: guest")
	errFormat = errors.New("tor: format")
)

// NewReader wraps an official Tornado stream as compress/gzip does.
func NewReader(ctx context.Context, r io.Reader) (io.ReadCloser, error) {
	if r == nil {
		return nil, errNil
	}
	in, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("tor: %w", err)
	}
	out, err := decode(ctx, in)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(out)), nil
}

func decode(ctx context.Context, in []byte) ([]byte, error) {
	if len(in) == 0 {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Same output ceiling the wasm guest was given.
	limit := len(in)*8 + 1<<20
	if limit < 4<<20 {
		limit = 4 << 20
	}
	out, err := decompress(ctx, in, limit)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, errGuest
	}
	if len(out) == 0 {
		return nil, errGuest
	}
	return out, nil
}
