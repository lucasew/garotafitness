package garotafitness

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
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
	vols, err := listVolumes(e.Source)
	if err != nil {
		return err
	}
	if err := verifyChecksums(e.Source, vols); err != nil {
		return err
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
	parsed, err := parseVolume(v.Name, data)
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
	for _, s := range groupSolids(parsed.Members) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := extractSolid(e, data, s); err != nil {
			return err
		}
	}
	return nil
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
	if len(s.pipe) != 1 {
		return unknownEncoderError(s.pipe.Last())
	}
	end := s.off + int64(s.csz)
	if s.off < 0 || end > int64(len(data)) {
		return fmt.Errorf("solid span")
	}
	r, err := openDecoder(bytes.NewReader(data[s.off:end]), s.pipe[0])
	if err != nil {
		return err
	}
	defer r.Close()
	for _, m := range s.files {
		if err := writeMember(e.Dest, m, io.LimitReader(r, int64(m.Size))); err != nil {
			return err
		}
	}
	return nil
}

func writeMember(dst Dest, m Member, r io.Reader) error {
	w, err := dst.Create(m.Path)
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, r); err != nil {
		w.Close()
		return fmt.Errorf("write %s: %w", m.Path, err)
	}
	return w.Close()
}
