package mpzz

import (
	"encoding/binary"
	"fmt"
	"io"
)

type cachedHeaders struct {
	pages []byte
	setup vorbisSetup
}

type oggreDecoder struct {
	r        io.Reader
	h        header
	cmd      *rangeDecoder
	control  [18]uint16
	previous uint32
	models   []integerModel
	probs    []uint16
	cache    bookCache
	headers  []cachedHeaders
	streams  [][]byte
}

func newOGGREDecoder(r io.Reader, h header) (*oggreDecoder, error) {
	frame, err := readFrame(r)
	if err != nil {
		return nil, err
	}
	d := &oggreDecoder{r: r, h: h, cmd: newRange(frame), models: make([]integerModel, 55)}
	for i := range d.models {
		d.models[i].reset()
	}
	for i := range d.control {
		d.control[i] = 0x8000
	}
	d.cache.capacity = 1 << (4 + (h.flags & 7))
	d.probs = make([]uint16, 0x9225c+d.cache.capacity*0x45400)
	for i := range d.probs {
		d.probs[i] = 0x8000
	}
	return d, d.cmd.err
}

func (d *oggreDecoder) next() ([]byte, error) {
	d.previous = d.cmd.bit(&d.control[d.previous])
	if d.cmd.err != nil {
		return nil, d.cmd.err
	}
	if d.previous == 0 {
		b, err := readFrame(d.r)
		if err != nil {
			return nil, err
		}
		if len(b) == 0 {
			return nil, io.EOF
		}
		return b, nil
	}
	if d.cmd.bit(&d.control[2]) != 0 {
		return nil, fmt.Errorf("mpzz: repeated whole stream not yet reconstructed")
	}
	store := d.cmd.bit(&d.control[6]) != 0
	var ranges [3]*rangeDecoder
	for i := range ranges {
		b, err := readFrame(d.r)
		if err != nil {
			return nil, err
		}
		ranges[i] = newRange(b)
	}
	if d.h.flags&8 != 0 {
		// Reset list at 0x10039006: probabilities persist in solid mode.
		for _, i := range []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 17, 18, 19, 20, 21, 22, 23, 27, 28, 29, 30} {
			m := &d.models[i]
			m.value, m.delta, m.nonzero, m.sign, m.length = 0, 0, 0, 0, 0
		}
	} else {
		for i := 0; i < 52; i++ {
			d.models[i].reset()
		}
		for i := range d.probs {
			d.probs[i] = 0x8000
		}
		d.cache.books = nil
	}
	a := &audioDecoder{body: ranges[2], header: ranges[1], aux: ranges[0], models: d.models, probs: d.probs}
	for i := range d.cache.books {
		d.cache.books[i].used = false
	}
	out, err := d.decodeStream(a)
	if err != nil {
		return nil, err
	}
	if store {
		d.streams = append(d.streams, append([]byte(nil), out...))
	}
	for i := range d.cache.books {
		b := &d.cache.books[i]
		if !b.used && b.uses != 0 {
			b.uses--
		}
	}
	return out, nil
}

func rewriteSerial(pages []byte, serial uint32) ([]byte, error) {
	out := append([]byte(nil), pages...)
	for pos := 0; pos < len(out); {
		if len(out)-pos < 27 {
			return nil, fmt.Errorf("mpzz: truncated cached page")
		}
		n := int(out[pos+26])
		size := 27 + n
		if size > len(out)-pos {
			return nil, fmt.Errorf("mpzz: truncated cached lacing")
		}
		for _, v := range out[pos+27 : pos+size] {
			size += int(v)
		}
		if size > len(out)-pos {
			return nil, fmt.Errorf("mpzz: truncated cached body")
		}
		p := out[pos : pos+size]
		binary.LittleEndian.PutUint32(p[14:], serial)
		clear(p[22:26])
		binary.LittleEndian.PutUint32(p[22:], oggCRC(p))
		pos += size
	}
	return out, nil
}

func (d *oggreDecoder) decodeStream(a *audioDecoder) ([]byte, error) {
	h, err := decodePageHeader(a.header, d.models, &d.probs[0x208], 2, 0)
	if err != nil {
		return nil, err
	}
	if len(h.packets) != 1 || h.packets[0] != 30 {
		return nil, fmt.Errorf("mpzz: invalid identification page")
	}
	ident := setupDecoder{r: a.body, models: d.models, probs: d.probs}
	ident.w.write(1, 8)
	for _, b := range []byte("vorbis") {
		ident.w.write(uint32(b), 8)
	}
	ident.w.write(0, 32)
	a.channels = int(ident.integer(9, 2, 1, 1, false, 1, 8))
	for i := 10; i < 14; i++ {
		ident.integer(i, 5, 1, 4, false, 1, 32)
	}
	block := ident.integer(14, 3, 1, 4, false, 1, 8)
	framing := ident.integer(15, 3, 1, 4, false, 1, 8)
	small, large := block&15, block>>4
	if a.channels < 1 || a.channels > 8 || small < 6 || large > 13 || small > large || framing&1 == 0 {
		return nil, fmt.Errorf("mpzz: invalid identification fields")
	}
	a.blocks = [2]int{1 << small, 1 << large}
	out, err := h.marshal(ident.w.data)
	if err != nil {
		return nil, err
	}
	serial := h.serial
	var config vorbisSetup
	if d.cmd.bit(&d.control[10]) != 0 {
		back := int(d.models[53].integer(d.cmd, 5, 0, 2, true))
		index := len(d.headers) - 1 - back
		if index < 0 {
			return nil, fmt.Errorf("mpzz: invalid cached header %d", back)
		}
		cached := d.headers[index]
		pages, err := rewriteSerial(cached.pages, serial)
		if err != nil {
			return nil, err
		}
		out = append(out, pages...)
		config = cached.setup
		config.books = append([]codebook(nil), config.books...)
		for i := range config.books {
			b := &config.books[i]
			if b.lookup != 0 {
				b.context = d.cache.selectBook(b, d.probs)
			}
		}
		d.models[37].value = d.models[29].value
		d.models[37].delta = 0
	} else {
		h, err = decodePageHeader(a.header, d.models, &d.probs[0x208], 0, 0)
		if err != nil {
			return nil, err
		}
		store := d.cmd.bit(&d.control[14]) != 0
		if len(h.packets) != 2 {
			return nil, fmt.Errorf("mpzz: header page packet count %d", len(h.packets))
		}
		comment := make([]byte, h.packets[0])
		prev := 0
		for i := range comment {
			comment[i] = byte(a.tree(a.body, 0x1218+(prev&0xf0)*16, 8))
			prev = int(comment[i])
		}
		s := setupDecoder{r: a.body, models: d.models, probs: d.probs, cache: &d.cache}
		books, err := s.codebooks()
		if err != nil {
			return nil, err
		}
		config, err = s.config(a.channels, books)
		if err != nil {
			return nil, err
		}
		a.w = s.w
		if err := a.finishPacket(h.packets[1]); err != nil {
			return nil, err
		}
		body := append(comment, a.w.data...)
		pages, err := h.marshal(body)
		if err != nil {
			return nil, err
		}
		out = append(out, pages...)
		if store {
			d.headers = append(d.headers, cachedHeaders{pages: pages, setup: config})
		}
	}
	a.setup = &config
	for page := 0; page < 1<<20; page++ {
		h, err = decodePageHeader(a.header, d.models, &d.probs[0x208], 0, a.granule)
		if err != nil {
			return nil, fmt.Errorf("mpzz: audio page %d: %w", page, err)
		}
		if h.flags&1 != 0 {
			return nil, fmt.Errorf("mpzz: continued audio packet not yet reconstructed")
		}
		var body []byte
		for i, size := range h.packets {
			mode, err := a.mode(h.flags)
			if err != nil {
				return nil, err
			}
			n := d.models[7].integer(a.aux, 3, 0, 1, true)
			if n != 0 {
				return nil, fmt.Errorf("mpzz: audio page %d packet %d has %d tail bits", page, i, n)
			}
			mapping := config.mappings[mode.mapping]
			silent, err := a.floors(mapping)
			if err != nil {
				return nil, fmt.Errorf("mpzz: audio page %d packet %d floor: %w", page, i, err)
			}
			if err := a.residues(mapping, silent); err != nil {
				return nil, fmt.Errorf("mpzz: audio page %d packet %d residue: %w", page, i, err)
			}
			if err := a.finishPacket(size); err != nil {
				return nil, fmt.Errorf("mpzz: audio page %d packet %d: %w", page, i, err)
			}
			body = append(body, a.w.data...)
		}
		data, err := h.marshal(body)
		if err != nil {
			return nil, err
		}
		out = append(out, data...)
		if h.flags&4 != 0 {
			return out, nil
		}
	}
	return nil, fmt.Errorf("mpzz: excessive audio pages")
}
