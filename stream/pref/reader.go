// Package pref restores official Precomp PCF streams (FreeArc method pref).
//
// unpackcmd is `precomp.exe -ostdout -r stdin`. Restore follows
// recompress_file from schnaader/precomp-cpp 0.4.8.
package pref

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
)

var (
	errNil   = errors.New("pref: nil reader")
	errGuest = errors.New("pref: guest")
)

// NewReader wraps an official PCF stream as compress/gzip does.
func NewReader(ctx context.Context, r io.Reader) (io.ReadCloser, error) {
	if r == nil {
		return nil, errNil
	}
	in, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("pref: %w", err)
	}
	if len(in) < 7 || string(in[:3]) != "PCF" {
		return nil, fmt.Errorf("pref: bad magic")
	}
	out, err := restore(ctx, in)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(out)), nil
}
