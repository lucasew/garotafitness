package mpzz

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
)

// DecodeGuest decodes one OGGRE buffer. dstCap is the output ceiling;
// when the next record does not fit, the ceiling doubles and the decode
// starts over, up to 1 GiB.
func DecodeGuest(ctx context.Context, src []byte, dstCap int) ([]byte, error) {
	if len(src) == 0 {
		return nil, errGuest
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if dstCap < len(src)*2 {
		dstCap = len(src) * 2
	}
	if dstCap < 64<<20 {
		dstCap = 64 << 20
	}
	if dstCap > 1<<30 {
		dstCap = 1 << 30
	}
	for {
		out, rc, err := decodeAll(ctx, src, dstCap)
		if err != nil {
			return nil, err
		}
		if rc == -2 {
			if dstCap >= 1<<30 {
				return nil, fmt.Errorf("%w: dest too small", errGuest)
			}
			dstCap *= 2
			if dstCap > 1<<30 {
				dstCap = 1 << 30
			}
			continue
		}
		if rc <= 0 {
			return nil, fmt.Errorf("%w: decode %d", errGuest, rc)
		}
		return out, nil
	}
}

// decodeAll returns the decoded bytes and a kernel-style code: a positive
// length, 0 when the buffer is not a complete decode, or -2 when dcap
// cannot hold the next record. A canceled context is returned as-is.
func decodeAll(ctx context.Context, src []byte, dcap int) ([]byte, int32, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	h, err := parseHeader(bytes.NewReader(src))
	if err != nil {
		return nil, 0, nil
	}
	d, err := newOGGREDecoder(src[7:], h)
	if err != nil {
		return nil, 0, nil
	}
	if dcap < 0 {
		dcap = 0
	}
	var out []byte
	for {
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}
		rec, err := d.next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, 0, nil
		}
		if dcap < len(out) || len(rec) > dcap-len(out) {
			return nil, -2, nil
		}
		out = append(out, rec...)
	}
	return out, int32(len(out)), nil
}
