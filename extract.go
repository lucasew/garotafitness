package garotafitness

import (
	"context"
	"fmt"
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
	parsed, err := readVolume(f, v.Name)
	if err != nil {
		return err
	}
	for _, m := range parsed.Members {
		if m.Dir {
			continue
		}
		return unknownEncoderError(m.Pipeline.Last())
	}
	return fmt.Errorf("unknown encoder: volume %s has no members", v.Name)
}
