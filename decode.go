package garotafitness

import (
	"io"

	"github.com/lucasew/garotafitness/magic2"
	"github.com/lucasew/garotafitness/mpzz"
	"github.com/lucasew/garotafitness/rzw"
	"github.com/lucasew/garotafitness/srep"
	"github.com/lucasew/garotafitness/storing"
)

// Decode wraps r with the atom's decompressor.
// The root package owns this switch. Child packages must not import
// it. 4x4 takes a func(io.Reader, name, params string) instead.
func Decode(r io.Reader, a Atom) (io.ReadCloser, error) {
	switch a.Algo {
	case AlgoStoring:
		return storing.NewReader(r)
	case AlgoSREP:
		return srep.NewReader(r)
	case AlgoMPZZ:
		return mpzz.NewReader(r)
	case AlgoMagic2:
		return magic2.NewReader(r)
	case AlgoRZW:
		return rzw.NewReader(r)
	default:
		return nil, unknownEncoderError(a)
	}
}
