package xt2png

import "fmt"

// preflate reencodes unpacked bytes and a preflate diff into a raw deflate
// stream. It is the single-threaded preflate_reencode path (extraThreadCount
// is 0, so each meta block is decoded and written inline).

func preflate(raw, diff []byte, dstCap int) ([]byte, error) {
	if dstCap <= 0 {
		dstCap = len(raw) + len(diff) + 64
	}
	out, ok := preflateReencode(raw, diff)
	if !ok {
		return nil, fmt.Errorf("%w: preflate %d", errGuest, -1)
	}
	if len(out) > dstCap || len(out) > 0x7fffffff {
		return nil, fmt.Errorf("%w: preflate %d", errGuest, -2)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: preflate %d", errGuest, 0)
	}
	return out, nil
}

type metaBlock struct {
	model     predModel
	params    pfParams
	reconOff  int
	reconSize int
	uncSize   int
}

func parseMeta(diff []byte, uncSize int) ([]metaBlock, bool) {
	if len(diff) == 0 || uncSize < 0 {
		return nil, false
	}
	br := newBitReader(diff)
	if br.get(1) != 0 {
		return nil, false
	}
	count := uint64(1)
	if br.get(1) != 0 {
		count = 2 + br.getVLI()
	}
	if count == 0 || count > 1<<20 {
		return nil, false
	}
	blocks := make([]metaBlock, 0, int(count))
	for i := uint64(0); i < count; i++ {
		var mb metaBlock
		if i > 0 && br.get(2) != 3 {
			return nil, false
		}
		var mc modelCodec
		if br.get(1) == 0 {
			mb.params.zlib = true
			mc.blockFull, mc.treeFull, mc.tokenFull = true, true, true
			mb.params.comp = uint8(br.get(4))
			mb.params.mem = uint8(br.get(4))
			mb.params.wbits = uint8(br.get(3) + 8)
			mb.model.readFrom(&mc, nil)
		} else {
			mb.params.comp = uint8(br.get(4))
			mb.params.mem = uint8(br.get(4))
			mb.params.wbits = uint8(br.get(3) + 8)
			mb.params.zlib = br.get(1) != 0
			if !mb.params.zlib {
				mb.params.veryFar = br.get(1) != 0
				mb.params.toStart = br.get(1) != 0
			}
			mb.params.log2 = uint8(br.get(4))
			sz := br.getVLI()
			blob := br.copyBytes(sz)
			dec := newArith(newBitReader(blob))
			mc.readFrom(dec)
			mb.model.readFrom(&mc, dec)
		}
		if i+1 != count {
			rs, ok := fitInt(br.getVLI())
			if !ok {
				return nil, false
			}
			us, ok := fitInt(br.getVLI())
			if !ok {
				return nil, false
			}
			mb.reconSize = rs
			mb.uncSize = us
		}
		blocks = append(blocks, mb)
	}
	br.skipToByte()
	recon := br.pos >> 3
	unc := 0
	for i := range blocks {
		blocks[i].reconOff = recon
		if i+1 != len(blocks) {
			recon += blocks[i].reconSize
			unc += blocks[i].uncSize
			if blocks[i].reconSize < 0 || blocks[i].uncSize < 0 || recon > len(diff) || unc > uncSize {
				return nil, false
			}
		} else {
			if recon > len(diff) || unc > uncSize {
				return nil, false
			}
			blocks[i].reconSize = len(diff) - recon
			blocks[i].uncSize = uncSize - unc
		}
	}
	return blocks, true
}

func fitInt(v uint64) (int, bool) {
	const maxInt = int(^uint(0) >> 1)
	if v > uint64(maxInt) {
		return 0, false
	}
	return int(v), true
}

func preflateReencode(raw, diff []byte) ([]byte, bool) {
	if diff == nil {
		diff = []byte{}
	}
	blocks, ok := parseMeta(diff, len(raw))
	if !ok {
		return nil, false
	}
	var w bitWriter
	var kept []byte
	inOff := 0
	for j := range blocks {
		add := blocks[j].uncSize
		if add < 0 || add > len(raw)-inOff {
			return nil, false
		}
		buf := make([]byte, len(kept)+add)
		copy(buf, kept)
		copy(buf[len(kept):], raw[inOff:inOff+add])
		inOff += add
		if !reencodeMeta(&w, &blocks[j], diff, buf, len(kept), j+1 == len(blocks)) {
			return nil, false
		}
		if j+1 < len(blocks) {
			const win = 1 << 15
			if len(buf) > win {
				kept = append([]byte(nil), buf[len(buf)-win:]...)
			} else {
				kept = append([]byte(nil), buf...)
			}
		}
	}
	w.flush()
	return w.buf, true
}

func reencodeMeta(w *bitWriter, mb *metaBlock, diff, data []byte, offset int, last bool) bool {
	if mb.reconOff < 0 || mb.reconSize < 0 || mb.reconOff > len(diff) {
		return false
	}
	end := mb.reconOff + mb.reconSize
	if end < mb.reconOff || end > len(diff) {
		return false
	}
	dec := newArith(newBitReader(diff[mb.reconOff:end]))
	tp := newTokenPred(mb.params, data, offset)
	tr := newTreePred(data, offset)
	if tp.fail {
		return false
	}
	var blocks []pfBlock
	for {
		b, ok := tp.decodeBlock(&mb.model, dec)
		if !ok || tp.fail {
			return false
		}
		if !tr.decodeBlock(&b, &mb.model, dec) || tr.fail {
			return false
		}
		blocks = append(blocks, b)
		eof := tp.inputEOF()
		if last {
			eof = tp.decodeEOF(&mb.model, dec)
		}
		if eof {
			break
		}
		if len(blocks) > 1<<20 {
			return false
		}
	}
	var padN, padV uint32
	if last && mb.model.block.nz.decode(dec) != 0 {
		padN = dec.decodeBits(3)
		if padN > 0 {
			padV = (uint32(1) << (padN - 1)) + dec.decodeBits(padN-1)
		}
	}
	pos := offset
	for i := range blocks {
		if !writeBlock(w, blocks[i], data, &pos, last && i == len(blocks)-1) {
			return false
		}
	}
	w.put(padV, padN)
	return true
}
