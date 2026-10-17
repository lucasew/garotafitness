package tor

import "context"

// lzDec is one of the four Tornado LZ77 decoders.
type lzDec interface {
	next() (literal bool, err error)
	char() byte
	length(minlen uint32) uint32
	distance() uint32
	failed() error
}

// decompress decodes one Tornado stream.
// limit < 0 keeps every flushed chunk. A non-negative limit drops a chunk
// that would pass the cap, without failing the decode — the wasm guest's
// write callback does the same and Tornado treats a 0 write as success.
func decompress(ctx context.Context, in []byte, limit int) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(in) < 6 {
		return nil, errFormat
	}
	method := in[0]
	minlen := uint32(in[1])
	dict := uint32(in[2]) | uint32(in[3])<<8 | uint32(in[4])<<16 | uint32(in[5])<<24
	if dict > 1<<30 {
		return nil, errFormat
	}
	rest := in[6:]
	var dec lzDec
	switch method {
	case byteCoder:
		dec = newByteDec(rest)
	case bitCoder:
		dec = newBitDec(rest)
	case hufCoder:
		dec = newLZ77(newHuffDec(rest, codes))
	case ariCoder:
		dec = newLZ77(newArithDec(rest, codes))
	default:
		return nil, errFormat
	}
	outSize := int(dict)
	if outSize < largeBuf {
		outSize = largeBuf
	}
	w := &window{
		buf:    make([]byte, padTables+outSize+padTables),
		base:   padTables,
		size:   outSize,
		dict:   int(dict),
		limit:  limit,
		tables: newDataTables(),
	}
	w.out = w.base
	w.wstart = w.base
	wend := w.size
	if wend > hugeBuf {
		wend = hugeBuf
	}
	w.wend = w.base + wend
	if err := w.run(ctx, dec, minlen); err != nil {
		return nil, err
	}
	return w.written, nil
}

type window struct {
	buf      []byte
	base     int
	size     int
	dict     int
	out      int
	wstart   int
	wend     int
	filePos  uint64
	overflow bool
	limit    int
	written  []byte
	tables   *dataTables
}

func (w *window) run(ctx context.Context, dec lzDec, minlen uint32) error {
	for step := 0; ; step++ {
		if step&4095 == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		if err := dec.failed(); err != nil {
			return err
		}
		lit, err := dec.next()
		if err != nil {
			return err
		}
		if lit {
			w.buf[w.out] = dec.char()
			w.out++
			if err := w.flush(false); err != nil {
				return err
			}
			continue
		}
		length := dec.length(minlen)
		if err := dec.failed(); err != nil {
			return err
		}
		dist := dec.distance()
		if err := dec.failed(); err != nil {
			return err
		}
		pos := w.out - w.base
		// Fast path: source is inside the window and the copy cannot reach write_end.
		if uint32(pos) >= dist && w.wend-w.out > int(length) {
			if length == 0 {
				return errFormat
			}
			p := w.out - int(dist)
			for length > 0 {
				w.buf[w.out] = w.buf[p]
				w.out++
				p++
				length--
			}
			continue
		}
		if length < impossible {
			if length == 0 || uint64(dist) > uint64(w.size) || int64(length) > 2*int64(w.dict) || (w.filePos+uint64(pos) < uint64(dist) && !w.overflow) {
				return errFormat
			}
			var p int
			if uint32(pos) >= dist {
				p = w.out - int(dist)
			} else {
				p = w.out - int(dist) + w.size
			}
			for length > 0 {
				w.buf[w.out] = w.buf[p]
				w.out++
				p++
				if p == w.base+w.size {
					p = w.base
				}
				if err := w.flush(false); err != nil {
					return err
				}
				length--
			}
			continue
		}
		if length == impossible && dist == impossible {
			return w.flush(true)
		}
		typ := length - impossible
		if typ == 0 || uint64(dist)*uint64(typ) > 2*uint64(w.dict) {
			return errFormat
		}
		if err := w.tables.add(int(typ), w.out, int(dist)); err != nil {
			return err
		}
		// compress_all_at_once is 0: a full table list flushes now.
		if w.tables.filled() {
			if err := w.flush(true); err != nil {
				return err
			}
		}
	}
}

func (w *window) flush(force bool) error {
	if !force && w.out < w.wend {
		return nil
	}
	if err := w.tables.undiff(w.buf, w.wstart, w.out); err != nil {
		return err
	}
	n := w.out - w.wstart
	if n < 0 {
		return errFormat
	}
	if n > 0 && (w.limit < 0 || len(w.written)+n <= w.limit) {
		w.written = append(w.written, w.buf[w.wstart:w.out]...)
	}
	if err := w.tables.diff(w.buf, w.wstart, w.out); err != nil {
		return err
	}
	w.wstart = w.out
	if w.out >= w.base+w.size {
		delta := w.out - w.base
		if w.filePos > 1<<63 {
			w.overflow = true
		}
		w.filePos += uint64(delta)
		w.wstart -= delta
		w.wend -= delta
		if err := w.tables.shift(w.buf, w.out, w.base); err != nil {
			return err
		}
		w.out -= delta
	}
	if w.wstart >= w.wend {
		room := w.base + w.size - w.wstart
		if room > hugeBuf {
			room = hugeBuf
		}
		if room < 0 {
			return errFormat
		}
		w.wend = w.wstart + room
	}
	return nil
}
