package lzma

import (
	"fmt"
	"io"

	ulzma "github.com/ulikunitz/xz/lzma"
)

// NewReader wraps an official LZMA stream as compress/gzip does.
func NewReader(r io.Reader) (io.ReadCloser, error) {
	if r == nil {
		return nil, fmt.Errorf("lzma: nil reader")
	}
	zr, err := ulzma.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("lzma: %w", err)
	}
	return io.NopCloser(zr), nil
}
