package garotafitness

import (
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"
)

const (
	arcMagic     = "ArC\x01"
	optionalMark = "optional"
)

// Volume is one fg-*.bin under Source.
type Volume struct {
	Name     string
	Optional bool
	Algos    []Algo
	Members  []Member
}

func listVolumes(src fs.FS) ([]Volume, error) {
	names, err := fs.Glob(src, "fg-*.bin")
	if err != nil {
		return nil, fmt.Errorf("glob volumes: %w", err)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no fg-*.bin")
	}
	out := make([]Volume, 0, len(names))
	for _, name := range names {
		v, err := inspectVolume(src, name)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func inspectVolume(src fs.FS, name string) (Volume, error) {
	f, err := src.Open(name)
	if err != nil {
		return Volume{}, fmt.Errorf("open %s: %w", name, err)
	}
	defer f.Close()
	head := make([]byte, 64*1024)
	n, err := io.ReadFull(f, head)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return Volume{}, fmt.Errorf("read %s: %w", name, err)
	}
	head = head[:n]
	if len(head) < 4 || string(head[:4]) != arcMagic {
		return Volume{}, fmt.Errorf("%s: not ArC", name)
	}
	return Volume{
		Name:     name,
		Optional: strings.Contains(strings.ToLower(path.Base(name)), optionalMark),
	}, nil
}
