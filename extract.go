package garotafitness

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/fs"
	"log/slog"

	"github.com/lucasew/garotafitness/setupdata"
)

// Extractor reads Source and writes Dest.
type Extractor struct {
	Source fs.FS
	Dest   Dest
}

func (e Extractor) Extract(ctx context.Context) error {
	if e.Source == nil {
		return fmt.Errorf("nil source")
	}
	if e.Dest == nil {
		return fmt.Errorf("nil dest")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	setup, setupErr := readSetup(e.Source)
	if err := setupErr; err != nil {
		return fmt.Errorf("setup.exe: %w", err)
	} else if len(setup.Encoders) > 0 {
		slog.Info("setup encoders", "names", setup.Encoders)
	}
	if n := len(setup.Operations); n > 0 {
		slog.Info("setup reconstruction records", "count", n)
	}
	optional, err := sourceComponents(setup.Operations)
	if err != nil {
		return err
	}
	vols, err := listVolumes(e.Source)
	if err != nil {
		return err
	}
	slog.Info("volumes", "count", len(vols))
	for i := range vols {
		if flag, ok := optional[vols[i].Name]; ok {
			vols[i].Optional = flag
		}
	}
	if err := verifyChecksums(ctx, e.Source, vols, optional); err != nil {
		return err
	}
	if setup.InstalledMD5 != "" || len(setup.Operations) != 0 {
		slog.Info("reconstruct from setup metadata")
		return e.extractReconstructed(ctx, vols, setup)
	}
	for _, v := range vols {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := extractVolume(ctx, e, v); err != nil {
			return err
		}
	}
	return nil
}

func scanSetup(src fs.FS) ([]string, error) {
	info, err := readSetup(src)
	return info.Encoders, err
}

func readSetup(src fs.FS) (setupdata.Info, error) {
	f, err := src.Open("setup.exe")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return setupdata.Info{}, nil
		}
		return setupdata.Info{}, err
	}
	defer f.Close()
	return setupdata.Scan(f)
}

func extractVolume(ctx context.Context, e Extractor, v Volume) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f, err := e.Source.Open(v.Name)
	if err != nil {
		return fmt.Errorf("open %s: %w", v.Name, err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("read %s: %w", v.Name, err)
	}
	slog.Info("read volume", "name", v.Name, "bytes", len(data))
	return extractVolumeData(ctx, e, v.Name, data)
}

func extractVolumeData(ctx context.Context, e Extractor, name string, data []byte) error {
	parsed, err := parseVolume(name, data)
	if err != nil {
		return err
	}
	for _, m := range parsed.Members {
		if !m.Dir {
			continue
		}
		if err := e.Dest.MkdirAll(m.Path, 0o755); err != nil {
			return err
		}
	}
	return extractSolids(ctx, e, data, groupSolids(parsed.Members))
}

func extractSolids(ctx context.Context, e Extractor, data []byte, solids []solid) error {
	fns := make([]func(context.Context) error, len(solids))
	for i, s := range solids {
		fns[i] = func(ctx context.Context) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			return extractSolid(e, data, s)
		}
	}
	return runParallel(ctx, fns)
}

type solid struct {
	pipe  Pipeline
	off   int64
	csz   uint64
	files []Member
}

func groupSolids(ms []Member) []solid {
	type key struct {
		pipe string
		off  int64
		csz  uint64
	}
	order := make([]key, 0)
	by := make(map[key]*solid)
	for _, m := range ms {
		if m.Dir {
			continue
		}
		k := key{m.Pipeline.String(), m.Offset, m.CompSize}
		s, ok := by[k]
		if !ok {
			s = &solid{pipe: m.Pipeline, off: m.Offset, csz: m.CompSize}
			by[k] = s
			order = append(order, k)
		}
		s.files = append(s.files, m)
	}
	out := make([]solid, 0, len(order))
	for _, k := range order {
		out = append(out, *by[k])
	}
	return out
}

func extractSolid(e Extractor, data []byte, s solid) error {
	if len(s.files) == 0 {
		return nil
	}
	if len(s.pipe) == 0 {
		return unknownEncoderError(Atom{})
	}
	slog.Info("decode solid", "pipeline", s.pipe.String(), "offset", s.off, "compressed", s.csz, "members", len(s.files))
	end := s.off + int64(s.csz)
	if s.off < 0 || end > int64(len(data)) {
		return fmt.Errorf("solid span")
	}
	var (
		src     io.Reader = bytes.NewReader(data[s.off:end])
		closers []io.Closer
	)
	defer func() {
		for i := len(closers) - 1; i >= 0; i-- {
			closers[i].Close()
		}
	}()
	for i := len(s.pipe) - 1; i >= 0; i-- {
		r, err := Decode(src, s.pipe[i])
		if err != nil {
			return err
		}
		closers = append(closers, r)
		src = r
	}
	for _, m := range s.files {
		if err := writeMember(e.Dest, m, io.LimitReader(src, int64(m.Size))); err != nil {
			return err
		}
	}
	var extra [1]byte
	if _, err := io.ReadFull(src, extra[:]); err != io.EOF {
		if err != nil {
			return fmt.Errorf("finish solid: %w", err)
		}
		return fmt.Errorf("solid contains data after its final member")
	}
	return nil
}

func writeMember(dst Dest, m Member, r io.Reader) error {
	w, err := dst.Create(m.Path)
	if err != nil {
		return err
	}
	table := m.crcTable
	if table == nil {
		table = crc32.IEEETable
	}
	h := crc32.New(table)
	n, err := io.Copy(w, io.TeeReader(r, h))
	if err != nil {
		w.Close()
		return fmt.Errorf("write %s: %w", m.Path, err)
	}
	if err := w.Close(); err != nil {
		return err
	}
	if uint64(n) != m.Size {
		return fmt.Errorf("write %s: size %d want %d", m.Path, n, m.Size)
	}
	if (m.crcTable != nil || m.CRC != 0) && h.Sum32() != m.CRC {
		return fmt.Errorf("write %s: crc %08x want %08x", m.Path, h.Sum32(), m.CRC)
	}
	slog.Debug("wrote member", "path", m.Path, "size", n, "crc", fmt.Sprintf("%08x", h.Sum32()))
	return nil
}
