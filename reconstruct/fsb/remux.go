package fsb

import (
	"bytes"
	"encoding/binary"
	"errors"
)

// errInvalid is a non-zero guest status: empty, truncated, or not Ogg Vorbis.
var errInvalid = errors.New("invalid ogg vorbis")

// Inverted reflected step from oggvorbis2fsb5.c crc32_for_byte, then ^ 0xFF000000.
func crc32ForByte(r uint32) uint32 {
	for j := 0; j < 8; j++ {
		x := uint32(0)
		if r&1 == 0 {
			x = 0xEDB88320
		}
		r = x ^ r>>1
	}
	return r ^ 0xFF000000
}

var setupCRCTable = func() [256]uint32 {
	var t [256]uint32
	for i := range t {
		t[i] = crc32ForByte(uint32(i))
	}
	return t
}()

func setupCRC(data []byte) uint32 {
	var crc uint32
	for _, b := range data {
		crc = setupCRCTable[byte(crc)^b] ^ crc>>8
	}
	return crc
}

// Ogg page checksum: polynomial 0x04c11db7, not reflected, init 0.
func oggCRC(p []byte) uint32 {
	const poly = 0x04c11db7
	var crc uint32
	for _, b := range p {
		crc ^= uint32(b) << 24
		for i := 0; i < 8; i++ {
			if crc&0x80000000 != 0 {
				crc = (crc << 1) ^ poly
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

type oggPage struct {
	version   int
	continued bool
	bos       bool
	eos       bool
	granule   int64
	serial    int32
	pageno    int64
	segments  []int
	body      []byte
}

type syncBuf struct {
	buf         []byte
	returned    int
	headerbytes int
	bodybytes   int
	unsynced    bool
	src         []byte
	srcPos      int
}

func (s *syncBuf) take(n int) int {
	if s.returned > 0 {
		s.buf = append([]byte(nil), s.buf[s.returned:]...)
		s.returned = 0
	}
	end := s.srcPos + n
	if end > len(s.src) {
		end = len(s.src)
	}
	s.buf = append(s.buf, s.src[s.srcPos:end]...)
	got := end - s.srcPos
	s.srcPos = end
	return got
}

func (s *syncBuf) syncFail() int {
	s.headerbytes = 0
	s.bodybytes = 0
	page := s.returned
	n := len(s.buf) - page
	next := -1
	for i := 1; i < n; i++ {
		if s.buf[page+i] == 'O' {
			next = page + i
			break
		}
	}
	if next < 0 {
		s.returned = len(s.buf)
		if n == 0 {
			return 0
		}
		return -n
	}
	s.returned = next
	return -(next - page)
}

func (s *syncBuf) pageseek() (*oggPage, int) {
	if s.headerbytes == 0 {
		n := len(s.buf) - s.returned
		if n < 27 {
			return nil, 0
		}
		b := s.buf[s.returned:]
		if !bytes.Equal(b[:4], []byte("OggS")) {
			return nil, s.syncFail()
		}
		headerbytes := int(b[26]) + 27
		if n < headerbytes {
			return nil, 0
		}
		body := 0
		for i := 0; i < int(b[26]); i++ {
			body += int(b[27+i])
		}
		s.bodybytes = body
		s.headerbytes = headerbytes
	}
	need := s.headerbytes + s.bodybytes
	if need > len(s.buf)-s.returned {
		return nil, 0
	}
	b := s.buf[s.returned : s.returned+need]
	var chk [4]byte
	copy(chk[:], b[22:26])
	b[22], b[23], b[24], b[25] = 0, 0, 0, 0
	sum := oggCRC(b)
	var got [4]byte
	got[0] = byte(sum)
	got[1] = byte(sum >> 8)
	got[2] = byte(sum >> 16)
	got[3] = byte(sum >> 24)
	if chk != got {
		copy(b[22:26], chk[:])
		s.headerbytes = 0
		s.bodybytes = 0
		return nil, s.syncFail()
	}
	copy(b[22:26], got[:])
	hdr := bytes.Clone(b[:s.headerbytes])
	body := bytes.Clone(b[s.headerbytes:])
	s.returned += need
	s.headerbytes = 0
	s.bodybytes = 0
	s.unsynced = false
	return parsePage(hdr, body), need
}

func parsePage(h, body []byte) *oggPage {
	segs := make([]int, h[26])
	for i := range segs {
		segs[i] = int(h[27+i])
	}
	flags := h[5]
	return &oggPage{
		version:   int(h[4]),
		continued: flags&0x01 != 0,
		bos:       flags&0x02 != 0,
		eos:       flags&0x04 != 0,
		granule:   int64(binary.LittleEndian.Uint64(h[6:14])),
		serial:    int32(binary.LittleEndian.Uint32(h[14:18])),
		pageno:    int64(binary.LittleEndian.Uint32(h[18:22])),
		segments:  segs,
		body:      body,
	}
}

func (s *syncBuf) pageout() (*oggPage, int) {
	for {
		p, ret := s.pageseek()
		if ret > 0 {
			return p, 1
		}
		if ret == 0 {
			return nil, 0
		}
		if !s.unsynced {
			s.unsynced = true
			return nil, -1
		}
	}
}

type rawPkt struct {
	data []byte
	gp   int64
	bos  bool
	gap  bool
}

// stream is one logical bitstream. nextPage starts at 0, as ogg_stream_init does.
type stream struct {
	serial     int32
	nextPage   int64
	q          []rawPkt
	open       []byte
	openActive bool
	openBOS    bool
}

func (st *stream) packetout() (rawPkt, int) {
	if len(st.q) == 0 {
		return rawPkt{}, 0
	}
	p := st.q[0]
	st.q[0] = rawPkt{}
	st.q = st.q[1:]
	if p.gap {
		return p, -1
	}
	return p, 1
}

// pagein mirrors ogg_stream_pagein. Granulepos is stored only on the last
// completed packet of the page (lace < 255); every other completed packet
// stays at -1. A packet that spans pages keeps the granulepos of the page
// that finishes it, unless a later packet on that page also completes.
func (st *stream) pagein(p *oggPage) error {
	if p.serial != st.serial || p.version > 0 {
		return errInvalid
	}
	if p.pageno != st.nextPage {
		st.open = nil
		st.openActive = false
		st.openBOS = false
		st.q = append(st.q, rawPkt{gap: true})
	}
	bodyPos := 0
	segptr := 0
	contSkip := false
	if p.continued && !st.openActive {
		contSkip = true
		for segptr < len(p.segments) {
			val := p.segments[segptr]
			bodyPos += val
			segptr++
			if val < 255 {
				break
			}
		}
	}
	startSeg := segptr
	cur := []byte(nil)
	curBOS := false
	if st.openActive {
		cur = st.open
		curBOS = st.openBOS
	}
	var done []rawPkt
	for ; segptr < len(p.segments); segptr++ {
		val := p.segments[segptr]
		if bodyPos+val > len(p.body) {
			return errInvalid
		}
		cur = append(cur, p.body[bodyPos:bodyPos+val]...)
		bodyPos += val
		if segptr == startSeg && !st.openActive && p.bos && !contSkip {
			curBOS = true
		}
		if val < 255 {
			done = append(done, rawPkt{data: bytes.Clone(cur), gp: -1, bos: curBOS})
			cur = nil
			curBOS = false
		}
	}
	if len(done) > 0 {
		done[len(done)-1].gp = p.granule
	}
	if segptr != startSeg {
		if p.segments[len(p.segments)-1] == 255 {
			st.open = cur
			st.openActive = true
			st.openBOS = curBOS
		} else {
			st.open = nil
			st.openActive = false
			st.openBOS = false
		}
	}
	st.q = append(st.q, done...)
	st.nextPage = p.pageno + 1
	return nil
}

func parseID(pkt []byte, bos bool) (int, uint32, error) {
	if !bos || len(pkt) < 30 || pkt[0] != 1 || string(pkt[1:7]) != "vorbis" {
		return 0, 0, errInvalid
	}
	if binary.LittleEndian.Uint32(pkt[7:11]) != 0 {
		return 0, 0, errInvalid
	}
	ch := int(pkt[11])
	rate := binary.LittleEndian.Uint32(pkt[12:16])
	bs0 := int(pkt[28] & 0x0f)
	bs1 := int(pkt[28] >> 4)
	block0 := 1 << bs0
	block1 := 1 << bs1
	// wasm32 long is signed: a rate with the high bit set is negative and rejected.
	if rate < 1 || rate > 0x7fffffff || ch < 1 || block0 < 64 || block1 < block0 || block1 > 8192 || pkt[29]&1 != 1 {
		return 0, 0, errInvalid
	}
	return ch, rate, nil
}

func parseComment(pkt []byte) error {
	if len(pkt) < 11 || pkt[0] != 3 || string(pkt[1:7]) != "vorbis" {
		return errInvalid
	}
	vendorLen := int(binary.LittleEndian.Uint32(pkt[7:11]))
	if vendorLen < 0 || vendorLen > len(pkt)-8 {
		return errInvalid
	}
	off := 11 + vendorLen
	if off+4 > len(pkt) {
		return errInvalid
	}
	n := int(binary.LittleEndian.Uint32(pkt[off : off+4]))
	off += 4
	if n < 0 || n > ((len(pkt)-off)>>2) {
		return errInvalid
	}
	for i := 0; i < n; i++ {
		if off+4 > len(pkt) {
			return errInvalid
		}
		ln := int(binary.LittleEndian.Uint32(pkt[off : off+4]))
		off += 4
		if ln < 0 || ln > len(pkt)-off {
			return errInvalid
		}
		off += ln
	}
	if off >= len(pkt) || pkt[off]&1 != 1 {
		return errInvalid
	}
	return nil
}

func parseSetup(pkt []byte) error {
	if len(pkt) < 7 || pkt[0] != 5 || string(pkt[1:7]) != "vorbis" {
		return errInvalid
	}
	return nil
}

func freqCode(rate uint32) uint64 {
	switch rate {
	case 48000:
		return 9
	case 44100:
		return 8
	case 32000:
		return 7
	case 24000:
		return 6
	case 22050:
		return 5
	case 16000:
		return 4
	case 11025:
		return 3
	case 11000:
		return 2
	case 8000:
		return 1
	default:
		return 0
	}
}

func putU32(b []byte, v uint32) []byte {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	return append(b, buf[:]...)
}

func encodeFSB(channels int, rate uint32, crc uint32, pkts []rawPkt) []byte {
	packetCount := 0
	totalData := 0
	for _, p := range pkts {
		if p.gp != -1 {
			packetCount++
		}
		totalData += 2 + len(p.data)
	}
	// 32 when totalData is already a multiple of 32.
	dataPad := 32 - (totalData % 32)

	freq := freqCode(rate)
	var stereo uint64
	if channels == 2 {
		stereo = 1
	}
	sampleSize := 8
	var chunks []byte
	if channels != 1 && channels != 2 {
		chunks = putU32(chunks, 1|(1<<1)|(1<<25))
		chunks = append(chunks, byte(channels))
		sampleSize += 5
	}
	if freq == 0 {
		chunks = putU32(chunks, 1|(4<<1)|(2<<25))
		chunks = putU32(chunks, rate)
		sampleSize += 8
	}
	vorbisPayload := 8 + 8*packetCount
	sampleSize += 4 + vorbisPayload
	// 16 when the header is already a multiple of 16.
	pad := 16 - ((60 + sampleSize) % 16)
	sampleSize += pad

	var maxGP int32
	entries := make([]byte, 0, 8*packetCount)
	off := 0
	for _, p := range pkts {
		if p.gp != -1 {
			if int64(maxGP) < p.gp {
				maxGP = int32(p.gp)
			}
			var e [8]byte
			binary.LittleEndian.PutUint32(e[0:4], uint32(p.gp))
			binary.LittleEndian.PutUint32(e[4:8], uint32(off))
			entries = append(entries, e[:]...)
		}
		off += 2 + len(p.data)
	}
	samples := uint64(uint32(maxGP)) & 0x3fffffff
	fsh := uint64(1) | (freq << 1) | (stereo << 5) | (samples << 34)

	out := make([]byte, 0, 60+sampleSize+totalData+dataPad)
	out = putU32(out, 0x35425346)
	out = putU32(out, 1)
	out = putU32(out, 1)
	out = putU32(out, uint32(sampleSize))
	out = putU32(out, 0)
	out = putU32(out, uint32(totalData+dataPad))
	out = putU32(out, 15)
	out = append(out, 1, 0, 0, 0, 0, 0, 0, 0)
	out = append(out, make([]byte, 24)...)
	var sh [8]byte
	binary.LittleEndian.PutUint64(sh[:], fsh)
	out = append(out, sh[:]...)
	out = append(out, chunks...)
	out = putU32(out, uint32(vorbisPayload)<<1|(11<<25))
	out = putU32(out, crc)
	out = putU32(out, uint32(8*packetCount))
	out = append(out, entries...)
	out = append(out, make([]byte, pad)...)
	for _, p := range pkts {
		var ln [2]byte
		binary.LittleEndian.PutUint16(ln[:], uint16(len(p.data)))
		out = append(out, ln[:]...)
		out = append(out, p.data...)
	}
	out = append(out, make([]byte, dataPad)...)
	return out
}

func remux(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errInvalid
	}
	sy := &syncBuf{src: data}
	var audio []rawPkt
	var channels int
	var rate uint32
	var crc uint32
	saw := false
	for {
		n := sy.take(4096)
		pg, res := sy.pageout()
		if res != 1 {
			if n < 4096 {
				break
			}
			return nil, errInvalid
		}
		st := &stream{serial: pg.serial}
		if err := st.pagein(pg); err != nil {
			return nil, errInvalid
		}
		id, r := st.packetout()
		if r != 1 {
			return nil, errInvalid
		}
		ch, rt, err := parseID(id.data, id.bos)
		if err != nil {
			return nil, errInvalid
		}
		channels, rate = ch, rt
		saw = true

		got := 0
		var setup []byte
		for got < 2 {
			for got < 2 {
				pg, res = sy.pageout()
				if res == 0 {
					break
				}
				if res != 1 {
					continue
				}
				_ = st.pagein(pg)
				for got < 2 {
					pkt, pr := st.packetout()
					if pr == 0 {
						break
					}
					if pr < 0 {
						return nil, errInvalid
					}
					var herr error
					if got == 0 {
						herr = parseComment(pkt.data)
					} else {
						herr = parseSetup(pkt.data)
						setup = pkt.data
					}
					if herr != nil {
						return nil, errInvalid
					}
					got++
				}
			}
			n = sy.take(4096)
			if n == 0 && got < 2 {
				return nil, errInvalid
			}
		}
		crc = setupCRC(setup)

		// Queued packets are pulled only after a later page is submitted, as in
		// oggvorbis2fsb5's decode loop. Anything still queued at EOS or EOF is dropped.
		eos := false
		for !eos {
			for !eos {
				pg, res = sy.pageout()
				if res == 0 {
					break
				}
				if res < 0 {
					continue
				}
				_ = st.pagein(pg)
				for {
					pkt, pr := st.packetout()
					if pr == 0 {
						break
					}
					if pr > 0 {
						audio = append(audio, pkt)
					}
				}
				if pg.eos {
					eos = true
				}
			}
			if !eos {
				if sy.take(4096) == 0 {
					eos = true
				}
			}
		}
	}
	if !saw || len(audio) == 0 {
		return nil, errInvalid
	}
	return encodeFSB(channels, rate, crc, audio), nil
}
