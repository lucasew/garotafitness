package x5n

import (
	"context"
	"errors"
	"fmt"
)

const (
	sf20Magic       = "HDIFFSF20"
	sf20StepSafe    = 16 << 20
	sf20MaxOut      = 2 << 30
	sf20SignTagBits = 1
)

var errSF20 = errors.New("x5n: corrupt HDIFFSF20 patch")

// applySF20 applies an uncompressed HDIFFSF20 single-stream diff.
// Behavior follows third_party/hdiffpatch4 patch_single_compressed_diff with
// compressedSize 0 and the bounds checks compiled into patch.c.
func applySF20(ctx context.Context, old, diff []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out, err := applySF20Bytes(ctx, old, diff)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, errSF20
	}
	return out, nil
}

type sf20Head struct {
	newSize      uint64
	oldSize      uint64
	coverCount   uint64
	stepMemSize  uint64
	uncompressed uint64
	compressed   uint64
	bodyAt       int
}

func applySF20Bytes(ctx context.Context, old, diff []byte) ([]byte, error) {
	head, err := readSF20(diff)
	if err != nil {
		return nil, err
	}
	if head.oldSize != uint64(len(old)) || head.compressed != 0 {
		return nil, errSF20
	}
	if head.compressed > head.uncompressed {
		return nil, errSF20
	}
	limit := uint64(sf20StepSafe)
	if head.newSize >= sf20StepSafe {
		limit = head.newSize
	}
	if head.stepMemSize > limit || head.stepMemSize > head.uncompressed {
		return nil, errSF20
	}
	if head.newSize > sf20MaxOut {
		return nil, errSF20
	}
	if uint64(len(diff)-head.bodyAt) < head.uncompressed {
		return nil, errSF20
	}
	body := diff[head.bodyAt : head.bodyAt+int(head.uncompressed)]
	out := make([]byte, head.newSize)
	var lastOld, lastNew uint64
	wrote := 0
	rle := rle0{need0: true}
	for head.coverCount > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		coverSize, rest, err := unpackUInt(body)
		if err != nil {
			return nil, errSF20
		}
		rleSize, rest, err := unpackUInt(rest)
		if err != nil {
			return nil, errSF20
		}
		if coverSize > head.stepMemSize || rleSize > head.stepMemSize || coverSize+rleSize > head.stepMemSize {
			return nil, errSF20
		}
		if uint64(len(rest)) < coverSize+rleSize {
			return nil, errSF20
		}
		covers := rest[:coverSize]
		rle.reset(rest[coverSize : coverSize+rleSize])
		body = rest[coverSize+rleSize:]
		if len(covers) == 0 {
			return nil, errSF20
		}
		for len(covers) > 0 {
			if head.coverCount == 0 || len(covers) == 0 {
				return nil, errSF20
			}
			sign := covers[0] >> 7
			incOld, next, err := unpackTag(covers, sf20SignTagBits)
			if err != nil {
				return nil, errSF20
			}
			incNew, next, err := unpackUInt(next)
			if err != nil {
				return nil, errSF20
			}
			length, next, err := unpackUInt(next)
			if err != nil {
				return nil, errSF20
			}
			covers = next
			var oldPos uint64
			if sign == 0 {
				if lastOld > ^uint64(0)-incOld {
					return nil, errSF20
				}
				oldPos = lastOld + incOld
			} else if incOld > lastOld {
				return nil, errSF20
			} else {
				oldPos = lastOld - incOld
			}
			if lastNew > ^uint64(0)-incNew {
				return nil, errSF20
			}
			newPos := lastNew + incNew
			head.coverCount--
			if newPos < lastNew {
				return nil, errSF20
			}
			gap := newPos - lastNew
			if gap > uint64(len(body)) || uint64(wrote) > sf20MaxOut-gap {
				return nil, errSF20
			}
			if uint64(wrote)+gap > head.newSize {
				return nil, errSF20
			}
			copy(out[wrote:wrote+int(gap)], body[:int(gap)])
			body = body[int(gap):]
			wrote += int(gap)
			if length == 0 {
				if head.coverCount != 0 {
					return nil, errSF20
				}
			} else {
				if oldPos > uint64(len(old)) || length > uint64(len(old))-oldPos {
					return nil, errSF20
				}
				if uint64(wrote) > head.newSize || length > head.newSize-uint64(wrote) {
					return nil, errSF20
				}
				dst := out[wrote : wrote+int(length)]
				copy(dst, old[int(oldPos):int(oldPos)+int(length)])
				if err := rle.add(dst); err != nil {
					return nil, errSF20
				}
				wrote += int(length)
			}
			lastOld = oldPos + length
			lastNew = newPos + length
			if lastOld < oldPos || lastNew < newPos {
				return nil, errSF20
			}
		}
	}
	if len(body) != 0 || uint64(wrote) != head.newSize {
		return nil, errSF20
	}
	return out, nil
}

func readLimited(b []byte, end byte, max int) (string, []byte, error) {
	n := max
	if n > len(b) {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if b[i] == end {
			return string(b[:i]), b[i+1:], nil
		}
	}
	return "", b, errSF20
}

func readSF20(diff []byte) (sf20Head, error) {
	var head sf20Head
	name, rest, err := readLimited(diff, '&', 260)
	if err != nil || name != sf20Magic {
		return head, errSF20
	}
	if _, rest, err = readLimited(rest, 0, 260); err != nil {
		return head, errSF20
	}
	fields := []*uint64{
		&head.newSize, &head.oldSize, &head.coverCount,
		&head.stepMemSize, &head.uncompressed, &head.compressed,
	}
	for _, f := range fields {
		v, next, err := unpackUInt(rest)
		if err != nil {
			return head, errSF20
		}
		*f = v
		rest = next
	}
	head.bodyAt = len(diff) - len(rest)
	return head, nil
}

type rle0 struct {
	code  []byte
	len0  uint64
	lenv  uint64
	need0 bool
}

func (r *rle0) reset(code []byte) {
	r.code = code
	r.len0 = 0
	r.lenv = 0
	r.need0 = true
}

func (r *rle0) add(dst []byte) error {
	for len(dst) > 0 {
		if r.len0 > 0 {
			n := r.len0
			if n > uint64(len(dst)) {
				n = uint64(len(dst))
			}
			dst = dst[n:]
			r.len0 -= n
			continue
		}
		if r.lenv > 0 {
			n := r.lenv
			if n > uint64(len(dst)) {
				n = uint64(len(dst))
			}
			if n > uint64(len(r.code)) {
				return errSF20
			}
			for i := uint64(0); i < n; i++ {
				dst[i] += r.code[i]
			}
			dst = dst[n:]
			r.code = r.code[n:]
			r.lenv -= n
			continue
		}
		if len(r.code) == 0 {
			return errSF20
		}
		if r.need0 {
			v, rest, err := unpackUInt(r.code)
			if err != nil {
				return errSF20
			}
			r.code = rest
			r.need0 = false
			r.len0 = v
			continue
		}
		v, rest, err := unpackUInt(r.code)
		if err != nil || v > uint64(len(rest)) {
			return errSF20
		}
		r.code = rest
		r.need0 = true
		r.lenv = v
	}
	return nil
}

func unpackTag(b []byte, tagBits uint) (uint64, []byte, error) {
	if len(b) == 0 || tagBits > 7 {
		return 0, b, fmt.Errorf("x5n: truncated integer")
	}
	code := b[0]
	b = b[1:]
	low := 7 - tagBits
	value := uint64(code) & ((1 << low) - 1)
	if code&(1<<low) == 0 {
		return value, b, nil
	}
	for {
		if len(b) == 0 {
			return 0, b, fmt.Errorf("x5n: truncated integer")
		}
		if value>>(64-7) != 0 {
			return 0, b, fmt.Errorf("x5n: integer overflow")
		}
		code = b[0]
		b = b[1:]
		value = (value << 7) | uint64(code&0x7f)
		if code&0x80 == 0 {
			return value, b, nil
		}
	}
}
