package garotafitness

import (
	"io"

	"github.com/lucasew/garotafitness/storing"
)

// openDecoder wraps r with the atom's decompressor.
func openDecoder(r io.Reader, a Atom) (io.ReadCloser, error) {
	switch a.Algo {
	case AlgoStoring:
		return storing.NewReader(r)
	default:
		return nil, unknownEncoderError(a)
	}
}
