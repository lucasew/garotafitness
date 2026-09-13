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

	"github.com/lucasew/garotafitness/fgpack"
	"github.com/lucasew/garotafitness/fsb"
	"github.com/lucasew/garotafitness/x2"
	"github.com/lucasew/garotafitness/x3"
	"github.com/lucasew/garotafitness/x5"
	"github.com/lucasew/garotafitness/xdelta"
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
		return nil, fmt.Errorf("reconstruction: missing %s", name)
	}
	return b, nil
}

func (e Extractor) extractReconstructed(ctx context.Context, vols []Volume, manifest string) error {
	s := &reconstruction{files: make(map[string][]byte), dirs: make(map[string]fs.FileMode)}
	staged := Extractor{Source: e.Source, Dest: s}
	for _, v := range vols {
		slog.Info("extract volume", "name", v.Name)
		if err := extractVolume(ctx, staged, v); err != nil {
			return err
		}
	}
	if _, ok := s.files["inner.fgpack"]; ok {
		if err := s.reconstructInner(ctx); err != nil {
			return err
		}
		if err := s.reconstructBundles(ctx); err != nil {
			return err
		}
		if err := s.applyUpdate(ctx); err != nil {
			return err
		}
	}
	if err := s.verifyInstalled(ctx, manifest); err != nil {
		return err
	}
	s.files["_Redist/fitgirl.md5"] = []byte(manifest)
	for _, name := range sortedKeys(s.dirs) {
		if scratchMember(name) {
			continue
		}
		if err := e.Dest.MkdirAll(name, s.dirs[name]); err != nil {
			return err
		}
	}
	for _, name := range sortedKeys(s.files) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if scratchMember(name) {
			continue
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

func (s *reconstruction) reconstructInner(ctx context.Context) error {
	slog.Info("reconstruct inner archive")
	old, err := s.require("inner.fgpack")
	if err != nil {
		return err
	}
	diff, err := s.require("inner.fgpack.x5")
	if err != nil {
		return err
	}
	var remuxed bytes.Buffer
	if err := fsb.Remux(ctx, &remuxed, bytes.NewReader(old)); err != nil {
		return err
	}
	delete(s.files, "inner.fgpack")
	inner, err := x5.Apply(ctx, remuxed.Bytes(), diff)
	if err != nil {
		return err
	}
	delete(s.files, "inner.fgpack.x5")
	return extractVolumeData(ctx, Extractor{Dest: s}, "inner.fgpack", inner)
}

var bundlePaths = []string{
	"Data/Anomaly/AssetBundles/resources_anomaly",
	"Data/Biotech/AssetBundles/resources_biotech",
	"Data/Ideology/AssetBundles/resources_ideology",
	"Data/Royalty/AssetBundles/resources_royalty",
}

func (s *reconstruction) reconstructBundles(ctx context.Context) error {
	for i, dest := range bundlePaths {
		slog.Info("reconstruct bundle", "name", dest)
		base := fmt.Sprintf("temp/%d", i+1)
		old, err := s.require(base + ".fgu")
		if err != nil {
			return err
		}
		diff, err := s.require(base + ".fgu.x2")
		if err != nil {
			return err
		}
		patched, err := x2.Apply(old, diff)
		if err != nil {
			return fmt.Errorf("%s: %w", dest, err)
		}
		delete(s.files, base+".fgu")
		delete(s.files, base+".fgu.x2")
		packed, err := fgpack.Encode(ctx, patched)
		if err != nil {
			return fmt.Errorf("%s: %w", dest, err)
		}
		diff, err = s.require(base + ".bundle.x")
		if err != nil {
			return err
		}
		bundle, err := xdelta.Apply(ctx, packed, diff)
		if err != nil {
			return fmt.Errorf("%s: %w", dest, err)
		}
		s.files[dest] = bundle
		delete(s.files, base+".bundle.x")
	}
	return nil
}

func (s *reconstruction) applyUpdate(ctx context.Context) error {
	diff, err := s.require("rimworld.x3")
	if err != nil {
		return err
	}
	records, err := x3.Parse(diff)
	if err != nil {
		return err
	}
	slog.Info("apply update", "files", len(records))
	for _, rec := range records {
		old, err := s.require(rec.Source)
		if err != nil {
			return err
		}
		out, err := rec.Apply(ctx, old)
		if err != nil {
			return err
		}
		delete(s.files, rec.Source)
		s.files[rec.Target] = out
	}
	delete(s.files, "rimworld.x3")
	return nil
}

func (s *reconstruction) verifyInstalled(ctx context.Context, manifest string) error {
	sc := bufio.NewScanner(strings.NewReader(manifest))
	count := 0
	for sc.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		line := strings.TrimSuffix(sc.Text(), "\r")
		if len(line) < 37 || line[32:37] != " *..\\" {
			return fmt.Errorf("installed checksum: invalid manifest line")
		}
		name := strings.ReplaceAll(line[37:], "\\", "/")
		if !fs.ValidPath(name) || strings.ContainsAny(name, ":\x00") {
			return fmt.Errorf("installed checksum: invalid path %q", name)
		}
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

func scratchMember(name string) bool {
	root, _, _ := strings.Cut(name, "/")
	if root == "temp" || root == "work" || root == "mover" {
		return true
	}
	if strings.HasPrefix(path.Base(name), "goggame-") && strings.HasSuffix(name, ".info") {
		return true
	}
	switch name {
	case "BorderlessFullscreen.bat", "How to install Anomaly.txt", "How to install Biotech.txt", "How to install Ideology.txt", "How to install Royalty.txt":
		return true
	}
	return false
}
