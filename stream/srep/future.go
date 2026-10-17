package srep

import "encoding/binary"

// Future-LZ kernel from third_party/srep Compression/SREP/decompress.cpp.
// The match heap, chunk allocator, and spill blocks live for the whole
// stream, which is what srep_open / srep_block / srep_close did.

const (
	srepChunk   = 64
	srepUseful  = srepChunk - 4
	srepBlock   = 1 << 20
	srepK       = srepBlock / srepChunk
	srepLbK     = 14
	srepVMBlock = 8 << 20
	srepVMMem   = 512 << 20
	statWords   = 4
	statBytes   = statWords * 4
)

type memMgr struct {
	blocks    [][]byte
	firstFree uint32
	used      uint64
	useful    uint64
}

func newMemMgr(limit uint64) memMgr {
	useful := (limit/srepBlock*srepBlock/srepChunk - 1) * srepUseful
	return memMgr{useful: useful}
}

func (m *memMgr) chunk(index uint32) []byte {
	off := (index & (srepK - 1)) * srepChunk
	return m.blocks[index>>srepLbK][off : off+srepChunk]
}

func (m *memMgr) next(index uint32) uint32 {
	return binary.LittleEndian.Uint32(m.chunk(index)[:4])
}

func (m *memMgr) setNext(index, next uint32) {
	binary.LittleEndian.PutUint32(m.chunk(index)[:4], next)
}

func (m *memMgr) allocBlock() {
	m.blocks = append(m.blocks, make([]byte, srepBlock))
	block := uint32(len(m.blocks) - 1)
	base := block * srepK
	last := base + srepK - 1
	for i := base; i < last; i++ {
		m.setNext(i, i+1)
	}
	m.setNext(last, m.firstFree)
	m.firstFree = base
	if m.firstFree == 0 {
		m.firstFree++
	}
}

func (m *memMgr) allocate() uint32 {
	if m.firstFree == 0 {
		m.allocBlock()
	}
	free := m.firstFree
	m.firstFree = m.next(free)
	m.used++
	return free
}

func (m *memMgr) markFree(index uint32) {
	m.setNext(index, m.firstFree)
	m.firstFree = index
	m.used--
}

func (m *memMgr) save(p []byte) uint32 {
	if len(p) == 0 {
		return 0
	}
	var index, first, prev uint32
	for len(p) > 0 {
		index = m.allocate()
		if prev != 0 {
			m.setNext(prev, index)
		} else {
			first = index
		}
		prev = index
		n := min(len(p), srepUseful)
		copy(m.chunk(index)[4:4+n], p[:n])
		p = p[n:]
	}
	m.setNext(index, 0)
	return first
}

func (m *memMgr) restore(index uint32, dst []byte) {
	for len(dst) > 0 {
		n := min(len(dst), srepUseful)
		copy(dst[:n], m.chunk(index)[4:4+n])
		dst = dst[n:]
		index = m.next(index)
	}
}

func (m *memMgr) free(index uint32) {
	for index != 0 {
		next := m.next(index)
		m.markFree(index)
		index = next
	}
}

func (m *memMgr) available() uint64 {
	used := m.used * srepUseful
	if m.useful > used {
		return m.useful - used
	}
	return 0
}

type match struct {
	src, dest uint64
	length    uint32
	index     uint32
}

type mnode struct {
	match
	seq         uint64
	pri         uint32
	left, right *mnode
	parent      *mnode
}

// heap orders matches by dest, then insertion order. Nodes stay put across
// inserts, matching std::multiset iterator stability.
type heap struct {
	root *mnode
	seq  uint64
	rng  uint32
}

func (h *heap) pri() uint32 {
	x := h.rng
	if x == 0 {
		x = 0x9e3779b9
	}
	x ^= x << 13
	x ^= x >> 17
	x ^= x << 5
	h.rng = x
	return x
}

func before(a, b *mnode) bool {
	if a.dest != b.dest {
		return a.dest < b.dest
	}
	return a.seq < b.seq
}

func (h *heap) rotateLeft(n *mnode) {
	r := n.right
	n.right = r.left
	if r.left != nil {
		r.left.parent = n
	}
	r.parent = n.parent
	if n.parent == nil {
		h.root = r
	} else if n.parent.left == n {
		n.parent.left = r
	} else {
		n.parent.right = r
	}
	r.left = n
	n.parent = r
}

func (h *heap) rotateRight(n *mnode) {
	l := n.left
	n.left = l.right
	if l.right != nil {
		l.right.parent = n
	}
	l.parent = n.parent
	if n.parent == nil {
		h.root = l
	} else if n.parent.left == n {
		n.parent.left = l
	} else {
		n.parent.right = l
	}
	l.right = n
	n.parent = l
}

func (h *heap) insert(m match) *mnode {
	n := &mnode{match: m, seq: h.seq, pri: h.pri()}
	h.seq++
	if h.root == nil {
		h.root = n
		return n
	}
	cur := h.root
	for {
		if before(n, cur) {
			if cur.left == nil {
				cur.left = n
				n.parent = cur
				break
			}
			cur = cur.left
			continue
		}
		if cur.right == nil {
			cur.right = n
			n.parent = cur
			break
		}
		cur = cur.right
	}
	for n.parent != nil && n.pri > n.parent.pri {
		if n.parent.left == n {
			h.rotateRight(n.parent)
		} else {
			h.rotateLeft(n.parent)
		}
	}
	return n
}

func (h *heap) eraseNode(n *mnode) {
	if n == nil || (n.parent == nil && h.root != n) {
		return
	}
	for n.left != nil || n.right != nil {
		if n.left == nil || (n.right != nil && n.right.pri > n.left.pri) {
			h.rotateLeft(n)
		} else {
			h.rotateRight(n)
		}
	}
	if n.parent == nil {
		h.root = nil
	} else if n.parent.left == n {
		n.parent.left = nil
	} else {
		n.parent.right = nil
	}
	n.left, n.right, n.parent = nil, nil, nil
}

func (h *heap) eraseDest(dest uint64) {
	var doomed []*mnode
	var walk func(*mnode)
	walk = func(n *mnode) {
		if n == nil {
			return
		}
		if n.dest > dest {
			walk(n.left)
			return
		}
		if n.dest < dest {
			walk(n.right)
			return
		}
		walk(n.left)
		doomed = append(doomed, n)
		walk(n.right)
	}
	walk(h.root)
	for _, n := range doomed {
		h.eraseNode(n)
	}
}

func (n *mnode) prev() *mnode {
	if n == nil {
		return nil
	}
	if n.left != nil {
		x := n.left
		for x.right != nil {
			x = x.right
		}
		return x
	}
	for n.parent != nil && n == n.parent.left {
		n = n.parent
	}
	return n.parent
}

func (n *mnode) next() *mnode {
	if n == nil {
		return nil
	}
	if n.right != nil {
		x := n.right
		for x.left != nil {
			x = x.left
		}
		return x
	}
	for n.parent != nil && n == n.parent.right {
		n = n.parent
	}
	return n.parent
}

func (h *heap) rightmost() *mnode {
	x := h.root
	for x != nil && x.right != nil {
		x = x.right
	}
	return x
}

func (h *heap) begin() *mnode {
	x := h.root
	for x != nil && x.left != nil {
		x = x.left
	}
	return x
}

// pred of a forward iterator. A nil base is end(), whose predecessor is the
// rightmost node.
func (h *heap) pred(base *mnode) *mnode {
	if base == nil {
		return h.rightmost()
	}
	return base.prev()
}

type virtMem struct {
	blockSize uint64
	buf       []byte
	blocks    [][]byte
	free      []uint32
	next      uint32
}

// save spills match data the way VIRTUAL_MEMORY_MANAGER::save_to_disk does.
// The reverse walk increments first, so the largest dest is skipped, and
// erasing the current dest leaves the iterator on the new predecessor: the
// following increment skips that match too.
func (v *virtMem) save(mm *memMgr, h *heap) bool {
	if v.blockSize == 0 {
		return false
	}
	if v.buf == nil || uint64(len(v.buf)) != v.blockSize {
		v.buf = make([]byte, v.blockSize)
	}
	before := mm.used
	minDest := ^uint64(0)
	p := 0
	base := (*mnode)(nil)
	for {
		base = h.pred(base)
		cur := h.pred(base)
		if cur == nil {
			break
		}
		if cur.index == 0 {
			continue
		}
		need := 24 + uint64(cur.length)
		if uint64(len(v.buf)-p) < need {
			break
		}
		binary.LittleEndian.PutUint32(v.buf[p:], cur.length)
		binary.LittleEndian.PutUint64(v.buf[p+4:], cur.src)
		binary.LittleEndian.PutUint64(v.buf[p+12:], cur.dest)
		minDest = cur.dest
		mm.restore(cur.index, v.buf[p+20:p+20+int(cur.length)])
		p += 20 + int(cur.length)
		mm.free(cur.index)
		cur.index = 0
		if base != nil && base.dest == cur.dest {
			for base != nil && base.dest == cur.dest {
				base = base.next()
			}
		}
		h.eraseDest(cur.dest)
	}
	if mm.used >= before {
		return false
	}
	binary.LittleEndian.PutUint32(v.buf[p:], 0)
	var block uint32
	if len(v.free) == 0 {
		block = v.next
		v.next++
	} else {
		block = v.free[len(v.free)-1]
		v.free = v.free[:len(v.free)-1]
	}
	for uint32(len(v.blocks)) <= block {
		v.blocks = append(v.blocks, nil)
	}
	stored := make([]byte, p+4)
	copy(stored, v.buf[:p+4])
	v.blocks[block] = stored
	h.insert(match{src: uint64(block), dest: minDest, length: 0})
	return true
}

func (v *virtMem) restore(mm *memMgr, h *heap, mark *mnode) bool {
	for mm.available() < v.blockSize {
		if !v.save(mm, h) {
			return false
		}
	}
	if mark.src > uint64(^uint32(0)) || uint32(mark.src) >= uint32(len(v.blocks)) {
		return false
	}
	block := uint32(mark.src)
	data := v.blocks[block]
	if data == nil {
		return false
	}
	v.blocks[block] = nil
	v.free = append(v.free, block)
	p := 0
	for {
		if p+4 > len(data) {
			return false
		}
		n := binary.LittleEndian.Uint32(data[p:])
		if n == 0 {
			return true
		}
		if uint64(p)+20+uint64(n) > uint64(len(data)) {
			return false
		}
		src := binary.LittleEndian.Uint64(data[p+4:])
		dest := binary.LittleEndian.Uint64(data[p+12:])
		idx := mm.save(data[p+20 : p+20+int(n)])
		h.insert(match{src: src, dest: dest, length: n, index: idx})
		p += 20 + int(n)
	}
}

type kernel struct {
	base uint32
	mm   memMgr
	vm   virtMem
	heap heap
}

func newKernel(base uint32) *kernel {
	return newKernelLimited(base, srepVMMem, srepVMBlock)
}

func newKernelLimited(base uint32, memLimit, vmBlock uint64) *kernel {
	k := &kernel{
		base: base,
		mm:   newMemMgr(memLimit),
		vm:   virtMem{blockSize: vmBlock},
		heap: heap{rng: 0x12345678},
	}
	// Barrier so the in-block loop always has a node to stop on.
	// len is STAT(-1), not a disk mark (those have len 0).
	k.heap.insert(match{src: ^uint64(0), dest: ^uint64(0), length: ^uint32(0)})
	return k
}

type decoded struct {
	src, dest uint64
	length    uint32
}

func decodeStats(stat []byte, base uint32, blockStart, blockEnd uint64) ([]decoded, bool) {
	var out []decoded
	pos := blockStart
	for len(stat) >= statBytes {
		lit := binary.LittleEndian.Uint32(stat[0:4])
		off := uint64(binary.LittleEndian.Uint32(stat[4:8])) | uint64(binary.LittleEndian.Uint32(stat[8:12]))<<32
		length := binary.LittleEndian.Uint32(stat[12:16]) + base
		stat = stat[statBytes:]
		src := pos + uint64(lit)
		dest := src + off
		if src < pos || src >= blockEnd || uint64(length) > blockEnd-src || dest <= src {
			return nil, false
		}
		out = append(out, decoded{src: src, dest: dest, length: length})
		pos = src
	}
	return out, true
}

func copyLZ(dst, src []byte) {
	for i := range dst {
		dst[i] = src[i]
	}
}

func (k *kernel) block(stat, lits, out []byte, blockStart uint64) bool {
	blockEnd := blockStart + uint64(len(out))
	matches, ok := decodeStats(stat, k.base, blockStart, blockEnd)
	if !ok {
		return false
	}
	for _, m := range matches {
		if m.dest < blockEnd {
			k.heap.insert(match{src: m.src, dest: m.dest, length: m.length})
		}
	}
	outPos := 0
	inPos := 0
	for {
		lz := k.heap.begin()
		if lz == nil || lz.dest >= blockEnd {
			break
		}
		if lz.length == 0 {
			if !k.vm.restore(&k.mm, &k.heap, lz) {
				return false
			}
			k.heap.eraseNode(lz)
			continue
		}
		if lz.dest < blockStart+uint64(outPos) || inPos > len(lits) || outPos > len(out) {
			return false
		}
		gap64 := lz.dest - blockStart - uint64(outPos)
		if gap64 > uint64(len(lits)-inPos) || gap64 > uint64(len(out)-outPos) {
			return false
		}
		gap := int(gap64)
		if uint64(lz.length) > uint64(len(out)-outPos-gap) {
			return false
		}
		copy(out[outPos:outPos+gap], lits[inPos:inPos+gap])
		inPos += gap
		outPos += gap
		dst := out[outPos : outPos+int(lz.length)]
		if lz.length == ^uint32(0) && lz.src < blockStart {
			return false
		}
		if lz.index != 0 {
			k.mm.restore(lz.index, dst)
		} else {
			if lz.src < blockStart || lz.src-blockStart > uint64(len(out)) {
				return false
			}
			srcOff := int(lz.src - blockStart)
			if srcOff+len(dst) > len(out) {
				return false
			}
			copyLZ(dst, out[srcOff:srcOff+len(dst)])
		}
		outPos += int(lz.length)
		if lz.index != 0 {
			k.mm.free(lz.index)
		}
		k.heap.eraseNode(lz)
	}
	if len(lits)-inPos != len(out)-outPos {
		return false
	}
	copy(out[outPos:], lits[inPos:])
	for _, m := range matches {
		if m.dest < blockEnd {
			continue
		}
		if m.length == ^uint32(0) {
			k.heap.insert(match{src: m.src, dest: m.dest, length: m.length})
			continue
		}
		for uint64(m.length) > k.mm.available() {
			if !k.vm.save(&k.mm, &k.heap) {
				return false
			}
		}
		rel := m.src - blockStart
		if rel > uint64(len(out)) || uint64(len(out))-rel < uint64(m.length) {
			return false
		}
		idx := k.mm.save(out[rel : rel+uint64(m.length)])
		k.heap.insert(match{src: m.src, dest: m.dest, length: m.length, index: idx})
	}
	return true
}
