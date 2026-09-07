package garotafitness

import (
	"context"
	"fmt"
	"io/fs"
	"slices"
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
	_ = ctx
	_ = e
	if slices.Contains(v.Encoders, "srep") {
		return fmt.Errorf("unknown encoder srep")
	}
	return fmt.Errorf("unknown encoder: volume %s needs a guest", v.Name)
}
