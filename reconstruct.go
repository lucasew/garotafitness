package garotafitness

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"path"
	"slices"
	"strings"

	"github.com/lucasew/garotafitness/setupdata"
)

// Intermediates stay in memory: Source remains read-only, and Dest needs no read,
// seek, rename, or delete operations. Only completed files are written to Dest.
type reconstruction struct {
	files map[string][]byte
	dirs  map[string]fs.FileMode
}

func (s *reconstruction) MkdirAll(name string, mode fs.FileMode) error {
	if _, err := memberPath(".", name); err != nil {
		return err
	}
	s.dirs[name] = mode
	return nil
}

func (s *reconstruction) Create(name string) (io.WriteCloser, error) {
	if _, err := memberPath(".", name); err != nil {
		return nil, err
	}
	return &stagedFile{store: s, name: name}, nil
}

type stagedFile struct {
	bytes.Buffer
	store *reconstruction
	name  string
}

func (f *stagedFile) Close() error { f.store.files[f.name] = f.Bytes(); return nil }

func (s *reconstruction) require(name string) ([]byte, error) {
	b, ok := s.files[name]
	if !ok {
		for candidate, data := range s.files {
			if strings.EqualFold(candidate, name) {
				if ok {
					return nil, fmt.Errorf("reconstruction: ambiguous filename %s", name)
				}
				b, ok = data, true
			}
		}
		if !ok {
			return nil, fmt.Errorf("reconstruction: missing %s", name)
		}
	}
	return b, nil
}

func (e Extractor) extractReconstructed(ctx context.Context, vols []Volume, setup setupdata.Info) error {
	s := newStaging()
	runner := &reconstructionPlan{source: e.Source, app: s, temp: newStaging(), volumes: map[string]Volume{}, remaining: map[string]int{}, decoded: map[string]*reconstruction{}, seen: map[string]bool{}}
	for _, v := range vols {
		runner.volumes[v.Name] = v
	}
	manifestPath := ""
	if setup.InstalledMD5 != "" {
		if setup.ManifestPath == "" {
			return fmt.Errorf("installed checksum: unresolved destination in setup metadata")
		}
		name, err := virtualPath(setup.ManifestPath, "")
		if err != nil {
			return err
		}
		if !strings.HasPrefix(name, "app/") {
			return fmt.Errorf("installed checksum: destination outside installed files")
		}
		manifestPath = strings.TrimPrefix(name, "app/")
		s.files[manifestPath] = []byte(setup.InstalledMD5)
	}
	if len(setup.Operations) > 0 {
		if err := runner.run(ctx, setup.Operations); err != nil {
			return err
		}
	} else {
		for _, v := range vols {
			if err := extractVolume(ctx, Extractor{Source: e.Source, Dest: s}, v); err != nil {
				return err
			}
		}
	}
	if manifestPath != "" {
		manifest, err := s.require(manifestPath)
		if err != nil {
			return err
		}
		slog.Info("verify installed checksums", "manifest", manifestPath)
		if err := s.verifyInstalled(ctx, string(manifest), path.Dir(manifestPath)); err != nil {
			return err
		}
	}
	slog.Info("write reconstructed tree", "dirs", len(s.dirs), "files", len(s.files))
	for _, name := range sortedKeys(s.dirs) {
		if err := e.Dest.MkdirAll(name, s.dirs[name]); err != nil {
			return err
		}
	}
	for _, name := range sortedKeys(s.files) {
		if err := ctx.Err(); err != nil {
			return err
		}
		b := s.files[name]
		if err := writeMember(e.Dest, Member{Path: name, Size: uint64(len(b))}, bytes.NewReader(b)); err != nil {
			return err
		}
		delete(s.files, name)
	}
	return nil
}

func sortedKeys[V any](m map[string]V) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func (s *reconstruction) verifyInstalled(ctx context.Context, manifest, directory string) error {
	sc := bufio.NewScanner(strings.NewReader(manifest))
	count := 0
	for sc.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		line := strings.TrimSuffix(sc.Text(), "\r")
		if len(line) < 35 || line[32:34] != " *" {
			return fmt.Errorf("installed checksum: invalid manifest line")
		}
		resolved, err := virtualPath(line[34:], path.Join("app", directory))
		if err != nil {
			return fmt.Errorf("installed checksum: %w", err)
		}
		name := strings.TrimPrefix(resolved, "app/")
		want, err := hex.DecodeString(line[:32])
		if err != nil {
			return fmt.Errorf("installed checksum: %w", err)
		}
		b, err := s.require(name)
		if err != nil {
			return err
		}
		got := md5.Sum(b)
		if !bytes.Equal(got[:], want) {
			return fmt.Errorf("installed checksum: %s: %x want %x", name, got, want)
		}
		count++
	}
	if err := sc.Err(); err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("installed checksum: empty manifest")
	}
	slog.Info("installed hashes verified", "files", count)
	return nil
}
