package gifrec

import "fmt"

func eGifOpen(out *cursor, mayWrite *bool, blockSize byte) *gifFile {
	return &gifFile{
		blockSize: blockSize,
		priv: &gifPriv{
			state:    fileStateWrite,
			out:      out,
			mayWrite: mayWrite,
			codes:    newCodeTable(),
		},
	}
}

func (f *gifFile) putWord(word int32) error {
	var c [2]byte
	c[0] = byte(word)
	c[1] = byte(uint32(word) >> 8)
	if f.priv.writeBytes(c[:]) != 2 {
		return fmt.Errorf("%w: put word", errRecompress)
	}
	return nil
}

func (f *gifFile) putScreenDesc(width, height, colorRes, bg, aspect int32, cm *colorMap) error {
	p := f.priv
	if p.state&fileStateScreen != 0 {
		return fmt.Errorf("%w: screen set", errRecompress)
	}
	if p.state&fileStateWrite == 0 {
		return fmt.Errorf("%w: not writable", errRecompress)
	}
	// Default GifVersionPrefix is GIF87a. Restore discards this write and
	// copies the PGF header instead.
	if p.writeBytes([]byte("GIF87a")) != 6 {
		return fmt.Errorf("%w: stamp", errRecompress)
	}
	f.w = width
	f.h = height
	f.colorRes = colorRes
	f.bg = bg
	f.aspect = aspect
	if cm != nil {
		f.sMap = makeMap(cm.count, cm.color)
		if f.sMap == nil {
			return fmt.Errorf("%w: screen map", errRecompress)
		}
	} else {
		f.sMap = nil
	}
	if err := f.putWord(width); err != nil {
		return err
	}
	if err := f.putWord(height); err != nil {
		return err
	}
	var buf [3]byte
	buf[0] = byte((colorRes-1)<<4) | 0x07
	if cm != nil {
		buf[0] = 0x80 | byte((colorRes-1)<<4) | byte(cm.bpp-1)
	}
	buf[1] = byte(bg)
	buf[2] = byte(aspect)
	if p.writeBytes(buf[:]) != 3 {
		return fmt.Errorf("%w: screen desc", errRecompress)
	}
	if cm != nil {
		for i := int32(0); i < cm.count; i++ {
			rgb := [3]byte{cm.color[i].r, cm.color[i].g, cm.color[i].b}
			if p.writeBytes(rgb[:]) != 3 {
				return fmt.Errorf("%w: screen colors", errRecompress)
			}
		}
	}
	p.state |= fileStateScreen
	return nil
}

func (f *gifFile) putImageDesc(left, top, width, height, interlace int32, cm *colorMap, gd *gifDiff) error {
	p := f.priv
	if p.state&fileStateImage != 0 && p.pixelCount > 0xffff0000 {
		return fmt.Errorf("%w: image active", errRecompress)
	}
	if p.state&fileStateWrite == 0 {
		return fmt.Errorf("%w: not writable", errRecompress)
	}
	f.image.left = left
	f.image.top = top
	f.image.width = width
	f.image.height = height
	f.image.interlace = interlace
	if cm != nil {
		f.image.colorMap = makeMap(cm.count, cm.color)
		if f.image.colorMap == nil {
			return fmt.Errorf("%w: image map", errRecompress)
		}
	} else {
		f.image.colorMap = nil
	}
	if p.writeBytes([]byte{','}) != 1 {
		return fmt.Errorf("%w: image sep", errRecompress)
	}
	if err := f.putWord(left); err != nil {
		return err
	}
	if err := f.putWord(top); err != nil {
		return err
	}
	if err := f.putWord(width); err != nil {
		return err
	}
	if err := f.putWord(height); err != nil {
		return err
	}
	var packed byte
	if cm != nil {
		packed |= 0x80 | byte(cm.bpp-1)
	}
	if interlace != 0 {
		packed |= 0x40
	}
	if p.writeBytes([]byte{packed}) != 1 {
		return fmt.Errorf("%w: image packed", errRecompress)
	}
	if cm != nil {
		for i := int32(0); i < cm.count; i++ {
			rgb := [3]byte{cm.color[i].r, cm.color[i].g, cm.color[i].b}
			if p.writeBytes(rgb[:]) != 3 {
				return fmt.Errorf("%w: image colors", errRecompress)
			}
		}
	}
	if f.sMap == nil && f.image.colorMap == nil {
		return fmt.Errorf("%w: no color map", errRecompress)
	}
	p.state |= fileStateImage
	p.pixelCount = uint64(int64(width) * int64(height))
	return f.setupCompress(gd)
}

func (f *gifFile) putLine(line []byte, gd *gifDiff) error {
	p := f.priv
	if p.state&fileStateWrite == 0 {
		return fmt.Errorf("%w: not writable", errRecompress)
	}
	lineLen := len(line)
	if p.pixelCount < uint64(lineLen) {
		return fmt.Errorf("%w: too many pixels", errRecompress)
	}
	p.pixelCount -= uint64(lineLen)
	mask := codeMask[p.bpp]
	for i := range line {
		line[i] &= mask
	}
	return f.compressLine(line, gd)
}

func (f *gifFile) close() {
	// Trailer is copied from the PGF; this write is suppressed on restore.
	f.priv.writeBytes([]byte{';'})
}

func (f *gifFile) setupCompress(gd *gifDiff) error {
	var bpp int32
	if f.image.colorMap != nil {
		bpp = f.image.colorMap.bpp
	} else if f.sMap != nil {
		bpp = f.sMap.bpp
	} else {
		return fmt.Errorf("%w: no color map", errRecompress)
	}
	if bpp < 2 {
		bpp = 2
	}
	p := f.priv
	p.writeBytes([]byte{byte(bpp)})
	p.buf[0] = 0
	p.bpp = bpp
	p.clearCode = int32(1) << bpp
	p.eofCode = p.clearCode + 1
	p.runningCode = p.eofCode + 1
	p.runningBits = bpp + 1
	p.maxCode1 = int32(1) << p.runningBits
	p.crntCode = firstCode
	p.shiftState = 0
	p.shiftDWord = 0
	clearCodeTable(p.codes)
	// NULL GifCodeStruct: count the initial clear instead of reading it.
	gd.codeCount++
	return f.compressOutput(p.clearCode)
}

func (f *gifFile) compressLine(line []byte, gd *gifDiff) error {
	p := f.priv
	var i int
	var crnt int32
	if p.crntCode == firstCode {
		crnt = int32(line[0])
		i = 1
	} else {
		crnt = p.crntCode
	}
	for i < len(line) {
		// Type 0: original stream had a clear the natural encoder would not emit yet.
		if gd.peekType() == 0 && gd.isPos() {
			gd.next()
			gd.codeCount++
			if err := f.compressOutput(p.clearCode); err != nil {
				return err
			}
			p.runningCode = p.eofCode + 1
			p.runningBits = p.bpp + 1
			p.maxCode1 = int32(1) << p.runningBits
			clearCodeTable(p.codes)
		}
		pixel := int32(line[i])
		i++
		newKey := (uint32(crnt) << 8) + uint32(pixel)
		// Type 2: emit CrntCode early even though the extended string is known.
		crntIsNext := gd.peekType() == 2 && gd.isPos() && gd.dataWord() == crnt
		newCode := p.codes[newKey]
		if newCode > -1 && !crntIsNext {
			crnt = newCode
			continue
		}
		if newCode > -1 && crntIsNext {
			gd.next()
		}
		gd.codeCount++
		if err := f.compressOutput(crnt); err != nil {
			return err
		}
		crnt = pixel
		if p.runningCode >= lzMaxCode {
			// Type 1: table is full but the original stream did not clear.
			if gd.peekType() == 1 && gd.isPos() {
				gd.next()
				p.codes[newKey] = p.runningCode
			} else {
				gd.codeCount++
				if err := f.compressOutput(p.clearCode); err != nil {
					return err
				}
				p.runningCode = p.eofCode + 1
				p.runningBits = p.bpp + 1
				p.maxCode1 = int32(1) << p.runningBits
				clearCodeTable(p.codes)
			}
		} else {
			p.codes[newKey] = p.runningCode
			p.runningCode++
		}
	}
	p.crntCode = crnt
	if p.pixelCount == 0 {
		gd.codeCount++
		if err := f.compressOutput(crnt); err != nil {
			return err
		}
		if err := f.compressOutput(p.eofCode); err != nil {
			return err
		}
		if err := f.compressOutput(flushOutput); err != nil {
			return err
		}
	}
	return nil
}

func (f *gifFile) compressOutput(code int32) error {
	p := f.priv
	if code == flushOutput {
		for p.shiftState > 0 {
			if err := f.buffered(int32(p.shiftDWord & 0xff)); err != nil {
				return err
			}
			p.shiftDWord >>= 8
			p.shiftState -= 8
		}
		p.shiftState = 0
		if err := f.buffered(flushOutput); err != nil {
			return err
		}
	} else {
		// C casts Code to long (64-bit here) before the shift into unsigned long.
		p.shiftDWord |= uint64(int64(code)) << uint(p.shiftState)
		p.shiftState += p.runningBits
		for p.shiftState >= 8 {
			if err := f.buffered(int32(p.shiftDWord & 0xff)); err != nil {
				return err
			}
			p.shiftDWord >>= 8
			p.shiftState -= 8
		}
	}
	if p.runningCode >= p.maxCode1 && code <= 4095 {
		p.runningBits++
		p.maxCode1 = int32(1) << p.runningBits
	}
	return nil
}

func (f *gifFile) buffered(c int32) error {
	p := f.priv
	if c == flushOutput {
		if p.buf[0] != 0 {
			n := int(p.buf[0]) + 1
			if p.writeBytes(p.buf[:n]) != n {
				return fmt.Errorf("%w: flush block", errRecompress)
			}
		}
		p.buf[0] = 0
		if p.writeBytes([]byte{0}) != 1 {
			return fmt.Errorf("%w: block end", errRecompress)
		}
		return nil
	}
	if p.buf[0] == f.blockSize {
		n := int(p.buf[0]) + 1
		if p.writeBytes(p.buf[:n]) != n {
			return fmt.Errorf("%w: full block", errRecompress)
		}
		p.buf[0] = 0
	}
	p.buf[0]++
	p.buf[p.buf[0]] = byte(c)
	return nil
}
