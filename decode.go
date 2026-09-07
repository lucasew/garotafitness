package garotafitness

import (
	"io"

	"github.com/lucasew/garotafitness/delta"
	"github.com/lucasew/garotafitness/dispack"
	"github.com/lucasew/garotafitness/fourx4"
	"github.com/lucasew/garotafitness/lzma"
	"github.com/lucasew/garotafitness/magic2"
	"github.com/lucasew/garotafitness/mpz"
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
	case AlgoLZMA:
		return lzma.NewReader(r)
	case AlgoSREP:
		return srep.NewReader(r)
	case AlgoDispack:
		return dispack.NewReader(r)
	case AlgoMPZZ:
		return mpzz.NewReader(r)
	case AlgoMagic2:
		return magic2.NewReader(r)
	case AlgoRZW:
		return rzw.NewReader(r)
	case AlgoMPZ:
		return mpz.NewReader(r)
	case AlgoDelta:
		return delta.NewReader(r)
	case Algo4x4:
		return fourx4.NewReader(r, a.Params, decodeInner)
	default:
		return nil, unknownEncoderError(a)
	}
}

func decodeInner(r io.Reader, name, params string) (io.ReadCloser, error) {
	return Decode(r, Atom{Algo: ParseAlgo(name), Params: params})
}
