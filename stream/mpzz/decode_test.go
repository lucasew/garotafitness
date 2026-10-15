package mpzz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"math/bits"
	"testing"

	"github.com/stretchr/testify/require"
)

// rngEnc is the test-only dual of rangeDecoder. discover() picks the
// shift/renorm pair that round-trips rangeDecoder.bit.
type rngEnc struct {
	low       uint64
	rg        uint32
	cache     uint32
	cacheSize uint32
	out       []byte
	st        encStyle
}

type encStyle struct {
	name        string
	kind        int // 0: lzma sdk ShiftLow, 1: classic delayed cache
	renormFirst bool
	flush       int
	prime       int
	drop        int
}

func newRng(st encStyle) *rngEnc {
	e := &rngEnc{rg: 0xffffffff, st: st}
	if st.kind == 1 {
		e.cacheSize = 1
	}
	for i := 0; i < st.prime; i++ {
		e.shift()
	}
	return e
}

func (e *rngEnc) shift() {
	if e.st.kind == 0 {
		low := uint32(e.low)
		high := uint32(e.low >> 32)
		e.low = uint64(low << 8)
		if low < 0xFF000000 || high != 0 {
			e.out = append(e.out, byte(e.cache+high))
			e.cache = low >> 24
			if e.cacheSize == 0 {
				return
			}
			high += 0xFF
			for {
				e.out = append(e.out, byte(high))
				e.cacheSize--
				if e.cacheSize == 0 {
					return
				}
			}
		}
		e.cacheSize++
		return
	}
	if uint32(e.low) < 0xFF000000 || e.low>>32 != 0 {
		b := e.cache
		for {
			e.out = append(e.out, byte(b+uint32(e.low>>32)))
			e.cacheSize--
			if e.cacheSize == 0 {
				break
			}
			b = 0xFF
		}
		e.cache = uint32(e.low>>24) & 0xFF
	}
	e.cacheSize++
	e.low = uint64(uint32(e.low) << 8)
}

func (e *rngEnc) bit(p *uint16, bit uint32) {
	renorm := func() {
		for e.rg < 1<<24 {
			e.rg <<= 8
			e.shift()
		}
	}
	if e.st.renormFirst {
		renorm()
	}
	f := uint32(*p)
	bound := (e.rg >> 16) * f
	if bit == 1 {
		e.low += uint64(bound)
		e.rg -= bound
		*p = uint16(f - (f >> 6))
	} else {
		e.rg = bound
		*p = uint16(f + ((65535 - f) >> 6))
	}
	if !e.st.renormFirst {
		renorm()
	}
}

func (e *rngEnc) finish() []byte {
	for i := 0; i < e.st.flush; i++ {
		e.shift()
	}
	if e.st.drop > len(e.out) {
		return nil
	}
	return append([]byte(nil), e.out[e.st.drop:]...)
}

func lockedStyle() encStyle {
	// LZMA ShiftLow, renorm after the bit, five flush shifts, drop the
	// leading byte a 5-byte range init would have shifted out.
	return encStyle{name: "k0-firstfalse-f5-d1", kind: 0, renormFirst: false, flush: 5, drop: 1}
}

func encodeBits(st encStyle, bits []uint32) []byte {
	var pe uint16 = 0x8000
	e := newRng(st)
	for _, b := range bits {
		e.bit(&pe, b)
	}
	return e.finish()
}

func decodeBits(data []byte, bits []uint32) bool {
	if len(data) == 0 {
		return false
	}
	r := newRange(data)
	var pd uint16 = 0x8000
	for _, b := range bits {
		if r.bit(&pd) != b || r.err != nil {
			return false
		}
	}
	return true
}

func roundBits(st encStyle, bits []uint32) bool {
	return decodeBits(encodeBits(st, bits), bits)
}

func sampleBits() []uint32 {
	bits := make([]uint32, 0, 360)
	for i := 0; i < 240; i++ {
		bits = append(bits, uint32((i*17+3)>>2)&1)
	}
	for i := 0; i < 40; i++ {
		bits = append(bits, 1)
	}
	for i := 0; i < 40; i++ {
		bits = append(bits, 0)
	}
	bits = append(bits, 1, 0, 1, 1, 0, 0, 1, 0)
	return bits
}

func (m *integerModel) putBit(e *rngEnc, offset, b uint32) {
	if offset < 12 || offset&1 != 0 || offset >= 0x2000 {
		panic(fmt.Sprintf("model offset %#x", offset))
	}
	e.bit(&m.prob[(offset-12)/2], b)
}

func (m *integerModel) putInt(e *rngEnc, lengthBits, signBits, lowBits uint, grouped bool, target uint32) {
	if target == 0 {
		m.putBit(e, 12+2*m.nonzero, 0)
		m.nonzero = (2 * m.nonzero) & 3
		m.length = 0
		return
	}
	sign := uint32(0)
	mag := target
	if signBits != 0 && int32(target) < 0 {
		sign = 1
		if int32(target) == -1<<31 {
			mag = 1 << 31
		} else {
			mag = uint32(-int32(target))
		}
	}
	m.putBit(e, 12+2*m.nonzero, 1)
	m.nonzero = (2*m.nonzero + 1) & 3
	if signBits != 0 {
		m.putBit(e, 20+2*m.sign, sign)
		m.sign = (2*m.sign + sign) & ((1 << signBits) - 1)
	}
	n := uint32(bits.Len32(mag) - 1)
	if n >= 1<<lengthBits {
		panic(fmt.Sprintf("magnitude %d does not fit in %d length bits", mag, lengthBits))
	}
	base := uint32(0x34) + m.length<<(lengthBits+1)
	v := uint32(1)
	for i := int(lengthBits) - 1; i >= 0; i-- {
		bit := (n >> uint(i)) & 1
		m.putBit(e, base+2*v, bit)
		v = 2*v + bit
	}
	if v&((1<<lengthBits)-1) != n {
		panic("length tree")
	}
	m.length = n + 1
	base = 0x834
	if grouped {
		base += n << (lowBits + 1)
	}
	v, ctx := uint32(1), uint32(1)
	low := mag - (1 << n)
	for i := int(n) - 1; i >= 0; i-- {
		bit := (low >> uint(i)) & 1
		m.putBit(e, base+2*ctx, bit)
		v = 2*v + bit
		if lowBits > 0 && ctx < 1<<(lowBits-1) {
			ctx = (2*ctx + bit) & ((1 << lowBits) - 1)
		}
	}
	got := (v ^ -sign) + sign
	if signBits == 0 {
		got = v
	}
	if got != target {
		panic(fmt.Sprintf("putInt %d decoded %d", target, got))
	}
}

func putSInt(m *integerModel, e *rngEnc, lengthBits, signBits, lowBits uint, grouped bool, order int, target uint32) {
	raw := target
	switch {
	case order >= 2:
		raw = target - m.value - m.delta
	case order >= 1:
		raw = target - m.value
	}
	m.putInt(e, lengthBits, signBits, lowBits, grouped, raw)
	if got := m.predict(raw, order); got != target {
		panic(fmt.Sprintf("predict got %d want %d", got, target))
	}
}

func putTree(e *rngEnc, probs []uint16, base, n, target uint32) {
	v := uint32(1)
	for i := int(n) - 1; i >= 0; i-- {
		bit := (target >> uint(i)) & 1
		e.bit(&probs[base+v], bit)
		v = 2*v + bit
	}
	if v&((1<<n)-1) != target {
		panic(fmt.Sprintf("tree got %d want %d", v&((1<<n)-1), target))
	}
}

func frameBytes(payload []byte) []byte {
	n := uint32(len(payload))
	var pref []byte
	switch {
	case n < 1<<6:
		pref = []byte{byte(n << 2)}
	case n < 1<<14:
		v := n<<2 | 1
		pref = []byte{byte(v), byte(v >> 8)}
	case n < 1<<22:
		v := n<<2 | 2
		pref = []byte{byte(v), byte(v >> 8), byte(v >> 16)}
	default:
		v := n<<2 | 3
		pref = []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)}
	}
	return append(pref, payload...)
}

func wrapOGGRE(flags byte, parts ...[]byte) []byte {
	out := []byte{'O', 'G', 'G', 'R', 'E', 0, flags}
	for _, p := range parts {
		out = append(out, frameBytes(p)...)
	}
	return out
}

func freshModels() ([]integerModel, []uint16) {
	models := make([]integerModel, 55)
	for i := range models {
		models[i].reset()
	}
	capacity := 1 << 4
	probs := make([]uint16, 0x9225c+capacity*0x45400)
	for i := range probs {
		probs[i] = 0x8000
	}
	return models, probs
}

func putPage(models []integerModel, e *rngEnc, probs []uint16, xor, flags byte, granule uint64, serial, seq uint32, sizes []uint32, laces bool) {
	putSInt(&models[0], e, 3, 0, 2, false, 0, uint32(flags^xor))
	putSInt(&models[1], e, 5, 2, 4, false, 1, uint32(granule))
	putSInt(&models[2], e, 5, 1, 1, false, 1, uint32(granule>>32))
	putSInt(&models[3], e, 5, 2, 4, false, 1, serial)
	putSInt(&models[4], e, 5, 2, 4, false, 2, seq)
	putSInt(&models[5], e, 3, 0, 4, true, 0, uint32(len(sizes)))
	if laces {
		putLaces(models, e, probs, sizes)
	}
}

func putLaces(models []integerModel, e *rngEnc, probs []uint16, sizes []uint32) {
	for i, size := range sizes {
		putSInt(&models[6], e, 4, 0, 6, true, 0, size)
		if size == 0 && i == len(sizes)-1 {
			e.bit(&probs[0x208], 1)
		}
	}
}

func putIdent(models []integerModel, e *rngEnc) {
	putSInt(&models[9], e, 2, 1, 1, false, 1, 1)
	for i := 10; i < 14; i++ {
		putSInt(&models[i], e, 5, 1, 4, false, 1, 0)
	}
	putSInt(&models[14], e, 3, 1, 4, false, 1, 102)
	putSInt(&models[15], e, 3, 1, 4, false, 1, 1)
}

func identPacketLen() int {
	w := &bitWriter{}
	w.write(1, 8)
	for _, b := range []byte("vorbis") {
		w.write(uint32(b), 8)
	}
	w.write(0, 32)
	w.write(1, 8)
	for range 4 {
		w.write(0, 32)
	}
	w.write(102, 8)
	w.write(1, 8)
	return len(w.data)
}

func setupPacketLen() int {
	w := &bitWriter{}
	w.write(5, 8)
	for _, b := range []byte("vorbis") {
		w.write(uint32(b), 8)
	}
	w.write(0, 8)
	w.write(0x564342, 24)
	w.write(1, 16)
	w.write(1, 24)
	w.write(0, 1)
	w.write(0, 1)
	w.write(0, 5)
	w.write(0, 4)
	w.write(0, 6)
	w.write(0, 16)
	w.write(0, 6)
	w.write(1, 16)
	w.write(0, 5)
	w.write(0, 2)
	w.write(0, 4)
	w.write(0, 6)
	w.write(1, 16)
	w.write(0, 24)
	w.write(0, 24)
	w.write(0, 24)
	w.write(0, 6)
	w.write(0, 8)
	w.write(0, 3)
	w.write(0, 1)
	w.write(0, 6)
	w.write(0, 16)
	w.write(0, 1)
	w.write(0, 1)
	w.write(0, 2)
	w.write(0, 8)
	w.write(0, 8)
	w.write(0, 8)
	w.write(0, 6)
	w.write(0, 1)
	w.write(0, 32)
	w.write(0, 8)
	w.write(1, 1)
	return len(w.data)
}

func putSetup(models []integerModel, e *rngEnc, probs []uint16) {
	putSInt(&models[16], e, 3, 1, 4, true, 1, 0)
	putSInt(&models[17], e, 4, 2, 4, false, 1, 1)
	putSInt(&models[18], e, 5, 0, 6, true, 0, 1)
	putTree(e, probs, 0x2218, 1, 0)
	putTree(e, probs, 0x221a, 1, 0)
	putSInt(&models[21], e, 3, 1, 4, true, 1, 0)
	putTree(e, probs, 0x2220, 2, 0)
	putSInt(&models[24], e, 3, 1, 2, false, 1, 0)
	putSInt(&models[25], e, 3, 1, 4, false, 1, 0)
	putTree(e, probs, 0x2226, 1, 1)
	putSInt(&models[26], e, 3, 1, 4, false, 1, 0)
	putTree(e, probs, 0x2234, 2, 0)
	putTree(e, probs, 0x2238, 4, 0)
	putSInt(&models[31], e, 3, 1, 2, false, 1, 0)
	putTree(e, probs, 0x2248, 2, 1)
	putSInt(&models[32], e, 4, 1, 4, false, 1, 0)
	putSInt(&models[33], e, 4, 1, 4, false, 1, 0)
	putSInt(&models[34], e, 4, 1, 1, true, 1, 0)
	putSInt(&models[35], e, 3, 1, 4, false, 1, 0)
	models[37].value = models[29].value
	models[37].delta = 0
	putSInt(&models[37], e, 3, 1, 4, true, 2, 0)
	putSInt(&models[36], e, 3, 1, 2, false, 1, 0)
	putSInt(&models[38], e, 3, 1, 2, false, 1, 0)
	putSInt(&models[39], e, 3, 1, 2, false, 1, 0)
	putSInt(&models[40], e, 3, 1, 2, false, 1, 0)
	putSInt(&models[43], e, 3, 1, 2, false, 1, 0)
	putSInt(&models[44], e, 3, 1, 2, false, 1, 0)
	putSInt(&models[45], e, 3, 1, 2, false, 1, 0)
	putSInt(&models[46], e, 3, 1, 2, false, 1, 0)
	putTree(e, probs, 0x224c, 1, 0)
	putSInt(&models[47], e, 3, 1, 2, false, 1, 0)
}

func synthRaw(st encStyle) (src, want []byte) {
	a := []byte("raw-one")
	b := []byte{0, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	var ctrl [18]uint16
	for i := range ctrl {
		ctrl[i] = 0x8000
	}
	cmd := newRng(st)
	cmd.bit(&ctrl[0], 0)
	cmd.bit(&ctrl[0], 0)
	cmd.bit(&ctrl[0], 0)
	src = wrapOGGRE(0, cmd.finish(), a, b)
	return src, append(append([]byte{}, a...), b...)
}

func synthVorbis(st encStyle) []byte {
	identN := uint32(identPacketLen())
	setupN := uint32(setupPacketLen())
	models, probs := freshModels()
	cmd := newRng(st)
	aux := newRng(st)
	hdr := newRng(st)
	body := newRng(st)
	var ctrl [18]uint16
	for i := range ctrl {
		ctrl[i] = 0x8000
	}
	cmd.bit(&ctrl[0], 1)
	cmd.bit(&ctrl[2], 0)
	cmd.bit(&ctrl[6], 1)
	putPage(models, hdr, probs, 2, 2, 0, 0, 0, []uint32{identN}, true)
	putIdent(models, body)
	cmd.bit(&ctrl[10], 0)
	putPage(models, hdr, probs, 0, 0, 0, 0, 0, []uint32{0, setupN}, false)
	putLaces(models, hdr, probs, []uint32{0, setupN})
	putSetup(models, body, probs)
	cmd.bit(&ctrl[14], 0)
	putPage(models, hdr, probs, 0, 4, 0, 0, 0, []uint32{1}, true)
	body.bit(&probs[0x2255], 0)
	putSInt(&models[7], aux, 3, 0, 1, true, 0, 0)
	cmd.bit(&ctrl[1], 1)
	cmd.bit(&ctrl[2], 1)
	putSInt(&models[52], cmd, 5, 0, 2, true, 0, 0)
	putSInt(&models[3], cmd, 5, 2, 4, false, 1, 7)
	cmd.bit(&ctrl[1], 0)
	return wrapOGGRE(0, cmd.finish(), aux.finish(), hdr.finish(), body.finish())
}

func TestRangeEncoderBits(t *testing.T) {
	require.True(t, roundBits(lockedStyle(), sampleBits()))
}

func TestRangeEncoderInts(t *testing.T) {
	var encM, decM integerModel
	encM.reset()
	decM.reset()
	e := newRng(lockedStyle())
	vals := []uint32{0, 1, 2, 4, 30, 51, 102, 0, 7}
	for _, v := range vals {
		putSInt(&encM, e, 4, 1, 4, true, 1, v)
	}
	r := newRange(e.finish())
	for _, v := range vals {
		got := decM.predict(decM.integer(r, 4, 1, 4, true), 1)
		require.NoError(t, r.err)
		require.Equal(t, v, got)
	}
	require.Equal(t, 30, identPacketLen())
	require.Equal(t, 51, setupPacketLen())
}

func requireOracle(t *testing.T, out []byte, n int, ieee uint32, sum string) {
	t.Helper()
	require.Equal(t, n, len(out))
	require.Equal(t, ieee, crc32.ChecksumIEEE(out))
	got := sha256.Sum256(out)
	require.Equal(t, sum, hex.EncodeToString(got[:]))
}

func TestDecodeRaw(t *testing.T) {
	src, want := synthRaw(lockedStyle())
	require.Equal(t, 31, len(src))
	out, err := DecodeGuest(t.Context(), src, 0)
	require.NoError(t, err)
	require.Equal(t, want, out)
	requireOracle(t, out, 17, 0xaca3603a, "2794f7e99254a2c76d3fda5dfa89d662132e3bd883d5cfc743f14e1bc416e7f3")
}

func TestDecodeVorbis(t *testing.T) {
	src := synthVorbis(lockedStyle())
	require.Equal(t, 44, len(src))
	out, err := DecodeGuest(t.Context(), src, 0)
	require.NoError(t, err)
	require.Equal(t, "OggS", string(out[:4]))
	requireOracle(t, out, 334, 0x26154179, "cfaeb9f39fc98037fe090e9546f41aa5b695acf8efc0f25a502ec8441a315f86")
}

func TestDecodeCap(t *testing.T) {
	src, want := synthRaw(lockedStyle())
	_, rc, err := decodeAll(t.Context(), src, 8)
	require.NoError(t, err)
	require.Equal(t, int32(-2), rc)
	out, err := DecodeGuest(t.Context(), src, 0)
	require.NoError(t, err)
	require.Equal(t, want, out)
}

func TestDecodeCanceled(t *testing.T) {
	src, _ := synthRaw(lockedStyle())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := DecodeGuest(ctx, src, 0)
	require.ErrorIs(t, err, context.Canceled)
}

func TestDecodeShortHead(t *testing.T) {
	_, err := DecodeGuest(t.Context(), fg01Head, 0)
	require.ErrorIs(t, err, errGuest)
}
