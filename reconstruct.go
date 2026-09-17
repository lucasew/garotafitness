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
	"iter"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"sync"

	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lucasew/garotafitness/setupdata"
)

// Intermediates stay in memory: Source remains read-only, and Dest needs no read,
// seek, rename, or delete operations. Only completed files are written to Dest.
type reconstruction struct {
	mu    sync.Mutex
	files map[string][]byte
	dirs  map[string]fs.FileMode
}

func (s *reconstruction) MkdirAll(name string, mode fs.FileMode) error {
	if _, err := memberName(name); err != nil {
		return err
	}
	s.mu.Lock()
	s.dirs[name] = mode
	s.mu.Unlock()
	return nil
}

func (s *reconstruction) Create(name string) (io.WriteCloser, error) {
	if _, err := memberName(name); err != nil {
		return nil, err
	}
	return &stagedFile{store: s, name: name}, nil
}

type stagedFile struct {
	bytes.Buffer
	store *reconstruction
	name  string
}

func (f *stagedFile) Close() error {
	f.store.mu.Lock()
	f.store.files[f.name] = f.Bytes()
	f.store.mu.Unlock()
	return nil
}

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
		want, err := parseInstalledWant(setup.InstalledMD5, lewpath.New(manifestPath).Parent().String())
		if err != nil {
			return err
		}
		runner.want = want
		runner.hashed = map[string]bool{}
	}
	if len(setup.Operations) > 0 {
		if err := runner.run(ctx, setup.Operations); err != nil {
			return err
		}
	} else {
		if err := extractVolumes(ctx, Extractor{Source: e.Source, Dest: s}, vols); err != nil {
			return err
		}
	}
	if len(runner.want) > 0 {
		slog.Info("verify installed checksums", "manifest", manifestPath, "files", len(runner.want))
		if err := runner.finishHashes(ctx); err != nil {
			return err
		}
	}
	slog.Info("write reconstructed tree", "dirs", len(s.dirs), "files", len(s.files))
	for name := range sortedKeys(s.dirs) {
		if err := e.Dest.MkdirAll(name, s.dirs[name]); err != nil {
			return err
		}
	}
	names := slices.Collect(sortedKeys(s.files))
	err := withSession(ctx, func(ctx context.Context) error {
		return taskgroup.Each[string]{
			Name:     "write dest",
			PoolKind: taskgroup.IO,
			Items:    names,
			TaskName: func(_ int, name string) string { return name },
			Fn: func(ctx context.Context, st *taskgroup.Status, name string) error {
				defer st.Unit()()
				b := s.files[name]
				return writeMember(ctx, e.Dest, Member{Path: name, Size: uint64(len(b))}, bytes.NewReader(b))
			},
		}.Run(ctx)
	})
	if err != nil {
		return err
	}
	s.files = map[string][]byte{}
	return nil
}

func sortedKeys[V any](m map[string]V) iter.Seq[string] {
	return slices.Values(slices.Sorted(maps.Keys(m)))
}

func parseInstalledWant(manifest, directory string) (map[string][]byte, error) {
	want := map[string][]byte{}
	sc := bufio.NewScanner(strings.NewReader(manifest))
	for sc.Scan() {
		line := strings.TrimSuffix(sc.Text(), "\r")
		if len(line) < 35 || line[32] != ' ' || (line[33] != '*' && line[33] != ' ') {
			return nil, fmt.Errorf("installed checksum: invalid manifest line")
		}
		resolved, err := virtualPath(line[34:], lewpath.New("app", directory).String())
		if err != nil {
			return nil, fmt.Errorf("installed checksum: %w", err)
		}
		sum, err := hex.DecodeString(line[:32])
		if err != nil {
			return nil, fmt.Errorf("installed checksum: %w", err)
		}
		want[strings.TrimPrefix(resolved, "app/")] = sum
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(want) == 0 {
		return nil, fmt.Errorf("installed checksum: empty manifest")
	}
	return want, nil
}

func (s *reconstruction) verifyInstalled(ctx context.Context, manifest, directory string) error {
	want, err := parseInstalledWant(manifest, directory)
	if err != nil {
		return err
	}
	type job struct {
		name string
		want []byte
	}
	jobs := make([]job, 0, len(want))
	for name, sum := range want {
		jobs = append(jobs, job{name: name, want: sum})
	}
	err = withSession(ctx, func(ctx context.Context) error {
		return taskgroup.Each[job]{
			Name:     "verify",
			PoolKind: taskgroup.CPU,
			Items:    jobs,
			TaskName: func(_ int, j job) string { return j.name },
			Fn: func(ctx context.Context, st *taskgroup.Status, j job) error {
				defer st.Unit()()
				if err := ctx.Err(); err != nil {
					return err
				}
				b, err := s.require(j.name)
				if err != nil {
					return err
				}
				got := md5.Sum(b)
				if !bytes.Equal(got[:], j.want) {
					return fmt.Errorf("installed checksum: %s: %x want %x", j.name, got, j.want)
				}
				return nil
			},
		}.Run(ctx)
	})
	if err != nil {
		return err
	}
	slog.Info("installed hashes verified", "files", len(jobs))
	return nil
}
