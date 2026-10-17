package gifrec

import (
	"errors"
	"fmt"
)

var (
	errRecompress = errors.New("gifrec: recompress")
	errPenalty    = errors.New("gifrec: penalty")
)

// Recompress implements precomp recompress_gif with a NULL GifCodeStruct.
// src is the PGF blob. blockSize is 254 or 255. diff is the raw GIFDiff bytes
// stored in the PCF (not including the vlint length).
func Recompress(src []byte, blockSize int, diff []byte) ([]byte, error) {
	in := &cursor{b: src}
	var out cursor
	mayWrite := false
	dec, err := dGifOpenPCF(in)
	if err != nil {
		return nil, err
	}
	enc := eGifOpen(&out, &mayWrite, byte(blockSize))
	screen, err := allocScreen(dec)
	if err != nil {
		return nil, err
	}
	if err := enc.putScreenDesc(dec.w, dec.h, dec.colorRes, dec.bg, dec.aspect, dec.sMap); err != nil {
		return nil, err
	}
	gd := newDiff(diff)
	initPos := 0
	lastPos := -1
	for {
		rec, err := dec.getRecordType()
		if err != nil {
			return nil, err
		}
		switch rec {
		case recImage:
			if err := dec.getImageDesc(); err != nil {
				return nil, err
			}
			srcPos := in.tell()
			if lastPos != srcPos {
				if lastPos == -1 {
					if err := in.seek(initPos); err != nil {
						return nil, err
					}
					if err := fastCopy(in, &out, srcPos-initPos); err != nil {
						return nil, err
					}
					if err := in.seek(srcPos); err != nil {
						return nil, err
					}
					dstPos := out.tell()
					if err := out.seek(0); err != nil {
						return nil, err
					}
					out.write([]byte{'G', 'I'})
					if err := out.seek(dstPos); err != nil {
						return nil, err
					}
				} else {
					if err := in.seek(lastPos); err != nil {
						return nil, err
					}
					if err := fastCopy(in, &out, srcPos-lastPos); err != nil {
						return nil, err
					}
					if err := in.seek(srcPos); err != nil {
						return nil, err
					}
				}
			}
			row := int(dec.image.top)
			col := int(dec.image.left)
			width := int(dec.image.width)
			height := int(dec.image.height)
			if err := readPixels(in, screen, row, col, width, height); err != nil {
				return nil, err
			}
			if err := enc.putImageDesc(dec.image.top, dec.image.left, dec.image.width, dec.image.height, dec.image.interlace, dec.image.colorMap, gd); err != nil {
				return nil, err
			}
			mayWrite = true
			if dec.image.interlace != 0 {
				for pass := 0; pass < 4; pass++ {
					for j := row + laceOff[pass]; j < row+height; j += laceJump[pass] {
						if err := enc.putLine(screen[j][col:col+width], gd); err != nil {
							return nil, err
						}
					}
				}
			} else {
				for y := row; y < row+height; y++ {
					if err := enc.putLine(screen[y][col:col+width], gd); err != nil {
						return nil, err
					}
				}
			}
			mayWrite = false
			lastPos = in.tell()
		case recExt:
			if err := dec.skipExtension(); err != nil {
				return nil, err
			}
		case recTerm:
		default:
			return nil, fmt.Errorf("%w: record", errRecompress)
		}
		if rec == recTerm {
			break
		}
	}
	srcPos := in.tell()
	if lastPos != srcPos {
		if lastPos < 0 {
			return nil, fmt.Errorf("%w: no image", errRecompress)
		}
		if err := in.seek(lastPos); err != nil {
			return nil, err
		}
		if err := fastCopy(in, &out, srcPos-lastPos); err != nil {
			return nil, err
		}
	}
	mayWrite = false
	enc.close()
	return out.b, nil
}

func allocScreen(f *gifFile) ([][]byte, error) {
	if f.h <= 0 || f.w <= 0 {
		return nil, fmt.Errorf("%w: screen size", errRecompress)
	}
	// GIF screen words are 16-bit; refuse anything that cannot be one.
	if f.w > 0xffff || f.h > 0xffff {
		return nil, fmt.Errorf("%w: screen size", errRecompress)
	}
	s := make([][]byte, f.h)
	row0 := make([]byte, f.w)
	bg := byte(f.bg)
	for i := range row0 {
		row0[i] = bg
	}
	s[0] = row0
	for y := 1; y < int(f.h); y++ {
		s[y] = append([]byte(nil), row0...)
	}
	return s, nil
}

func readPixels(in *cursor, screen [][]byte, row, col, width, height int) error {
	if row < 0 || col < 0 || width < 0 || height < 0 || row+height > len(screen) {
		return fmt.Errorf("%w: image bounds", errRecompress)
	}
	for y := row; y < row+height; y++ {
		if col+width > len(screen[y]) {
			return fmt.Errorf("%w: image bounds", errRecompress)
		}
		pix, err := in.read(width)
		if err != nil {
			return err
		}
		copy(screen[y][col:col+width], pix)
	}
	return nil
}

// ApplyPenalty patches output the way D_GIF does after recompress.
// Each record is 4-byte big-endian position plus one byte. Positions are
// relative to the start of the recompressed GIF.
func ApplyPenalty(out []byte, penalty []byte) ([]byte, error) {
	if len(penalty)%5 != 0 {
		return nil, fmt.Errorf("%w: truncated", errPenalty)
	}
	dst := append([]byte(nil), out...)
	for pbc := 0; pbc < len(penalty); pbc += 5 {
		var pb int32
		pb = int32(penalty[pbc]) << 24
		pb += int32(penalty[pbc+1]) << 16
		pb += int32(penalty[pbc+2]) << 8
		pb += int32(penalty[pbc+3])
		if pb < 0 || int(pb) >= len(dst) {
			return nil, fmt.Errorf("%w: position %d", errPenalty, pb)
		}
		dst[pb] = penalty[pbc+4]
	}
	return dst, nil
}
