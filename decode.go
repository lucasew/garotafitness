package garotafitness

import (
	"io"

	"github.com/lucasew/garotafitness/storing"
)

// Decode wraps r with the atom's decompressor.
// The root package owns this switch. Child packages must not import
// it. 4x4 takes a func(io.Reader, name, params string) instead.
func Decode(r io.Reader, a Atom) (io.ReadCloser, error) {
	switch a.Algo {
	case AlgoStoring:
		return storing.NewReader(r)
	default:
		return nil, unknownEncoderError(a)
	}
}
