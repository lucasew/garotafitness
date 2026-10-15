package fgpack

import (
	"context"
	"encoding/binary"
	"fmt"
)

// Encoder keeps the call shape used when many small files are packed together.
type Encoder struct {
	closed bool
}

// NewEncoder prepares an encoder. There is no guest to compile.
func NewEncoder(ctx context.Context) (*Encoder, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &Encoder{}, nil
}

func (e *Encoder) Close(ctx context.Context) error {
	if e == nil {
		return nil
	}
	e.closed = true
	return nil
}

// Encode compresses one buffer with the given LZMA options.
func (e *Encoder) Encode(ctx context.Context, data []byte, o Options) ([]byte, error) {
	if e == nil || e.closed {
		return nil, fmt.Errorf("fgpack: closed encoder")
	}
	return EncodeWithOptions(ctx, data, o)
}

func encodeConfig(o Options) []byte {
	n := 20
	if o.Algo != 0 || o.BTMode != 0 || o.HashBytes != 0 || o.ReduceToSize {
		n = 36
	}
	config := make([]byte, n)
	for i, v := range []int{o.DictLog, o.FastBytes, o.LC, o.LP, o.PB} {
		binary.LittleEndian.PutUint32(config[4*i:], uint32(v))
	}
	if n >= 32 {
		hash := o.HashBytes
		if hash == 0 {
			hash = 4
		}
		binary.LittleEndian.PutUint32(config[20:], uint32(o.Algo))
		binary.LittleEndian.PutUint32(config[24:], uint32(o.BTMode))
		binary.LittleEndian.PutUint32(config[28:], uint32(hash))
	}
	return config
}
