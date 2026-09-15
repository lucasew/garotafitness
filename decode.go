package garotafitness

import (
	"io"

	"github.com/lucasew/garotafitness/stream/delta"
	"github.com/lucasew/garotafitness/stream/dispack"
	"github.com/lucasew/garotafitness/stream/fourx4"
	"github.com/lucasew/garotafitness/stream/lzma"
	"github.com/lucasew/garotafitness/stream/magic2"
	"github.com/lucasew/garotafitness/stream/mpz"
	"github.com/lucasew/garotafitness/stream/mpzz"
	"github.com/lucasew/garotafitness/stream/pref"
	"github.com/lucasew/garotafitness/stream/rzw"
	"github.com/lucasew/garotafitness/stream/srep"
	"github.com/lucasew/garotafitness/stream/storing"
	"github.com/lucasew/garotafitness/stream/xt3u"
)

// Decode wraps r with the atom's decompressor.
// The root package owns this switch. Child packages must not import
// it. 4x4 takes a func(io.Reader, name, params string) instead.
func Decode(r io.Reader, a Atom) (io.ReadCloser, error) {
	switch a.Algo {
	case Algo4x4:
		return fourx4.NewReader(r, a.Params, decodeInner)
	case AlgoSREP:
		return srep.NewReader(r)
	case AlgoLZMA:
		return lzma.NewReader(r)
	case AlgoStoring:
		return storing.NewReader(r)
	case AlgoDelta:
		return delta.NewReader(r)
	case AlgoDispack:
		return dispack.NewReader(r)
	case AlgoMPZZ:
		return mpzz.NewReader(r)
	case AlgoMPZ:
		return mpz.NewReader(r)
	case AlgoRZW:
		return rzw.NewReader(r)
	case AlgoMagic2:
		return magic2.NewReader(r)
	case AlgoPref:
		return pref.NewReader(r)
	case AlgoXT3U:
		return xt3u.NewReader(r)
	default:
		return nil, unknownEncoderError(a)
	}
}

func decodeInner(r io.Reader, name, params string) (io.ReadCloser, error) {
	return Decode(r, Atom{Algo: ParseAlgo(name), Params: params})
}
