// Package x5 applies the uncompressed HDIFF13 patches used by the installer's x5 step.
package x5

import (
	"context"
	"errors"
	"fmt"
)

// Limits match the old guest wrapper: each input is at most 1 GiB, and the
// three buffers together stay under the guest's 2 GiB memory cap minus stack.
const (
	maxSize    = 1 << 30
	memCeiling = (2 << 30) - (64 << 20)
)

const (
	signTagBits = 1
	rleTagBits  = 2

	rleType0    = 0
	rleType255  = 1
	rleTypeByte = 2
	rleTypeRaw  = 3
)

var (
	errHeader  = errors.New("x5: invalid or unsupported patch header")
	errCorrupt = errors.New("x5: corrupt patch")
)

// Apply reconstructs the target from old and an uncompressed HDIFF13 patch.
// Behavior follows third_party/hdiffpatch v2.5.3 patch_decompress with a null
// decompressor, including the bounds checks compiled into patch.c.
func Apply(ctx context.Context, old, diff []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(old) > maxSize || len(diff) > maxSize {
		return nil, fmt.Errorf("x5: input exceeds Guest memory limit")
	}
	head, err := readHDiff13(diff)
	if err != nil {
		return nil, err
	}
	if head.oldSize != uint64(len(old)) || head.compressType != "" {
		return nil, errHeader
	}
	if head.newSize == 0 || head.newSize > maxSize || uint64(len(old))+uint64(len(diff))+head.newSize > memCeiling {
		return nil, errHeader
	}
	if head.compCover != 0 || head.compRleCtrl != 0 || head.compRleCode != 0 || head.compNewDiff != 0 {
		return nil, errCorrupt
	}
	cover, rleCtrl, rleCode, lit, err := sliceSections(diff, head)
	if err != nil {
		return nil, err
	}
	return applyCovers(ctx, old, head, cover, rleCtrl, rleCode, lit)
}

type hdiff13 struct {
	newSize      uint64
	oldSize      uint64
	coverCount   uint64
	coverSize    uint64
	compCover    uint64
	rleCtrlSize  uint64
	compRleCtrl  uint64
	rleCodeSize  uint64
	compRleCode  uint64
	newDiffSize  uint64
	compNewDiff  uint64
	headEnd      int
	compressType string
}

func readHDiff13(diff []byte) (hdiff13, error) {
	var head hdiff13
	if len(diff) < 8 || string(diff[:8]) != "HDIFF13&" {
		return head, errHeader
	}
	rest := diff[8:]
	comp, rest, err := readCString(rest, 256)
	if err != nil {
		return head, errHeader
	}
	head.compressType = comp
	fields := []*uint64{
		&head.newSize, &head.oldSize, &head.coverCount,
		&head.coverSize, &head.compCover,
		&head.rleCtrlSize, &head.compRleCtrl,
		&head.rleCodeSize, &head.compRleCode,
		&head.newDiffSize, &head.compNewDiff,
	}
	for _, f := range fields {
		v, next, err := unpackTag(rest, 0)
		if err != nil {
			return head, errHeader
		}
		*f = v
		rest = next
	}
	head.headEnd = len(diff) - len(rest)
	return head, nil
}

func readCString(b []byte, maxLen int) (string, []byte, error) {
	n := maxLen + 1
	if n > len(b) {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if b[i] == 0 {
			return string(b[:i]), b[i+1:], nil
		}
	}
	return "", b, errHeader
}

func sliceSections(diff []byte, head hdiff13) (cover, rleCtrl, rleCode, lit []byte, err error) {
	rest := diff[head.headEnd:]
	take := func(n uint64) ([]byte, error) {
		if n > uint64(len(rest)) {
			return nil, errCorrupt
		}
		out := rest[:n]
		rest = rest[n:]
		return out, nil
	}
	if cover, err = take(head.coverSize); err != nil {
		return
	}
	if rleCtrl, err = take(head.rleCtrlSize); err != nil {
		return
	}
	if rleCode, err = take(head.rleCodeSize); err != nil {
		return
	}
	if lit, err = take(head.newDiffSize); err != nil {
		return
	}
	if len(rest) != 0 {
		err = errCorrupt
	}
	return
}

func applyCovers(ctx context.Context, old []byte, head hdiff13, cover, rleCtrl, rleCode, lit []byte) ([]byte, error) {
	out := make([]byte, head.newSize)
	rle := rleDec{ctrl: rleCtrl, code: rleCode}
	var oldBack, newBack uint64
	wrote := uint64(0)
	litAt := 0
	for i := uint64(0); i < head.coverCount; i++ {
		if i&0xfff == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		if len(cover) == 0 {
			return nil, errCorrupt
		}
		sign := cover[0] >> (8 - signTagBits)
		incOld, rest, err := unpackTag(cover, signTagBits)
		if err != nil {
			return nil, errCorrupt
		}
		copyLen, rest, err := unpackTag(rest, 0)
		if err != nil {
			return nil, errCorrupt
		}
		coverLen, rest, err := unpackTag(rest, 0)
		if err != nil {
			return nil, errCorrupt
		}
		cover = rest
		var oldPos uint64
		if sign == 0 {
			if addOverflow(oldBack, incOld) {
				return nil, errCorrupt
			}
			oldPos = oldBack + incOld
		} else if incOld > oldBack {
			return nil, errCorrupt
		} else {
			oldPos = oldBack - incOld
		}
		if addOverflow(newBack, copyLen) {
			return nil, errCorrupt
		}
		newPos := newBack + copyLen
		if newPos > head.newSize || coverLen > head.newSize-newPos || newPos < wrote {
			return nil, errCorrupt
		}
		if oldPos > uint64(len(old)) || coverLen > uint64(len(old))-oldPos {
			return nil, errCorrupt
		}
		if newPos > wrote {
			gap := int(newPos - wrote)
			if litAt > len(lit)-gap {
				return nil, errCorrupt
			}
			copy(out[int(wrote):int(newPos)], lit[litAt:litAt+gap])
			if err := rle.consume(gap, nil); err != nil {
				return nil, errCorrupt
			}
			litAt += gap
			wrote = newPos
		}
		dst := out[int(wrote) : int(wrote)+int(coverLen)]
		copy(dst, old[int(oldPos):int(oldPos)+int(coverLen)])
		if err := rle.consume(int(coverLen), dst); err != nil {
			return nil, errCorrupt
		}
		if addOverflow(oldPos, coverLen) || addOverflow(newPos, coverLen) {
			return nil, errCorrupt
		}
		oldBack = oldPos + coverLen
		newBack = newPos + coverLen
		wrote = newBack
	}
	if len(cover) != 0 {
		return nil, errCorrupt
	}
	if wrote < head.newSize {
		gap := int(head.newSize - wrote)
		if litAt > len(lit)-gap {
			return nil, errCorrupt
		}
		copy(out[wrote:], lit[litAt:litAt+gap])
		if err := rle.consume(gap, nil); err != nil {
			return nil, errCorrupt
		}
		litAt += gap
		wrote = head.newSize
	}
	if litAt != len(lit) || !rle.done() || wrote != head.newSize {
		return nil, errCorrupt
	}
	return out, nil
}

type rleDec struct {
	ctrl    []byte
	code    []byte
	setLen  uint64
	setVal  byte
	copyLen uint64
}

func (d *rleDec) done() bool {
	return d.setLen == 0 && d.copyLen == 0 && len(d.ctrl) == 0 && len(d.code) == 0
}

func (d *rleDec) consume(n int, dst []byte) error {
	if n < 0 {
		return errCorrupt
	}
	for n > 0 {
		if d.setLen > 0 {
			step := d.setLen
			if step > uint64(n) {
				step = uint64(n)
			}
			if dst != nil {
				if d.setVal != 0 {
					for i := uint64(0); i < step; i++ {
						dst[i] += d.setVal
					}
				}
				dst = dst[step:]
			}
			n -= int(step)
			d.setLen -= step
			continue
		}
		if d.copyLen > 0 {
			step := d.copyLen
			if step > uint64(n) {
				step = uint64(n)
			}
			if step > uint64(len(d.code)) {
				return errCorrupt
			}
			if dst != nil {
				for i := uint64(0); i < step; i++ {
					dst[i] += d.code[i]
				}
				dst = dst[step:]
			}
			d.code = d.code[step:]
			d.copyLen -= step
			n -= int(step)
			continue
		}
		if len(d.ctrl) == 0 {
			return errCorrupt
		}
		typ := d.ctrl[0] >> (8 - rleTagBits)
		length, rest, err := unpackTag(d.ctrl, rleTagBits)
		if err != nil || length == ^uint64(0) {
			return errCorrupt
		}
		d.ctrl = rest
		length++
		switch typ {
		case rleType0:
			d.setLen = length
			d.setVal = 0
		case rleType255:
			d.setLen = length
			d.setVal = 255
		case rleTypeByte:
			if len(d.code) < 1 {
				return errCorrupt
			}
			d.setVal = d.code[0]
			d.code = d.code[1:]
			d.setLen = length
		case rleTypeRaw:
			d.copyLen = length
		default:
			return errCorrupt
		}
	}
	return nil
}

func unpackTag(b []byte, tagBits uint) (uint64, []byte, error) {
	if len(b) == 0 || tagBits > 7 {
		return 0, b, errCorrupt
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
			return 0, b, errCorrupt
		}
		if value>>(64-7) != 0 {
			return 0, b, errCorrupt
		}
		code = b[0]
		b = b[1:]
		value = (value << 7) | uint64(code&0x7f)
		if code&0x80 == 0 {
			return value, b, nil
		}
	}
}

func addOverflow(a, b uint64) bool {
	return a > ^uint64(0)-b
}
