package garotafitness

import (
	"fmt"
	"hash/crc32"
	"io"
	"path"
	"strings"
)

const maxFooter = 4096

// Member is one file or directory inside a Volume.
type Member struct {
	Path     string
	Size     uint64
	Pipeline Pipeline
	Offset   int64
	CompSize uint64
	CRC      uint32
	Dir      bool
	crcTable *crc32.Table
}

func parseVolume(name string, data []byte) (Volume, error) {
	if len(data) < 4 || string(data[:4]) != arcMagic {
		return Volume{}, fmt.Errorf("%s: not ArC", name)
	}
	pos := lastSignature(data)
	if pos < 0 {
		return Volume{}, fmt.Errorf("%s: no footer descriptor", name)
	}
	loc, err := parseLocal(data[pos:])
	if err != nil {
		return Volume{}, fmt.Errorf("%s footer: %w", name, err)
	}
	if loc.kind != BlockFooter {
		return Volume{}, fmt.Errorf("%s: footer type %s", name, loc.kind)
	}
	if loc.csz > uint64(pos) {
		return Volume{}, fmt.Errorf("%s: footer compsize", name)
	}
	fpos := int64(pos) - int64(loc.csz)
	footer, err := rawLZMA1(data[fpos:pos], int(loc.orig))
	if err != nil {
		return Volume{}, fmt.Errorf("%s footer lzma: %w", name, err)
	}
	if crc32.Checksum(footer, loc.table) != loc.crc {
		return Volume{}, fmt.Errorf("%s: footer CRC mismatch", name)
	}
	blocks, err := parseControl(footer, fpos)
	if err != nil {
		return Volume{}, fmt.Errorf("%s control: %w", name, err)
	}
	var members []Member
	var pipes []Pipeline
	for _, b := range blocks {
		b.table = loc.table
		pipes = append(pipes, b.pipe)
		if b.kind != BlockDir {
			continue
		}
		ms, err := parseDir(data, b)
		if err != nil {
			return Volume{}, fmt.Errorf("%s dir: %w", name, err)
		}
		members = append(members, ms...)
		for _, m := range ms {
			pipes = append(pipes, m.Pipeline)
		}
	}
	return Volume{
		Name:     name,
		Optional: strings.Contains(strings.ToLower(path.Base(name)), optionalMark),
		Algos:    mergeAlgos(pipes...),
		Members:  members,
	}, nil
}

type localDesc struct {
	kind  BlockKind
	pipe  Pipeline
	orig  uint64
	csz   uint64
	crc   uint32
	table *crc32.Table
}

type ctrlBlock struct {
	kind  BlockKind
	pipe  Pipeline
	pos   int64
	orig  uint64
	csz   uint64
	crc   uint32
	table *crc32.Table
}

func mergeAlgos(ps ...Pipeline) []Algo {
	seen := make(map[Algo]struct{})
	out := make([]Algo, 0)
	for _, p := range ps {
		for _, a := range p.Algos() {
			if !a.Known() {
				continue
			}
			if _, ok := seen[a]; ok {
				continue
			}
			seen[a] = struct{}{}
			out = append(out, a)
		}
	}
	return out
}

func lastSignature(data []byte) int {
	end := len(data)
	start := end - maxFooter
	if start < 0 {
		start = 0
	}
	sig := []byte(arcMagic)
	pos := -1
	for i := start; i+4 <= end; i++ {
		if data[i] == sig[0] && data[i+1] == sig[1] && data[i+2] == sig[2] && data[i+3] == sig[3] {
			pos = i
		}
	}
	return pos
}

func parseLocal(chunk []byte) (localDesc, error) {
	if len(chunk) < 8 || string(chunk[:4]) != arcMagic {
		return localDesc{}, fmt.Errorf("bad magic")
	}
	i := 4
	typ, i, err := readPacked(chunk, i)
	if err != nil {
		return localDesc{}, err
	}
	comp, i, err := readCString(chunk, i)
	if err != nil {
		return localDesc{}, err
	}
	orig, i, err := readPacked(chunk, i)
	if err != nil {
		return localDesc{}, err
	}
	csz, i, err := readPacked(chunk, i)
	if err != nil {
		return localDesc{}, err
	}
	crc, i, err := readU32(chunk, i)
	if err != nil {
		return localDesc{}, err
	}
	want, _, err := readU32(chunk, i)
	if err != nil {
		return localDesc{}, err
	}
	for _, table := range []*crc32.Table{crc32.IEEETable, fitgirlCRCTable} {
		if crc32.Checksum(chunk[:i], table) == want {
			return localDesc{kind: blockKind(int(typ)), pipe: ParsePipeline(comp), orig: orig, csz: csz, crc: crc, table: table}, nil
		}
	}
	return localDesc{}, fmt.Errorf("descriptor CRC mismatch")
}

func parseControl(footer []byte, footerPos int64) ([]ctrlBlock, error) {
	n, i, err := readPacked(footer, 0)
	if err != nil {
		return nil, err
	}
	out := make([]ctrlBlock, 0, n)
	for range int(n) {
		typ, j, err := readPacked(footer, i)
		if err != nil {
			return nil, err
		}
		comp, j, err := readCString(footer, j)
		if err != nil {
			return nil, err
		}
		rel, j, err := readPacked(footer, j)
		if err != nil {
			return nil, err
		}
		orig, j, err := readPacked(footer, j)
		if err != nil {
			return nil, err
		}
		csz, j, err := readPacked(footer, j)
		if err != nil {
			return nil, err
		}
		crc, j, err := readU32(footer, j)
		if err != nil {
			return nil, err
		}
		out = append(out, ctrlBlock{
			kind: blockKind(int(typ)),
			pipe: ParsePipeline(comp),
			pos:  footerPos - int64(rel),
			orig: orig,
			csz:  csz,
			crc:  crc,
		})
		i = j
	}
	return out, nil
}

func parseDir(data []byte, b ctrlBlock) ([]Member, error) {
	if b.pos < 0 || b.csz > uint64(len(data)) || int(b.pos)+int(b.csz) > len(data) {
		return nil, fmt.Errorf("dir span")
	}
	raw, err := rawLZMA1(data[b.pos:int(b.pos)+int(b.csz)], int(b.orig))
	if err != nil {
		return nil, err
	}
	if b.table == nil || crc32.Checksum(raw, b.table) != b.crc {
		return nil, fmt.Errorf("directory CRC mismatch")
	}
	nb, i, err := readPacked(raw, 0)
	if err != nil {
		return nil, err
	}
	nfiles := make([]int, nb)
	for k := range nfiles {
		v, j, err := readPacked(raw, i)
		if err != nil {
			return nil, err
		}
		nfiles[k] = int(v)
		i = j
	}
	comps := make([]string, nb)
	for k := range comps {
		s, j, err := readCString(raw, i)
		if err != nil {
			return nil, err
		}
		comps[k] = s
		i = j
	}
	offs := make([]uint64, nb)
	for k := range offs {
		v, j, err := readPacked(raw, i)
		if err != nil {
			return nil, err
		}
		offs[k] = v
		i = j
	}
	cszs := make([]uint64, nb)
	for k := range cszs {
		v, j, err := readPacked(raw, i)
		if err != nil {
			return nil, err
		}
		cszs[k] = v
		i = j
	}
	nd, i, err := readPacked(raw, i)
	if err != nil {
		return nil, err
	}
	dirs := make([]string, nd)
	for k := range dirs {
		s, j, err := readCString(raw, i)
		if err != nil {
			return nil, err
		}
		dirs[k] = s
		i = j
	}
	total := 0
	for _, n := range nfiles {
		total += n
	}
	names := make([]string, total)
	for k := range names {
		s, j, err := readCString(raw, i)
		if err != nil {
			return nil, err
		}
		names[k] = s
		i = j
	}
	dnums := make([]int, total)
	for k := range dnums {
		v, j, err := readPacked(raw, i)
		if err != nil {
			return nil, err
		}
		dnums[k] = int(v)
		i = j
	}
	sizes := make([]uint64, total)
	for k := range sizes {
		v, j, err := readPacked(raw, i)
		if err != nil {
			return nil, err
		}
		sizes[k] = v
		i = j
	}
	if i+4*total+total+4*total > len(raw) {
		return nil, fmt.Errorf("dir truncated")
	}
	times := make([]uint32, total)
	for k := range times {
		times[k], i, err = readU32(raw, i)
		if err != nil {
			return nil, err
		}
	}
	isdir := raw[i : i+total]
	i += total
	crcs := make([]uint32, total)
	for k := range crcs {
		crcs[k], i, err = readU32(raw, i)
		if err != nil {
			return nil, err
		}
	}
	_ = times
	out := make([]Member, 0, total)
	idx := 0
	for bi, nf := range nfiles {
		for range nf {
			dir := ""
			if dnums[idx] >= 0 && dnums[idx] < len(dirs) {
				dir = dirs[dnums[idx]]
			}
			p := names[idx]
			if dir != "" {
				p = path.Join(dir, names[idx])
			}
			out = append(out, Member{
				Path:     p,
				Size:     sizes[idx],
				Pipeline: ParsePipeline(comps[bi]),
				Offset:   b.pos - int64(offs[bi]),
				CompSize: cszs[bi],
				CRC:      crcs[idx],
				Dir:      isdir[idx] != 0,
				crcTable: b.table,
			})
			idx++
		}
	}
	return out, nil
}

func readVolume(r io.Reader, name string) (Volume, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return Volume{}, fmt.Errorf("read %s: %w", name, err)
	}
	return parseVolume(name, data)
}
