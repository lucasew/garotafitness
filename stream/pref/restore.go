package pref

import (
	"bytes"
	"compress/bzip2"
	"context"
	"fmt"
	"io"

	"github.com/lucasew/garotafitness/stream/pref/bz2"
	"github.com/lucasew/garotafitness/stream/pref/gifrec"
	"github.com/lucasew/garotafitness/stream/pref/jpgrec"
	"github.com/lucasew/garotafitness/stream/pref/mp3rec"
	"github.com/lucasew/garotafitness/stream/xt2png"
	"github.com/ulikunitz/xz"
)

// PCF 0.4.8 chunk tags. zlib_perfect is never set by this precomp's writer
// (the header byte always has bit 1 set); the reader still accepts the flag.
const (
	dPDF      = 0
	dZIP      = 1
	dGZIP     = 2
	dPNG      = 3
	dMultiPNG = 4
	dGIF      = 5
	dJPG      = 6
	dSWF      = 7
	dBase64   = 8
	dBzip2    = 9
	dMP3      = 10
	dBrute    = 254
	dRaw      = 255

	otfNone  = 0
	otfBzip2 = 1
	otfXZ    = 2

	// Same ceiling as the old prefdec.wasm MAXIMUM_MEMORY.
	maxBlob = 1 << 31
	div3    = 262143
)

const b64alpha = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

type cursor struct {
	b []byte
	i int
}

func (c *cursor) u8() (byte, error) {
	if c.i >= len(c.b) {
		return 0, fmt.Errorf("%w: truncated", errGuest)
	}
	v := c.b[c.i]
	c.i++
	return v, nil
}

func (c *cursor) take(n int) ([]byte, error) {
	if n < 0 || n > len(c.b)-c.i {
		return nil, fmt.Errorf("%w: truncated", errGuest)
	}
	p := c.b[c.i : c.i+n]
	c.i += n
	return p, nil
}

// vlint is precomp's integer, not LEB128. The encoder is
// while v >= 128 { emit (v&127)+128; v = (v>>7)-1 }; emit v.
func (c *cursor) vlint() (uint64, error) {
	var v, o uint64
	var s uint
	for {
		b, err := c.u8()
		if err != nil {
			return 0, err
		}
		if b < 128 {
			return v + o + (uint64(b) << s), nil
		}
		if s > 56 {
			return 0, fmt.Errorf("%w: integer", errGuest)
		}
		v += uint64(b&127) << s
		s += 7
		o = (o + 1) << 7
	}
}

func (c *cursor) vlintN() (int, error) {
	v, err := c.vlint()
	if err != nil {
		return 0, err
	}
	if v > maxBlob {
		return 0, fmt.Errorf("%w: size", errGuest)
	}
	return int(v), nil
}

func putVlint(v uint64) []byte {
	var b []byte
	for v >= 128 {
		b = append(b, byte(v&127)+128)
		v = (v >> 7) - 1
	}
	return append(b, byte(v))
}

func restore(ctx context.Context, in []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(in) < 7 || string(in[:3]) != "PCF" {
		return nil, fmt.Errorf("pref: bad magic")
	}
	if in[3] != 0 || in[4] != 4 || in[5] != 8 {
		return nil, fmt.Errorf("%w: version %d.%d.%d", errGuest, in[3], in[4], in[5])
	}
	method := in[6]
	i := 7
	for i < len(in) && in[i] != 0 {
		i++
	}
	if i >= len(in) {
		return nil, fmt.Errorf("%w: truncated", errGuest)
	}
	body := in[i+1:]
	switch method {
	case otfNone:
	case otfBzip2:
		dec, err := io.ReadAll(bzip2.NewReader(bytes.NewReader(body)))
		if err != nil {
			return nil, fmt.Errorf("%w: bzip2: %v", errGuest, err)
		}
		body = dec
	case otfXZ:
		r, err := xz.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("%w: xz: %v", errGuest, err)
		}
		dec, err := io.ReadAll(r)
		if err != nil {
			return nil, fmt.Errorf("%w: xz: %v", errGuest, err)
		}
		body = dec
	default:
		return nil, fmt.Errorf("%w: compression %d", errGuest, method)
	}
	return chunks(ctx, body)
}

func chunks(ctx context.Context, in []byte) ([]byte, error) {
	c := &cursor{b: in}
	var out bytes.Buffer
	for c.i < len(c.b) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		hdr, err := c.u8()
		if err != nil {
			return nil, err
		}
		if hdr == 0 {
			n, err := c.vlintN()
			if err != nil {
				return nil, err
			}
			if n == 0 {
				break
			}
			part, err := c.take(n)
			if err != nil {
				return nil, err
			}
			out.Write(part)
			continue
		}
		kind, err := c.u8()
		if err != nil {
			return nil, err
		}
		if err := oneChunk(hdr, kind, c, &out); err != nil {
			return nil, err
		}
	}
	return out.Bytes(), nil
}

func oneChunk(flags, kind byte, in *cursor, out *bytes.Buffer) error {
	switch kind {
	case dPDF:
		return chunkPDF(flags, in, out)
	case dZIP:
		return deflateRec(flags, in, out, []byte{'P', 'K', 3, 4}, false)
	case dGZIP:
		return deflateRec(flags, in, out, []byte{31, 139}, false)
	case dPNG:
		return chunkPNG(flags, in, out)
	case dMultiPNG:
		return chunkMultiPNG(flags, in, out)
	case dSWF:
		return deflateRec(flags, in, out, []byte{'C', 'W', 'S'}, true)
	case dBase64:
		return chunkBase64(flags, in, out)
	case dBrute:
		return deflateRec(flags, in, out, nil, false)
	case dRaw:
		return deflateRec(flags, in, out, nil, true)
	case dGIF:
		return chunkGIF(flags, in, out)
	case dJPG:
		return chunkJPG(flags, in, out)
	case dBzip2:
		return chunkBzip2(flags, in, out)
	case dMP3:
		return chunkMP3(flags, in, out)
	default:
		return fmt.Errorf("%w: unsupported stream %d", errGuest, kind)
	}
}

func (c *cursor) blob() ([]byte, error) {
	n, err := c.vlintN()
	if err != nil {
		return nil, err
	}
	return c.take(n)
}

// applyPenalty writes the 5-byte records stored for GIF and bzip2:
// big-endian position, then the replacement byte.
func applyPenalty(buf, penalty []byte) ([]byte, error) {
	if len(penalty)%5 != 0 {
		return nil, fmt.Errorf("%w: penalty", errGuest)
	}
	out := append([]byte(nil), buf...)
	for i := 0; i < len(penalty); i += 5 {
		pos := int(penalty[i])<<24 | int(penalty[i+1])<<16 | int(penalty[i+2])<<8 | int(penalty[i+3])
		if pos < 0 || pos >= len(out) {
			return nil, fmt.Errorf("%w: penalty pos", errGuest)
		}
		out[pos] = penalty[i+4]
	}
	return out, nil
}

func writeLimit(out *bytes.Buffer, produced []byte, n int) {
	if n >= 0 && n < len(produced) {
		produced = produced[:n]
	}
	out.Write(produced)
}

func chunkGIF(flags byte, in *cursor, out *bytes.Buffer) error {
	block := 255
	if flags&4 != 0 {
		block = 254
	}
	diff, err := in.blob()
	if err != nil {
		return err
	}
	var penalty []byte
	if flags&2 != 0 {
		penalty, err = in.blob()
		if err != nil {
			return err
		}
	}
	recompressed, err := in.vlintN()
	if err != nil {
		return err
	}
	decompressed, err := in.vlintN()
	if err != nil {
		return err
	}
	src, err := in.take(decompressed)
	if err != nil {
		return err
	}
	gif, err := gifrec.Recompress(src, block, diff)
	if err != nil {
		return fmt.Errorf("%w: %v", errGuest, err)
	}
	if recompressed < len(gif) {
		gif = gif[:recompressed]
	}
	if penalty != nil {
		gif, err = applyPenalty(gif, penalty)
		if err != nil {
			return err
		}
	}
	out.Write(gif)
	return nil
}

func chunkJPG(flags byte, in *cursor, out *bytes.Buffer) error {
	recompressed, err := in.vlintN()
	if err != nil {
		return err
	}
	payload, err := in.blob()
	if err != nil {
		return err
	}
	var jpeg []byte
	if flags&8 != 0 {
		jpeg, err = jpgrec.RestoreBrunsli(payload, flags&16 != 0)
	} else {
		jpeg, err = jpgrec.RestorePackJPG(payload)
	}
	if err != nil {
		return fmt.Errorf("%w: %v", errGuest, err)
	}
	if flags&4 != 0 {
		jpeg, err = jpgrec.StripMotionDHT(jpeg, recompressed)
		if err != nil {
			return fmt.Errorf("%w: %v", errGuest, err)
		}
		out.Write(jpeg)
		return nil
	}
	writeLimit(out, jpeg, recompressed)
	return nil
}

func chunkBzip2(flags byte, in *cursor, out *bytes.Buffer) error {
	level, err := in.u8()
	if err != nil {
		return err
	}
	var penalty []byte
	if flags&2 != 0 {
		penalty, err = in.blob()
		if err != nil {
			return err
		}
	}
	recompressed, err := in.vlintN()
	if err != nil {
		return err
	}
	decompressed, err := in.vlintN()
	if err != nil {
		return err
	}
	var src []byte
	if flags&128 != 0 {
		n, err := in.vlintN()
		if err != nil {
			return err
		}
		blob, err := in.take(n)
		if err != nil {
			return err
		}
		src, err = chunks(context.Background(), blob)
		if err != nil {
			return err
		}
	} else {
		src, err = in.take(decompressed)
		if err != nil {
			return err
		}
	}
	if len(src) > decompressed {
		src = src[:decompressed]
	}
	comp, err := bz2.Compress(src, int(level))
	if err != nil {
		return fmt.Errorf("%w: %v", errGuest, err)
	}
	if recompressed < len(comp) {
		comp = comp[:recompressed]
	}
	if penalty != nil {
		comp, err = applyPenalty(comp, penalty)
		if err != nil {
			return err
		}
	}
	out.Write(comp)
	return nil
}

func chunkMP3(_ byte, in *cursor, out *bytes.Buffer) error {
	recompressed, err := in.vlintN()
	if err != nil {
		return err
	}
	payload, err := in.blob()
	if err != nil {
		return err
	}
	mp3, err := mp3rec.Restore(payload)
	if err != nil {
		return fmt.Errorf("%w: %v", errGuest, err)
	}
	writeLimit(out, mp3, recompressed)
	return nil
}

type deflateMeta struct {
	hdr   []byte
	recon []byte
	unc   int
}

func readDeflateMeta(flags byte, in *cursor, incLast bool) (deflateMeta, error) {
	var m deflateMeta
	hdr, perfect, err := readDeflateHdr(flags, in, incLast)
	if err != nil {
		return m, err
	}
	m.hdr = hdr
	m.recon, m.unc, err = readRecon(in, perfect)
	return m, err
}

func deflateRec(flags byte, in *cursor, out *bytes.Buffer, prefix []byte, incLast bool) error {
	m, err := readDeflateMeta(flags, in, incLast)
	if err != nil {
		return err
	}
	var raw []byte
	if flags&128 != 0 {
		raw, err = nested(in)
	} else {
		raw, err = in.take(m.unc)
	}
	if err != nil {
		return err
	}
	return writeDeflate(out, prefix, m.hdr, raw, m.recon)
}

func writeDeflate(out *bytes.Buffer, prefix, hdr, raw, recon []byte) error {
	enc, ok := xt2png.Reencode(raw, recon)
	if !ok {
		return fmt.Errorf("%w: preflate", errGuest)
	}
	out.Write(prefix)
	out.Write(hdr)
	out.Write(enc)
	return nil
}

func nested(in *cursor) ([]byte, error) {
	n, err := in.vlintN()
	if err != nil {
		return nil, err
	}
	blob, err := in.take(n)
	if err != nil {
		return nil, err
	}
	return chunks(context.Background(), blob)
}

func chunkPDF(flags byte, in *cursor, out *bytes.Buffer) error {
	m, err := readDeflateMeta(flags, in, false)
	if err != nil {
		return err
	}
	bmp := flags >> 6
	var width int
	switch bmp {
	case 1:
		hdr, err := in.take(54 + 1024)
		if err != nil {
			return err
		}
		width = int(uint32(hdr[18]) | uint32(hdr[19])<<8 | uint32(hdr[20])<<16 | uint32(hdr[21])<<24)
	case 2:
		hdr, err := in.take(54)
		if err != nil {
			return err
		}
		width = int(uint32(hdr[18]) | uint32(hdr[19])<<8 | uint32(hdr[20])<<16 | uint32(hdr[21])<<24)
		width *= 3
	}
	var raw []byte
	if bmp == 0 || width%4 == 0 {
		raw, err = readSkip(in, m.unc, m.unc, 0)
	} else {
		raw, err = readSkip(in, m.unc, width, (-width)&3)
	}
	if err != nil {
		return err
	}
	return writeDeflate(out, []byte("/FlateDecode"), m.hdr, raw, m.recon)
}

// readSkip drops per-row padding. skip == 0 and line == count reads the
// whole buffer; a zero-length padding read must not end the copy early.
func readSkip(in *cursor, count, line, skip int) ([]byte, error) {
	if count == 0 {
		return []byte{}, nil
	}
	if line <= 0 {
		return nil, fmt.Errorf("%w: pdf row", errGuest)
	}
	out := make([]byte, 0, count)
	offset := 0
	for len(out) < count {
		if count-len(out) >= line-offset {
			n := line - offset
			if n > 0 {
				part, err := in.take(n)
				if err != nil {
					return nil, err
				}
				out = append(out, part...)
			}
			if skip > 0 {
				if _, err := in.take(skip); err != nil {
					return nil, err
				}
			}
			offset = 0
			continue
		}
		part, err := in.take(count - len(out))
		if err != nil {
			return nil, err
		}
		out = append(out, part...)
		offset += len(part)
	}
	return out, nil
}

func chunkPNG(flags byte, in *cursor, out *bytes.Buffer) error {
	m, err := readDeflateMeta(flags, in, true)
	if err != nil {
		return err
	}
	raw, err := in.take(m.unc)
	if err != nil {
		return err
	}
	return writeDeflate(out, []byte("IDAT"), m.hdr, raw, m.recon)
}

func chunkMultiPNG(flags byte, in *cursor, out *bytes.Buffer) error {
	hdr, perfect, err := readDeflateHdr(flags, in, true)
	if err != nil {
		return err
	}
	count, err := in.vlintN()
	if err != nil {
		return err
	}
	count++
	if count <= 0 || count > 1<<20 {
		return fmt.Errorf("%w: idat count", errGuest)
	}
	lengths := make([]int, count)
	crcs := make([]uint32, count)
	first, err := in.vlintN()
	if err != nil {
		return err
	}
	if first < 2 {
		return fmt.Errorf("%w: idat length", errGuest)
	}
	lengths[0] = first - 2
	for i := 1; i < count; i++ {
		c0, err := in.u8()
		if err != nil {
			return err
		}
		c1, err := in.u8()
		if err != nil {
			return err
		}
		c2, err := in.u8()
		if err != nil {
			return err
		}
		c3, err := in.u8()
		if err != nil {
			return err
		}
		crcs[i] = uint32(c0)<<24 | uint32(c1)<<16 | uint32(c2)<<8 | uint32(c3)
		lengths[i], err = in.vlintN()
		if err != nil {
			return err
		}
	}
	recon, unc, err := readRecon(in, perfect)
	if err != nil {
		return err
	}
	raw, err := in.take(unc)
	if err != nil {
		return err
	}
	enc, ok := xt2png.Reencode(raw, recon)
	if !ok {
		return fmt.Errorf("%w: preflate", errGuest)
	}
	out.WriteString("IDAT")
	out.Write(hdr)
	return writeMulti(out, enc, lengths, crcs)
}

func writeMulti(out *bytes.Buffer, enc []byte, lengths []int, crcs []uint32) error {
	idx := 0
	left := lengths[0]
	off := 0
	for off < len(enc) {
		if idx >= len(lengths) {
			return fmt.Errorf("%w: idat short", errGuest)
		}
		n := len(enc) - off
		if n > left {
			n = left
		}
		out.Write(enc[off : off+n])
		off += n
		left -= n
		if left > 0 {
			continue
		}
		idx++
		if idx >= len(lengths) {
			break
		}
		var be [4]byte
		be[0] = byte(crcs[idx] >> 24)
		be[1] = byte(crcs[idx] >> 16)
		be[2] = byte(crcs[idx] >> 8)
		be[3] = byte(crcs[idx])
		out.Write(be[:])
		left = lengths[idx]
		be[0] = byte(uint32(left) >> 24)
		be[1] = byte(uint32(left) >> 16)
		be[2] = byte(uint32(left) >> 8)
		be[3] = byte(left)
		out.Write(be[:])
		out.WriteString("IDAT")
	}
	return nil
}

func readDeflateHdr(flags byte, in *cursor, incLast bool) (hdr []byte, perfect bool, err error) {
	perfect = flags&2 == 0
	if perfect {
		if _, err = in.u8(); err != nil {
			return nil, false, err
		}
	}
	n, err := in.vlintN()
	if err != nil {
		return nil, false, err
	}
	if incLast {
		if n == 0 {
			return nil, false, fmt.Errorf("%w: empty header", errGuest)
		}
		body, err := in.take(n - 1)
		if err != nil {
			return nil, false, err
		}
		last, err := in.u8()
		if err != nil {
			return nil, false, err
		}
		hdr = append(append([]byte{}, body...), last-1)
		return hdr, perfect, nil
	}
	hdr, err = in.take(n)
	if err != nil {
		return nil, false, err
	}
	return append([]byte{}, hdr...), perfect, nil
}

func readRecon(in *cursor, perfect bool) (recon []byte, unc int, err error) {
	if !perfect {
		rn, err := in.vlintN()
		if err != nil {
			return nil, 0, err
		}
		recon, err = in.take(rn)
		if err != nil {
			return nil, 0, err
		}
		recon = append([]byte{}, recon...)
	}
	if _, err = in.vlint(); err != nil {
		return nil, 0, err
	}
	unc, err = in.vlintN()
	return recon, unc, err
}

func chunkBase64(flags byte, in *cursor, out *bytes.Buffer) error {
	lineCase := int((flags >> 2) & 3)
	n, err := in.vlintN()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("%w: base64 header", errGuest)
	}
	hdr, err := in.take(n)
	if err != nil {
		return err
	}
	out.WriteByte(hdr[0] + 1)
	out.Write(hdr[1:])
	lines, err := in.vlintN()
	if err != nil {
		return err
	}
	if lines <= 0 || lines > 1<<20 {
		return fmt.Errorf("%w: base64 lines", errGuest)
	}
	lens := make([]int, lines)
	if lineCase == 2 {
		for i := range lens {
			b, err := in.u8()
			if err != nil {
				return err
			}
			lens[i] = int(b)
		}
	} else {
		b, err := in.u8()
		if err != nil {
			return err
		}
		for i := range lens {
			lens[i] = int(b)
		}
		if lineCase == 1 {
			last, err := in.u8()
			if err != nil {
				return err
			}
			lens[len(lens)-1] = int(last)
		}
	}
	inN, err := in.vlintN()
	if err != nil {
		return err
	}
	outN, err := in.vlintN()
	if err != nil {
		return err
	}
	var src []byte
	if flags&128 != 0 {
		src, err = nested(in)
	} else {
		src, err = in.take(inN)
	}
	if err != nil {
		return err
	}
	if flags&128 != 0 {
		inN = len(src)
	}
	out.Write(base64Reencode(src, lens, inN, outN))
	return nil
}

func base64Reencode(src []byte, lineLen []int, maxIn, maxOut int) []byte {
	if maxIn < len(src) {
		src = src[:maxIn]
	}
	var out []byte
	remaining := len(src)
	off := 0
	lineNr := 0
	actLine := 0
	actCount := 0
	emit := func(b byte) {
		if actCount < maxOut {
			out = append(out, b)
		}
		actCount++
	}
	for {
		avail := remaining
		if avail > div3 {
			avail = div3
		}
		buf := make([]byte, avail)
		copy(buf, src[off:off+avail])
		off += avail
		remaining -= avail
		for len(buf)%3 != 0 {
			buf = append(buf, 0)
		}
		stop := false
		for i := 0; i < len(buf)/3; i++ {
			a, b, c := buf[i*3], buf[i*3+1], buf[i*3+2]
			chars := [4]byte{
				b64alpha[a>>2],
				b64alpha[((a&0x03)<<4)|(b>>4)],
				b64alpha[((b&0x0f)<<2)|(c>>6)],
				b64alpha[c&63],
			}
			for _, ch := range chars {
				emit(ch)
				actLine++
				if actLine == lineLen[lineNr] {
					emit('\r')
					emit('\n')
					actLine = 0
					lineNr++
					if lineNr == len(lineLen) {
						stop = true
						break
					}
				}
			}
			if stop {
				break
			}
		}
		if stop || remaining <= 0 || avail == 0 {
			break
		}
	}
	return out
}
