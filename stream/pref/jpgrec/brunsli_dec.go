package jpgrec

import "fmt"

// BrunsliDecodeJpeg + helpers, translated from brunsli_decode.cc.
// C int is int32. brotli=false copies metadata bytes; brotli=true is rejected
// on a compressed metadata section.

const (
	brMaxBlocks   = 1 << 21
	brMaxDim      = 65535
	brMaxSampling = 15
	brShortLimit  = 0x40 + 3*0x100
	brCoeffAlpha  = 18
	kACPredPrec   = 1 << 13
	kNumAvg       = 9
)

var kApp0Densities = [8]uint16{1, 72, 96, 100, 150, 180, 240, 300}

type jpegQuant struct {
	values    [64]int32
	precision int
	index     int
	isLast    bool
}

type jpegHuff struct {
	counts [17]int
	values [257]int
	slot   int
	isLast bool
}

type jpegCompScan struct{ comp, dc, ac int }

type jpegExtraZero struct{ block, n int }

type jpegScan struct {
	ss, se, ah, al int
	comps          []jpegCompScan
	reset          map[int]struct{}
	extra          []jpegExtraZero
}

type jpegComp struct {
	id, h, v, quant  int
	wBlocks, hBlocks int
	nBlocks          int
	coeffs           []int16
}

type jpegPic struct {
	w, h, version    int
	maxH, maxV       int
	mcuRows, mcuCols int
	restart          int
	app              [][]byte
	com              [][]byte
	quant            []jpegQuant
	huff             []jpegHuff
	comps            []jpegComp
	scans            []jpegScan
	order            []byte
	inter            [][]byte
	tail             []byte
	orig             []byte
	hasPad           bool
	pad              []int
}

type compMeta struct {
	h, v, width, height int
	ctxBits, ctxOff     int
	quant               [64]int32
	coeffs              []int16
	block               []byte
	acStride, bStride   int
}

type brState struct {
	stage      int
	data       []byte
	pos        int
	useBrotli  bool
	tags       int32
	tag        int
	sectionEnd int
	ctxMap     []byte
	codes      []ansCode
	meta       []compMeta
	numCtx     int
	numHist    int
	storage    bool
	warm       bool
	fail       error
}

func (s *brState) failf(format string, args ...any) int {
	s.fail = fmt.Errorf(format, args...)
	return brStageError
}

func (s *brState) can(n int) bool {
	if n < 0 || len(s.data) < n {
		return false
	}
	return s.pos <= len(s.data)-n
}

func (s *brState) u8() (byte, bool) {
	if s.pos >= len(s.data) {
		return 0, false
	}
	b := s.data[s.pos]
	s.pos++
	return b, true
}

func (s *brState) base128() (uint64, bool) {
	var val uint64
	b := uint64(0x80)
	for i := 0; i < 9 && b&0x80 != 0; i++ {
		by, ok := s.u8()
		if !ok {
			return 0, false
		}
		b = uint64(by)
		val |= (b & 0x7f) << uint(i*7)
	}
	return val, b&0x80 == 0
}

func (s *brState) dataLen() (int, bool) {
	n, ok := s.base128()
	if !ok || n > uint64(len(s.data)) {
		return 0, false
	}
	if !s.can(int(n)) {
		return 0, false
	}
	return int(n), true
}

func divCeil(a, b int) int { return (a + b - 1) / b }

func updateSubsampling(jpg *jpegPic) bool {
	jpg.maxH, jpg.maxV = 1, 1
	for i := range jpg.comps {
		c := &jpg.comps[i]
		if c.h > jpg.maxH {
			jpg.maxH = c.h
		}
		if c.v > jpg.maxV {
			jpg.maxV = c.v
		}
	}
	jpg.mcuRows = divCeil(jpg.h, jpg.maxV*8)
	jpg.mcuCols = divCeil(jpg.w, jpg.maxH*8)
	for i := range jpg.comps {
		c := &jpg.comps[i]
		c.wBlocks = jpg.mcuCols * c.h
		c.hBlocks = jpg.mcuRows * c.v
		n := int64(c.wBlocks) * int64(c.hBlocks)
		if n > brMaxBlocks {
			return false
		}
		c.nBlocks = int(n)
	}
	return true
}

func prepareMeta(jpg *jpegPic, st *brState) {
	st.meta = make([]compMeta, len(jpg.comps))
	for i := range jpg.comps {
		c := &jpg.comps[i]
		m := &st.meta[i]
		m.h = c.h
		m.v = c.v
		m.width = jpg.mcuCols * m.h
		m.height = jpg.mcuRows * m.v
	}
}

func warmupMeta(jpg *jpegPic, st *brState) {
	if !st.storage {
		st.storage = true
		for i := range st.meta {
			n := st.meta[i].width * st.meta[i].height
			jpg.comps[i].coeffs = make([]int16, n*64)
			st.meta[i].block = make([]byte, n)
		}
	}
	if !st.warm {
		st.warm = true
		for c := range st.meta {
			m := &st.meta[c]
			q := &jpg.quant[jpg.comps[c].quant]
			m.coeffs = jpg.comps[c].coeffs
			m.acStride = m.width * 64
			m.bStride = m.width
			m.quant = q.values
		}
	}
}

func verifySignature(st *brState) int {
	if !st.can(6) {
		return st.failf("jpgrec: truncated brunsli signature")
	}
	sig := [6]byte{0x0A, 0x04, 0x42, 0xD2, 0xD5, 0x4E}
	bad := false
	for i := 0; i < 6; i++ {
		if st.data[st.pos+i] != sig[i] {
			bad = true
		}
	}
	st.pos += 6
	st.tags |= 1 << 1
	if bad {
		return st.failf("jpgrec: invalid brunsli signature")
	}
	return brStageHeader
}

func decodeHeader(st *brState, jpg *jpegPic) int {
	b, ok := st.u8()
	if !ok || b != byte((2<<3)|2) {
		return st.failf("jpgrec: invalid brunsli header")
	}
	st.tags |= 1 << 2
	markerLen, ok := st.dataLen()
	if !ok {
		return st.failf("jpgrec: invalid brunsli header")
	}
	markerEnd := st.pos + markerLen
	var tags int32
	var vals [16]uint64
	for st.pos < markerEnd {
		m, ok := st.u8()
		if !ok {
			return st.failf("jpgrec: invalid brunsli header")
		}
		tag := int(m >> 3)
		if tag == 0 || tag > 15 {
			return st.failf("jpgrec: invalid brunsli header")
		}
		bit := int32(1) << uint(tag)
		wire := m & 7
		if wire != 0 && wire != 2 {
			return st.failf("jpgrec: invalid brunsli header")
		}
		if tags&bit != 0 {
			return st.failf("jpgrec: invalid brunsli header")
		}
		tags |= bit
		isVar := wire == 0
		if (bit == 1<<1 || bit == 1<<2 || bit == 1<<3 || bit == 1<<4) && !isVar {
			return st.failf("jpgrec: invalid brunsli header")
		}
		value, ok := st.base128()
		if !ok {
			return st.failf("jpgrec: invalid brunsli header")
		}
		if isVar {
			vals[tag] = value
		} else {
			if !st.can(int(value)) {
				return st.failf("jpgrec: invalid brunsli header")
			}
			st.pos += int(value)
		}
	}
	if st.pos != markerEnd {
		return st.failf("jpgrec: invalid brunsli header")
	}
	if tags&(1<<3) == 0 {
		return st.failf("jpgrec: invalid brunsli header")
	}
	vc := vals[3]
	version := int(vc >> 2)
	jpg.version = version
	if version == 1 {
		jpg.w, jpg.h = 0, 0
		return brStageFallback
	}
	if version != 0 {
		return st.failf("jpgrec: invalid brunsli header")
	}
	st.tags |= 1 << 9
	if tags&(1<<1) == 0 || tags&(1<<2) == 0 || tags&(1<<4) == 0 {
		return st.failf("jpgrec: invalid brunsli header")
	}
	width, height := vals[1], vals[2]
	if width == 0 || height == 0 || width > brMaxDim || height > brMaxDim {
		return st.failf("jpgrec: invalid brunsli header")
	}
	jpg.w = int(width)
	jpg.h = int(height)
	ncomp := int(vc&3) + 1
	jpg.comps = make([]jpegComp, ncomp)
	for i := range jpg.comps {
		jpg.comps[i].h = 1
		jpg.comps[i].v = 1
	}
	sub := vals[4]
	for i := range jpg.comps {
		c := &jpg.comps[i]
		c.v = int(sub&0xF) + 1
		sub >>= 4
		c.h = int(sub&0xF) + 1
		sub >>= 4
		if c.v > brMaxSampling || c.h > brMaxSampling {
			return st.failf("jpgrec: invalid brunsli header")
		}
	}
	if !updateSubsampling(jpg) {
		return st.failf("jpgrec: invalid brunsli header")
	}
	prepareMeta(jpg, st)
	return brStageSection
}

func generateApp0(status byte) ([]byte, error) {
	app := []byte{0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0x01,
		0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00}
	if status&1 != 0 {
		app[9] = 2
	} else {
		app[9] = 1
	}
	status >>= 1
	app[10] = status & 3
	status >>= 2
	if int(status) >= len(kApp0Densities) {
		return nil, errBr
	}
	dens := kApp0Densities[status]
	app[11] = byte(dens >> 8)
	app[13] = app[11]
	app[12] = byte(dens)
	app[14] = app[12]
	return app, nil
}

func generateApp(marker, code byte) ([]byte, error) {
	var s []byte
	switch marker {
	case 0x80:
		s = append([]byte(nil), kAppDataE2[:]...)
		s[84] = code
	case 0x81:
		s = append([]byte(nil), kAppDataEC[:]...)
		s[15] = code
	case 0x82:
		s = append([]byte(nil), kAppDataEE[:]...)
		s[10] = code
	default:
		return nil, errBr
	}
	return s, nil
}

func addMeta(meta []byte, jpg *jpegPic) error {
	pos := 0
	shorts := 0
	for pos < len(meta) {
		marker := meta[pos]
		pos++
		if marker == 0xD9 {
			jpg.tail = append([]byte(nil), meta[pos:]...)
			break
		} else if marker < 0x40 {
			shorts++
			if shorts > brShortLimit {
				return errBr
			}
			app, err := generateApp0(marker)
			if err != nil {
				return err
			}
			jpg.app = append(jpg.app, app)
		} else if marker >= 0x80 && marker <= 0x82 {
			shorts++
			if shorts > brShortLimit || pos >= len(meta) {
				return errBr
			}
			code := meta[pos]
			pos++
			app, err := generateApp(marker, code)
			if err != nil {
				return err
			}
			jpg.app = append(jpg.app, app)
		} else {
			if pos+1 >= len(meta) {
				return errBr
			}
			markerLen := int(meta[pos])<<8 | int(meta[pos+1])
			if marker == 0xFE {
				if pos+markerLen > len(meta) {
					return errBr
				}
				jpg.com = append(jpg.com, append([]byte(nil), meta[pos:pos+markerLen]...))
			} else if marker>>4 == 0x0E {
				if pos-1+markerLen+1 > len(meta) {
					return errBr
				}
				jpg.app = append(jpg.app, append([]byte(nil), meta[pos-1:pos-1+markerLen+1]...))
			} else {
				return errBr
			}
			pos += markerLen
		}
	}
	return nil
}

func decodeMeta(st *brState, jpg *jpegPic) error {
	if st.pos == st.sectionEnd {
		return nil
	}
	if st.pos+1 == st.sectionEnd {
		b, ok := st.u8()
		if !ok {
			return errBr
		}
		return addMeta([]byte{b}, jpg)
	}
	n, ok := st.base128()
	if !ok {
		return errBr
	}
	if st.sectionEnd < st.pos {
		return errBr
	}
	compLen := st.sectionEnd - st.pos
	if st.useBrotli {
		return fmt.Errorf("jpgrec: brunsli brotli metadata is not implemented")
	}
	if n > uint64(len(st.data)) || compLen > int(n) {
		return errBr
	}
	meta := make([]byte, int(n))
	copy(meta, st.data[st.pos:st.pos+compLen])
	if err := addMeta(meta, jpg); err != nil {
		return err
	}
	st.pos += compLen
	return nil
}

type permCoder struct{ v []byte }

func (p *permCoder) nbits() int {
	n := len(p.v)
	if n <= 1 {
		return 0
	}
	return log2FloorNonZero(uint32(n-1)) + 1
}

func (p *permCoder) remove(code int) int {
	if code < 0 || code >= len(p.v) {
		return -1
	}
	val := int(p.v[code])
	p.v = append(p.v[:code], p.v[code+1:]...)
	return val
}

func decodeHuffCode(br *brBitReader, h *jpegHuff, knownLast bool) bool {
	if !br.readMore() {
		return false
	}
	h.slot = int(br.readBits(2))
	isDC := br.readBits(1) == 0
	if !isDC {
		h.slot += 0x10
	}
	h.isLast = knownLast || br.readBits(1) != 0
	h.counts[0] = 0
	if br.readBits(1) != 0 {
		idx := int(br.readBits(1))
		if isDC {
			for i := 0; i < 16; i++ {
				h.counts[i+1] = int(kStockDCCounts[idx*16+i])
			}
			for i := 0; i < 13; i++ {
				h.values[i] = int(kStockDCValues[idx*13+i])
			}
		} else {
			for i := 0; i < 16; i++ {
				h.counts[i+1] = int(kStockACCounts[idx*16+i])
			}
			for i := 0; i < 163; i++ {
				h.values[i] = int(kStockACValues[idx*163+i])
			}
		}
		return true
	}
	total := 0
	space := int32(1) << 16
	maxLen := int(br.readBits(4)) + 1
	maxCount := 256
	if isDC {
		maxCount = 12
	}
	space -= int32(1) << uint(16-maxLen)
	for i := 1; i <= maxLen; i++ {
		shift := uint(16 - i)
		limit := maxCount - total
		room := int(space >> shift)
		if room < limit {
			limit = room
		}
		if limit > 0 {
			nbits := log2FloorNonZero(uint32(limit)) + 1
			count := int(br.readBits(uint32(nbits)))
			if count > limit {
				return false
			}
			h.counts[i] = count
			total += count
			space -= int32(count) << shift
		}
	}
	h.counts[maxLen]++
	var src []byte
	if isDC {
		src = kDefaultDCValues[:]
	} else {
		src = kDefaultACValues[:]
	}
	p := permCoder{v: append([]byte(nil), src...)}
	for i := 0; i < total; i++ {
		if !br.readMore() {
			return false
		}
		nb := p.nbits()
		code := int(decodeLimitedVarint(br, 2, (nb+1)>>1))
		val := p.remove(code)
		if val < 0 {
			return false
		}
		h.values[i] = val
	}
	h.values[total] = 256
	return true
}

func decodeScanInfo(br *brBitReader, si *jpegScan) bool {
	if !br.readMore() {
		return false
	}
	si.ss = int(br.readBits(6))
	si.se = int(br.readBits(6))
	si.ah = int(br.readBits(4))
	si.al = int(br.readBits(4))
	n := int(br.readBits(2)) + 1
	si.comps = make([]jpegCompScan, n)
	si.reset = map[int]struct{}{}
	for i := 0; i < n; i++ {
		si.comps[i] = jpegCompScan{
			comp: int(br.readBits(2)),
			dc:   int(br.readBits(2)),
			ac:   int(br.readBits(2)),
		}
	}
	last := int32(-1)
	for br.readBits(1) != 0 {
		if !br.readMore() {
			return false
		}
		block := last + int32(decodeVarint(br, 28)) + 1
		si.reset[int(block)] = struct{}{}
		last = block
		if last > 1<<30 {
			return false
		}
	}
	last = 0
	lastNum := 0
	for br.readBits(1) != 0 {
		if !br.readMore() {
			return false
		}
		block := last + int32(decodeVarint(br, 28))
		if block > last {
			if lastNum > 0 {
				si.extra = append(si.extra, jpegExtraZero{int(last), lastNum})
				lastNum = 0
			}
		}
		lastNum++
		last = block
		if last > 1<<30 {
			return false
		}
	}
	if lastNum > 0 {
		si.extra = append(si.extra, jpegExtraZero{int(last), lastNum})
	}
	return true
}

func paddingLimit(jpg *jpegPic) uint64 {
	nb := (uint64(jpg.w) + 15) >> 3
	nb *= (uint64(jpg.h) + 15) >> 3
	return 7*nb*uint64(len(jpg.comps)) + 256
}

func decodeAux(br *brBitReader, jpg *jpegPic) bool {
	haveDRI := false
	numScans := 0
	dhtCount := 0
	for {
		if !br.readMore() {
			return false
		}
		marker := byte(0xC0 + br.readBits(6))
		jpg.order = append(jpg.order, marker)
		if marker == 0xC4 {
			dhtCount++
		}
		if marker == 0xDD {
			haveDRI = true
		}
		if marker == 0xDA {
			numScans++
		}
		if marker == 0xD9 {
			break
		}
	}
	if haveDRI {
		jpg.restart = int(br.readBits(16))
	}
	terminal := 0
	for i := 0; ; i++ {
		knownLast := br.readBits(1) != 0
		var h jpegHuff
		if !decodeHuffCode(br, &h, knownLast) {
			return false
		}
		if h.isLast {
			terminal++
		}
		jpg.huff = append(jpg.huff, h)
		if knownLast {
			break
		}
		if i == 512 {
			return false
		}
	}
	if dhtCount != terminal {
		return false
	}
	jpg.scans = make([]jpegScan, numScans)
	for i := 0; i < numScans; i++ {
		if !decodeScanInfo(br, &jpg.scans[i]) {
			return false
		}
	}
	nq := int(br.readBits(2)) + 1
	jpg.quant = make([]jpegQuant, nq)
	for i := 0; i < nq; i++ {
		q := &jpg.quant[i]
		q.index = int(br.readBits(2))
		q.isLast = i == nq-1 || br.readBits(1) != 0
		q.precision = int(br.readBits(4))
		if q.precision > 1 {
			return false
		}
	}
	compIDs := int(br.readBits(2))
	minC := []int{3, 1, 3, 0}[compIDs]
	if len(jpg.comps) < minC {
		return false
	}
	switch compIDs {
	case 0:
		jpg.comps[0].id, jpg.comps[1].id, jpg.comps[2].id = 1, 2, 3
	case 1:
		jpg.comps[0].id = 1
	case 2:
		jpg.comps[0].id, jpg.comps[1].id, jpg.comps[2].id = 'R', 'G', 'B'
	default:
		for i := range jpg.comps {
			jpg.comps[i].id = int(br.readBits(8))
		}
	}
	nSize := decodeLimitedVarint(br, 8, 4)
	jpg.hasPad = nSize > 0
	if nSize > 0 {
		if nSize > paddingLimit(jpg) {
			return false
		}
		jpg.pad = make([]int, int(nSize))
		for i := range jpg.pad {
			if !br.readMore() {
				return false
			}
			jpg.pad[i] = int(br.readBits(1))
		}
	}
	return true
}

func fillQuantMatrix(chroma bool, q uint32, dst []byte) {
	in := kDefaultQuantMatrix[:64]
	if chroma {
		in = kDefaultQuantMatrix[64:]
	}
	for i := 0; i < 64; i++ {
		v := (uint32(in[i])*q + 32) >> 6
		if v < 1 {
			v = 1
		} else if v > 255 {
			v = 255
		}
		dst[i] = byte(v)
	}
}

func decodeQuantTables(br *brBitReader, jpg *jpegPic) bool {
	if !br.readMore() {
		return false
	}
	have := len(jpg.quant) != 0
	nq := int(br.readBits(2)) + 1
	if len(jpg.quant) != nq {
		return false
	}
	for i := 0; i < nq; i++ {
		q := &jpg.quant[i]
		prec := 0
		if br.readBits(1) == 0 {
			code := int(br.readBits(3))
			base := 0
			if i > 0 {
				base = 1
			}
			off := (base*8 + code) * 64
			for k := 0; k < 64; k++ {
				q.values[k] = int32(kStockQuant[off+k])
			}
		} else {
			qf := br.readBits(6)
			var pred [64]byte
			fillQuantMatrix(i > 0, qf, pred[:])
			delta := 0
			for k := 0; k < 64; k++ {
				if !br.readMore() {
					return false
				}
				if br.readBits(1) != 0 {
					sign := br.readBits(1)
					diff := int(decodeVarint(br, 16)) + 1
					if sign != 0 {
						delta -= diff
					} else {
						delta += diff
					}
				}
				j := int(kJPEGNaturalOrder[k])
				qv := int(pred[j]) + delta
				q.values[j] = int32(qv)
				if qv <= 0 || qv >= 65536 {
					return false
				}
				if qv >= 256 {
					prec = 1
				}
			}
		}
		if !have {
			q.precision = prec
			q.isLast = true
			q.index = i
		} else if q.precision != prec {
			return false
		}
	}
	for i := range jpg.comps {
		jpg.comps[i].quant = int(br.readBits(2))
		if jpg.comps[i].quant >= len(jpg.quant) {
			return false
		}
	}
	return true
}

func decodeInternals(st *brState, jpg *jpegPic) error {
	if st.pos == st.sectionEnd {
		return errBr
	}
	sec := st.data[st.pos:st.sectionEnd]
	br := newBrBitReader(sec)
	if !decodeAux(br, jpg) {
		return errBr
	}
	tail := br.jump()
	if tail < 0 || tail > len(sec) {
		return errBr
	}
	st.pos += len(sec) - tail
	for _, m := range jpg.order {
		if m != 0xFF {
			continue
		}
		n, ok := st.dataLen()
		if !ok {
			return errBr
		}
		st.pos += n
		blob := st.data[st.pos-n : st.pos]
		jpg.inter = append(jpg.inter, append([]byte(nil), blob...))
	}
	return nil
}

func decodeQuantSec(st *brState, jpg *jpegPic) error {
	if st.pos == st.sectionEnd {
		return errBr
	}
	sec := st.data[st.pos:st.sectionEnd]
	br := newBrBitReader(sec)
	if !decodeQuantTables(br, jpg) {
		return errBr
	}
	if br.jump() != 0 {
		return errBr
	}
	st.pos += len(sec)
	return nil
}

func decodeHistSec(st *brState, jpg *jpegPic) error {
	if st.pos == st.sectionEnd || len(jpg.comps) == 0 {
		return errBr
	}
	sec := st.data[st.pos:st.sectionEnd]
	br := newBrBitReader(sec)
	numCtx := len(jpg.comps)
	for i := range jpg.comps {
		scheme := int(br.readBits(3))
		if scheme >= 7 {
			return errBr
		}
		st.meta[i].ctxBits = scheme
		st.meta[i].ctxOff = numCtx
		numCtx += int(kNumNonzeroContextSkip[scheme])
	}
	st.numCtx = numCtx
	st.numHist = int(decodeVarLenUint8(br)) + 1
	st.ctxMap = make([]byte, st.numCtx*kNumAvg)
	if !decodeContextMap(st.numHist, len(st.ctxMap), st.ctxMap, br) {
		return errBr
	}
	st.codes = make([]ansCode, st.numHist)
	for i := 0; i < st.numHist; i++ {
		c, ok := ansFromBits(brCoeffAlpha, br)
		if !ok {
			return errBr
		}
		st.codes[i] = c
	}
	if br.jump() != 0 {
		return errBr
	}
	st.pos += len(sec)
	return nil
}

func adaptiveMedian(w, n, nw int32) int32 {
	mx, mn := w, n
	if n > w {
		mx, mn = n, w
	}
	if nw > mx {
		return mn
	}
	if nw < mn {
		return mx
	}
	return n + w - nw
}

func predictDC(coeffs []int16, idx, x, y, stride int) int32 {
	if y != 0 {
		if x != 0 {
			return adaptiveMedian(int32(coeffs[idx-64]), int32(coeffs[idx-stride]), int32(coeffs[idx-stride-64]))
		}
		return int32(coeffs[idx-stride])
	}
	if x != 0 {
		return int32(coeffs[idx-64])
	}
	return 0
}

func wavgDC(vals []int32, x int) int {
	// vals is prev_abs_coeff. C passes &vals[2], so x-2..x+1 land on vals[x..x+3].
	sum := int32(1) + vals[x] + vals[x+1] + vals[x+2] + vals[x+3]
	if sum>>8 != 0 {
		return 8
	}
	return log2FloorNonZero(uint32(sum))
}

func wavgAC(vals []int32, idx, delta int) int {
	sum := int32(4) + vals[idx] + (vals[idx-64]+vals[idx+delta])*2 + vals[idx-128] + vals[idx+delta-64] + vals[idx+delta+64]
	if sum>>10 != 0 {
		return 8
	}
	return log2FloorNonZero(uint32(sum)) - 2
}

func acPredictContext(p int64) int {
	const kMaxPred = (1 << 9) - 1
	if p >= 0 {
		u := uint64(p) << 1
		if u > kMaxPred {
			return 8
		}
		u++
		return log2FloorNonZero(uint32(u))
	}
	u := uint64(-p) << 1
	if u > kMaxPred {
		return -8
	}
	u++
	return -log2FloorNonZero(uint32(u))
}

func acPredictCol(coeffs []int16, prevAt, curAt int, mult []int32) int {
	var terms [8]int16
	terms[1] = int16(int32(coeffs[curAt+1]) + int32(coeffs[prevAt+1]))
	terms[2] = int16(int32(coeffs[curAt+2]) - int32(coeffs[prevAt+2]))
	terms[3] = int16(int32(coeffs[curAt+3]) + int32(coeffs[prevAt+3]))
	terms[4] = int16(int32(coeffs[curAt+4]) - int32(coeffs[prevAt+4]))
	terms[5] = int16(int32(coeffs[curAt+5]) + int32(coeffs[prevAt+5]))
	terms[6] = int16(int32(coeffs[curAt+6]) - int32(coeffs[prevAt+6]))
	terms[7] = int16(int32(coeffs[curAt+7]) + int32(coeffs[prevAt+7]))
	var delta int64
	for i := 0; i < 8; i++ {
		delta += int64(terms[i]) * int64(mult[i])
	}
	return acPredictContext(int64(coeffs[prevAt]) - delta/kACPredPrec)
}

func acPredictRow(coeffs []int16, prevAt, curAt int, mult []int32) int {
	var terms [8]int16
	terms[1] = int16(int32(coeffs[curAt+8]) + int32(coeffs[prevAt+8]))
	terms[2] = int16(int32(coeffs[curAt+16]) - int32(coeffs[prevAt+16]))
	terms[3] = int16(int32(coeffs[curAt+24]) + int32(coeffs[prevAt+24]))
	terms[4] = int16(int32(coeffs[curAt+32]) - int32(coeffs[prevAt+32]))
	terms[5] = int16(int32(coeffs[curAt+40]) + int32(coeffs[prevAt+40]))
	terms[6] = int16(int32(coeffs[curAt+48]) - int32(coeffs[prevAt+48]))
	terms[7] = int16(int32(coeffs[curAt+56]) + int32(coeffs[prevAt+56]))
	var delta int64
	for i := 0; i < 8; i++ {
		delta += int64(terms[i]) * int64(mult[i])
	}
	return acPredictContext(int64(coeffs[prevAt]) - delta/kACPredPrec)
}

type dcComp struct {
	isZero prob
	empty  [3]prob
	sign   [9]prob
	extra  [10]prob
	non0   []int32
	abs    []int32
	sgn    []int32
}

func newDCComp(w int) *dcComp {
	c := &dcComp{
		non0: make([]int32, w+1),
		abs:  make([]int32, w+3),
		sgn:  make([]int32, w+1),
	}
	for i := range c.non0 {
		c.non0[i] = 1
	}
	c.isZero.init(135)
	for i := range c.sign {
		c.sign[i].init(128)
	}
	for i := range c.empty {
		c.empty[i].init(74)
	}
	for i := range c.extra {
		c.extra[i].init(150)
	}
	return c
}

func decodeNumNZ(p []prob, ac *arithDec, in *brInput) int {
	val := 1
	for b := 0; b < 6; b++ {
		bit := ac.readBit(int(p[val-1].get()), in)
		p[val-1].add(bit)
		val = 2*val + bit
	}
	return val - 64
}

func decodeDC(st *brState, in *brInput) bool {
	meta := st.meta
	if len(meta) == 0 || meta[0].v == 0 {
		return false
	}
	mcuRows := meta[0].height / meta[0].v
	comps := make([]*dcComp, len(meta))
	for c := range meta {
		comps[c] = newDCComp(meta[c].width)
	}
	var ac arithDec
	var ans ansDec
	ans.init(in)
	in.initBits()
	ac.init(in)
	for mcuY := 0; mcuY < mcuRows; mcuY++ {
		for i := range meta {
			c := comps[i]
			m := &meta[i]
			ctx := st.ctxMap[i*kNumAvg:]
			y := mcuY * m.v
			for iy := 0; iy < m.v; iy, y = iy+1, y+1 {
				for x := 0; x < m.width; x++ {
					emptyCtx := int(c.non0[x] + c.non0[x+1])
					if emptyCtx < 0 || emptyCtx >= 3 {
						return false
					}
					isEmpty := ac.readBit(int(c.empty[emptyCtx].get()), in) == 0
					c.empty[emptyCtx].add(b2i(!isEmpty))
					c.non0[x+1] = boolTo32(!isEmpty)
					bi := y*m.bStride + x
					m.block[bi] = boolByte(isEmpty)
					absVal := 0
					sign := 0
					if !isEmpty {
						isZero := ac.readBit(int(c.isZero.get()), in)
						c.isZero.add(isZero)
						if isZero == 0 {
							avg := wavgDC(c.abs, x)
							signCtx := int(c.sgn[1+x]*3 + c.sgn[x])
							sign = ac.readBit(int(c.sign[signCtx].get()), in)
							c.sign[signCtx].add(sign)
							if avg < 0 || avg >= kNumAvg {
								return false
							}
							ent := int(ctx[avg])
							if ent >= len(st.codes) {
								return false
							}
							code := ans.read(&st.codes[ent], in)
							if code < 8 {
								absVal = code + 1
							} else {
								nbits := code - 8
								if nbits < 0 || nbits >= 10 {
									return false
								}
								feb := ac.readBit(int(c.extra[nbits].get()), in)
								c.extra[nbits].add(feb)
								extra := feb << uint(nbits)
								if nbits > 0 {
									extra |= in.readBits(nbits)
								}
								absVal = 8 - 1 + (2 << uint(nbits)) + extra
							}
						}
					}
					c.abs[2+x] = int32(absVal)
					if absVal != 0 {
						c.sgn[1+x] = int32(sign + 1)
					} else {
						c.sgn[1+x] = 0
					}
					idx := y*m.acStride + x*64
					pred := predictDC(m.coeffs, idx, x, y, m.acStride)
					sum := int32(1-2*sign)*int32(absVal) + pred
					m.coeffs[idx] = int16(sum)
				}
			}
		}
	}
	return ans.check() && in.err == 0
}

func b2i(v bool) int {
	if v {
		return 1
	}
	return 0
}

func boolTo32(v bool) int32 {
	if v {
		return 1
	}
	return 0
}

func boolByte(v bool) byte {
	if v {
		return 1
	}
	return 0
}

func computeMult(quant *[64]int32, row, col []int32) bool {
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			qy := quant[y*8]
			qx := quant[x]
			if qy == 0 || qx == 0 {
				return false
			}
			row[x+8*y] = (quant[x+8*y] * kSqrt2FixedPoint) / qy
			col[x*8+y] = (quant[x+8*y] * kSqrt2FixedPoint) / qx
		}
	}
	return true
}

type acComp struct {
	order   [64]int
	multRow [64]int32
	multCol [64]int32
	isZero  []prob
	sign    []prob
	extra   []prob
	nnzProb [32][63]prob
	non0    []int32
	abs     []int32
	sgn     []int32
	width   int
}

func newACComp(w int) *acComp {
	c := &acComp{
		isZero: make([]prob, 11*64),
		sign:   make([]prob, 17*64),
		extra:  make([]prob, 10*64),
		non0:   make([]int32, w+1),
		abs:    make([]int32, 64*2*(w+3)),
		sgn:    make([]int32, 64*(w+1)),
		width:  w,
	}
	for i := 0; i < 11; i++ {
		for k := 0; k < 64; k++ {
			v := int(kInitProbAC[k]) + 9*(i-7)
			c.isZero[i*64+k].init(v)
		}
	}
	for i := range c.sign {
		if i < 8*64 {
			c.sign[i].init(108)
		} else if i < 9*64 {
			c.sign[i].init(128)
		} else {
			c.sign[i].init(148)
		}
	}
	for i := range c.extra {
		c.extra[i].init(158)
	}
	for i := 0; i < 32; i++ {
		for j := 0; j < 63; j++ {
			c.nnzProb[i][j].init(int(kInitProbNonzero[i*63+j]))
		}
	}
	return c
}

func decodeLehmer(code, sigma []int) bool {
	items := make([]int, len(code))
	for i := range items {
		items[i] = i
	}
	for i, index := range code {
		if index < 0 || index >= len(items) {
			return false
		}
		sigma[i] = items[index]
		items = append(items[:index], items[index+1:]...)
	}
	return true
}

func decodeCoeffOrder(order []int, in *brInput) bool {
	var lehmer [64]int
	for i := 0; i < 64; i += 16 {
		if in.readBits(1) == 0 {
			continue
		}
		start := i
		if i == 0 {
			start = 1
		}
		for j := start; j < i+16; j++ {
			v := 0
			for v <= 64 {
				bits := in.readBits(3)
				v += bits
				if bits < 7 {
					break
				}
			}
			if v > 64 {
				return false
			}
			lehmer[j] = v
		}
	}
	end := 63
	for end >= 1 && lehmer[end] == 0 {
		end--
	}
	if lehmer[end] == 1 {
		return false
	}
	for i := 1; i <= end; i++ {
		if lehmer[i] == 0 {
			return false
		}
		lehmer[i]--
	}
	var sigma [64]int
	if !decodeLehmer(lehmer[:], sigma[:]) {
		return false
	}
	for k := 0; k < 64; k++ {
		order[k] = int(kJPEGNaturalOrder[sigma[k]])
	}
	return true
}

func nnzContext(prev []int32, x, y int) int {
	if y == 0 {
		return int(prev[x] >> 1)
	}
	if x == 0 {
		return int(prev[1] >> 1)
	}
	return int((prev[x] + prev[x+1] + 1) >> 2)
}

func decodeAC(st *brState, in *brInput) bool {
	meta := st.meta
	if len(meta) == 0 || meta[0].v == 0 {
		return false
	}
	mcuRows := meta[0].height / meta[0].v
	comps := make([]*acComp, len(meta))
	for c := range meta {
		comps[c] = newACComp(meta[c].width)
		if !computeMult(&meta[c].quant, comps[c].multRow[:], comps[c].multCol[:]) {
			return false
		}
	}
	var ac arithDec
	var ans ansDec
	ans.init(in)
	in.initBits()
	ac.init(in)
	for i := range comps {
		if !decodeCoeffOrder(comps[i].order[:], in) {
			return false
		}
	}
	for mcuY := 0; mcuY < mcuRows; mcuY++ {
		for i := range meta {
			c := comps[i]
			m := &meta[i]
			ctxBase := m.ctxOff * kNumAvg
			y := mcuY * m.v
			rowDelta := (1 - 2*(y&1)) * (m.width + 3) * 64
			for iy := 0; iy < m.v; iy, y = iy+1, y+1 {
				absBase := ((y&1)*(m.width+3) + 2) * 64
				sgnBase := 64
				for x := 0; x < m.width; x++ {
					bi := y*m.bStride + x
					empty := m.block[bi] != 0
					lastNZ := 0
					if !empty {
						nzCtx := nnzContext(c.non0, x, y)
						if nzCtx < 0 || nzCtx >= 32 {
							return false
						}
						lastNZ = decodeNumNZ(c.nnzProb[nzCtx][:], &ac, in)
					}
					for k := 63; k > lastNZ; k-- {
						c.sgn[sgnBase+k] = 0
						c.abs[absBase+k] = 0
					}
					numNZ := 0
					block := y*m.acStride + x*64
					for k := lastNZ; k >= 1; k-- {
						isZero := 0
						if k < lastNZ {
							bucket := int(kNonzeroBuckets[numNZ-1])
							zctx := bucket*64 + k
							isZero = ac.readBit(int(c.isZero[zctx].get()), in)
							c.isZero[zctx].add(isZero)
						}
						absVal := 0
						sign := 1
						kNat := c.order[k]
						if isZero == 0 {
							avg := 0
							signCtx := 8
							if kNat < 8 {
								if y > 0 {
									pctx := acPredictRow(m.coeffs, block-m.acStride+kNat, block+kNat, c.multCol[kNat*8:])
									if pctx < 0 {
										avg = -pctx
									} else {
										avg = pctx
									}
									signCtx += pctx
								}
							} else if kNat&7 == 0 {
								if x > 0 {
									pctx := acPredictCol(m.coeffs, block-64+kNat, block+kNat, c.multRow[kNat:])
									if pctx < 0 {
										avg = -pctx
									} else {
										avg = pctx
									}
									signCtx += pctx
								}
							} else {
								avg = wavgAC(c.abs, absBase+k, rowDelta)
								signCtx = int(c.sgn[sgnBase+k]*3 + c.sgn[sgnBase+k-64])
							}
							signCtx = signCtx*64 + k
							if signCtx < 0 || signCtx >= len(c.sign) {
								return false
							}
							sb := ac.readBit(int(c.sign[signCtx].get()), in)
							c.sign[signCtx].add(sb)
							c.sgn[sgnBase+k] = int32(sb + 1)
							sign = 1 - 2*sb
							zd := zeroDens(numNZ, k, m.ctxBits)
							histo := zd*kNumAvg + avg
							if ctxBase+histo < 0 || ctxBase+histo >= len(st.ctxMap) {
								return false
							}
							ent := int(st.ctxMap[ctxBase+histo])
							if ent < 0 || ent >= len(st.codes) {
								return false
							}
							code := ans.read(&st.codes[ent], in)
							if code < 8 {
								absVal = code + 1
							} else {
								nbits := code - 8
								p := &c.extra[k*10+nbits]
								feb := ac.readBit(int(p.get()), in)
								p.add(feb)
								extra := feb << uint(nbits)
								if nbits > 0 {
									extra |= in.readBits(nbits)
								}
								absVal = 7 + (2 << uint(nbits)) + extra
							}
							numNZ++
						} else {
							c.sgn[sgnBase+k] = 0
						}
						m.coeffs[block+kNat] = int16(int32(sign) * int32(absVal))
						c.abs[absBase+k] = int32(absVal)
					}
					c.non0[x+1] = int32(numNZ)
					sgnBase += 64
					absBase += 64
				}
				rowDelta = -rowDelta
			}
		}
	}
	return ans.check() && in.err == 0
}

func zeroDens(nz, k, bits int) int {
	return int(kNumNonzeroContext[bits*64+nz]) + int(kFreqContext[bits*64+k])
}

func decodeDCSec(st *brState) error {
	sec := st.data[st.pos:st.sectionEnd]
	in := newBrInput(sec)
	if !decodeDC(st, in) || in.nwords != in.pos {
		return errBr
	}
	st.pos += len(sec)
	return nil
}

func decodeACSec(st *brState) error {
	sec := st.data[st.pos:st.sectionEnd]
	in := newBrInput(sec)
	if !decodeAC(st, in) || in.nwords != in.pos {
		return errBr
	}
	st.pos += len(sec)
	return nil
}

func decodeOriginal(st *brState, jpg *jpegPic) int {
	b, ok := st.u8()
	if !ok || b != byte((9<<3)|2) {
		return st.failf("jpgrec: invalid brunsli original jpeg")
	}
	n, ok := st.dataLen()
	if !ok {
		return st.failf("jpgrec: invalid brunsli original jpeg")
	}
	jpg.orig = st.data[st.pos : st.pos+n]
	st.pos += n
	return brStageDone
}

func parseSection(st *brState) int {
	if st.pos >= len(st.data) {
		if st.tags&(1<<8) == 0 {
			return st.failf("jpgrec: truncated brunsli stream")
		}
		return brStageDone
	}
	m, ok := st.u8()
	if !ok {
		return st.failf("jpgrec: invalid brunsli stream")
	}
	tag := int(m >> 3)
	if tag == 0 || tag > 15 {
		return st.failf("jpgrec: invalid brunsli stream")
	}
	bit := int32(1) << uint(tag)
	wire := m & 7
	if wire != 0 && wire != 2 {
		return st.failf("jpgrec: invalid brunsli stream")
	}
	if st.tags&bit != 0 {
		return st.failf("jpgrec: duplicate brunsli section")
	}
	st.tags |= bit
	known := bit == 1<<1 || bit == 1<<2 || bit == 1<<3 || bit == 1<<4 || bit == 1<<5 || bit == 1<<6 || bit == 1<<7 || bit == 1<<8 || bit == 1<<9
	if wire != 2 {
		if known {
			return st.failf("jpgrec: invalid brunsli stream")
		}
		if _, ok := st.base128(); !ok {
			return st.failf("jpgrec: invalid brunsli stream")
		}
		return brStageSection
	}
	n, ok := st.dataLen()
	if !ok {
		return st.failf("jpgrec: invalid brunsli stream")
	}
	st.tag = tag
	st.sectionEnd = st.pos + n
	return brStageBody
}

func processSection(st *brState, jpg *jpegPic) int {
	bit := int32(1) << uint(st.tag)
	known := bit == 1<<3 || bit == 1<<4 || bit == 1<<5 || bit == 1<<6 || bit == 1<<7 || bit == 1<<8
	// Signature/header/original are not length-delimited body tags here.
	// Unknown section tags are skipped. Known set also includes those three,
	// which are not expected in this switch.
	fullKnown := known || bit == 1<<1 || bit == 1<<2 || bit == 1<<9
	if !fullKnown {
		st.pos = st.sectionEnd
		return brStageSection
	}
	var err error
	switch st.tag {
	case 3:
		err = decodeMeta(st, jpg)
	case 4:
		err = decodeInternals(st, jpg)
	case 5:
		if st.tags&(1<<4) == 0 {
			return st.failf("jpgrec: brunsli quant section without internals")
		}
		err = decodeQuantSec(st, jpg)
	case 6:
		if st.tags&(1<<4) == 0 {
			return st.failf("jpgrec: brunsli histogram section without internals")
		}
		err = decodeHistSec(st, jpg)
	case 7:
		if st.tags&(1<<6) == 0 || st.tags&(1<<5) == 0 {
			return st.failf("jpgrec: brunsli DC section out of order")
		}
		warmupMeta(jpg, st)
		err = decodeDCSec(st)
	case 8:
		if st.tags&(1<<7) == 0 {
			return st.failf("jpgrec: brunsli AC section out of order")
		}
		warmupMeta(jpg, st)
		err = decodeACSec(st)
	default:
		return st.failf("jpgrec: invalid brunsli section")
	}
	if err != nil {
		st.fail = err
		return brStageError
	}
	if st.pos != st.sectionEnd {
		return st.failf("jpgrec: invalid brunsli stream")
	}
	return brStageSection
}

func processJPEG(st *brState, jpg *jpegPic) error {
	for {
		switch st.stage {
		case brStageSignature:
			st.stage = verifySignature(st)
		case brStageHeader:
			st.stage = decodeHeader(st, jpg)
		case brStageFallback:
			st.stage = decodeOriginal(st, jpg)
		case brStageSection:
			st.stage = parseSection(st)
		case brStageBody:
			st.stage = processSection(st, jpg)
		case brStageDone:
			if st.pos != len(st.data) {
				return errBr
			}
			return nil
		case brStageError:
			if st.fail != nil {
				return st.fail
			}
			return errBr
		default:
			return errBr
		}
	}
}

func restoreBrunsli(payload []byte, brotli bool) ([]byte, error) {
	jpg := &jpegPic{maxH: 1, maxV: 1}
	st := &brState{data: payload, useBrotli: brotli}
	if err := processJPEG(st, jpg); err != nil {
		return nil, err
	}
	out, err := writeJPEG(jpg)
	if err != nil {
		return nil, err
	}
	if len(out) > jpgMaxMemory {
		return nil, fmt.Errorf("jpgrec: output length %d exceeds JPG_MAX_MEMORY_SIZE", len(out))
	}
	return out, nil
}
