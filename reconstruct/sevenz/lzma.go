package sevenz

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/lucasew/garotafitness/reconstruct/sevenz/lzma"
)

type lzmaOpts struct {
	dictLog, fb, lc, lp, pb int
	algo, btMode, hashBytes int
	reduce                  uint32
}

func fileOpts(reduce uint32) lzmaOpts {
	// 7-Zip -m0=lzma:x=4 is HC5 (algo 0, 5-byte hash).
	return lzmaOpts{dictLog: 19, fb: 32, lc: 3, lp: 0, pb: 2, algo: 0, btMode: 0, hashBytes: 5, reduce: reduce}
}

func headerOpts() lzmaOpts {
	// 7zHandlerOut.cpp SetHeaderMethod: LZMA BT2, level 5, fb=273, dict=1MiB.
	return lzmaOpts{dictLog: 20, fb: 273, lc: 3, lp: 0, pb: 2, algo: 1, btMode: 1, hashBytes: 2}
}

type lzmaEnc struct{ closed bool }

func newLzmaEnc(ctx context.Context) (*lzmaEnc, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &lzmaEnc{}, nil
}

func (e *lzmaEnc) Close(ctx context.Context) error {
	if e == nil {
		return nil
	}
	e.closed = true
	return nil
}

func (e *lzmaEnc) encode(ctx context.Context, data []byte, o lzmaOpts) ([]byte, error) {
	if e == nil || e.closed {
		return nil, fmt.Errorf("sevenz: rejected input")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("sevenz: %w", err)
	}
	n := 36
	if o.reduce != 0 {
		n = 40
	}
	config := make([]byte, n)
	for i, v := range []int{o.dictLog, o.fb, o.lc, o.lp, o.pb, o.algo, o.btMode, o.hashBytes} {
		binary.LittleEndian.PutUint32(config[4*i:], uint32(v))
	}
	if n >= 40 {
		binary.LittleEndian.PutUint32(config[32:], o.reduce)
	}
	out := lzma.Encode(data, config)
	if out == nil {
		return nil, fmt.Errorf("sevenz: rejected input")
	}
	if len(out) < 13 {
		return nil, fmt.Errorf("sevenz: short LZMA stream")
	}
	return out, nil
}
