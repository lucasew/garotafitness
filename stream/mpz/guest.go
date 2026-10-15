package mpz

import (
	"context"
	"encoding/binary"
)

func decodeWASM(ctx context.Context, src []byte, h Header) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(src) == 0 {
		return nil, errGuest
	}
	orig := h.Orig
	if orig == 0 {
		return nil, errGuest
	}
	buf := make([]byte, headerLen+len(src))
	binary.LittleEndian.PutUint32(buf[0:4], h.Version)
	binary.LittleEndian.PutUint32(buf[4:8], h.Orig)
	binary.LittleEndian.PutUint32(buf[8:12], h.Frames)
	binary.LittleEndian.PutUint32(buf[12:16], h.Extra)
	copy(buf[headerLen:], src)
	out, err := decodeLift(buf, orig)
	if err != nil {
		return nil, err
	}
	if uint32(len(out)) != orig {
		return nil, errGuest
	}
	return out, nil
}
