package gifrec

import "fmt"

func dGifOpenPCF(in *cursor) (*gifFile, error) {
	f := &gifFile{priv: &gifPriv{state: fileStateRead, in: in}}
	buf, err := f.priv.read(6)
	if err != nil {
		return nil, err
	}
	// GIF_STAMP_PCF compared through GIF_VERSION_POS (3).
	if buf[0] != 'P' || buf[1] != 'G' || buf[2] != 'F' {
		return nil, fmt.Errorf("%w: not pgf", errRecompress)
	}
	if err := f.getScreenDesc(); err != nil {
		return nil, err
	}
	return f, nil
}

func (f *gifFile) getWord() (int32, error) {
	c, err := f.priv.read(2)
	if err != nil {
		return 0, err
	}
	return int32(c[0]) | int32(c[1])<<8, nil
}

func (f *gifFile) getScreenDesc() error {
	if f.priv.state&fileStateRead == 0 {
		return fmt.Errorf("%w: not readable", errRecompress)
	}
	w, err := f.getWord()
	if err != nil {
		return err
	}
	h, err := f.getWord()
	if err != nil {
		return err
	}
	b, err := f.priv.read(3)
	if err != nil {
		return err
	}
	f.w = w
	f.h = h
	f.colorRes = (((int32(b[0]) & 0x70) + 1) >> 4) + 1
	bpp := int32(b[0]&0x07) + 1
	f.bg = int32(b[1])
	f.aspect = int32(b[2])
	if b[0]&0x80 != 0 {
		m := makeMap(int32(1)<<bpp, nil)
		if m == nil {
			return fmt.Errorf("%w: screen map", errRecompress)
		}
		for i := int32(0); i < m.count; i++ {
			rgb, err := f.priv.read(3)
			if err != nil {
				return err
			}
			m.color[i] = gifColor{rgb[0], rgb[1], rgb[2]}
		}
		f.sMap = m
	}
	return nil
}

func (f *gifFile) getRecordType() (int, error) {
	if f.priv.state&fileStateRead == 0 {
		return 0, fmt.Errorf("%w: not readable", errRecompress)
	}
	b, err := f.priv.read(1)
	if err != nil {
		return 0, err
	}
	switch b[0] {
	case ',':
		return recImage, nil
	case '!':
		return recExt, nil
	case ';':
		return recTerm, nil
	default:
		return 0, fmt.Errorf("%w: record %q", errRecompress, b[0])
	}
}

func (f *gifFile) getImageDesc() error {
	if f.priv.state&fileStateRead == 0 {
		return fmt.Errorf("%w: not readable", errRecompress)
	}
	left, err := f.getWord()
	if err != nil {
		return err
	}
	top, err := f.getWord()
	if err != nil {
		return err
	}
	w, err := f.getWord()
	if err != nil {
		return err
	}
	h, err := f.getWord()
	if err != nil {
		return err
	}
	b, err := f.priv.read(1)
	if err != nil {
		return err
	}
	f.image.left = left
	f.image.top = top
	f.image.width = w
	f.image.height = h
	f.image.interlace = int32(b[0] & 0x40)
	if b[0]&0x80 != 0 {
		bpp := int32(b[0]&0x07) + 1
		m := makeMap(int32(1)<<bpp, nil)
		if m == nil {
			return fmt.Errorf("%w: local map", errRecompress)
		}
		for i := int32(0); i < m.count; i++ {
			rgb, err := f.priv.read(3)
			if err != nil {
				return err
			}
			m.color[i] = gifColor{rgb[0], rgb[1], rgb[2]}
		}
		f.image.colorMap = m
	} else {
		f.image.colorMap = nil
	}
	// DGifSetupDecompress reads the LZW minimum code size and stops there
	// on the restore path; the pixel bytes that follow are raw.
	if _, err := f.priv.read(1); err != nil {
		return err
	}
	return nil
}

func (f *gifFile) skipExtension() error {
	if f.priv.state&fileStateRead == 0 {
		return fmt.Errorf("%w: not readable", errRecompress)
	}
	if _, err := f.priv.read(1); err != nil {
		return err
	}
	for {
		b, err := f.priv.read(1)
		if err != nil {
			return err
		}
		if b[0] == 0 {
			return nil
		}
		if _, err := f.priv.read(int(b[0])); err != nil {
			return err
		}
	}
}
