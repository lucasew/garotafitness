package garotafitness

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"slices"
	"strings"

	"github.com/lucasew/garotafitness/setupdata"
)

type reconstructionPlan struct {
	source    fs.FS
	app, temp *reconstruction
	volumes   map[string]Volume
	remaining map[string]int
	decoded   map[string]*reconstruction
	seen      map[string]bool
}

func newStaging() *reconstruction {
	return &reconstruction{files: map[string][]byte{}, dirs: map[string]fs.FileMode{}}
}

// virtualPath resolves installer paths inside three disjoint namespaces. Source
// paths can only be read; temporary paths never become installed files.
func virtualPath(name, cwd string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	lower := strings.ToLower(name)
	expectedRoot := ""
	for _, root := range []string{"app", "tmp", "src"} {
		prefix := "{" + root + "}"
		if lower == prefix || strings.HasPrefix(lower, prefix+"/") {
			expectedRoot = root
			name = root + "/" + strings.TrimLeft(name[len(prefix):], "/")
			cwd = ""
			break
		}
	}
	if name == "" {
		name = "."
	}
	if strings.ContainsAny(name, ":\x00{}%") || strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("reconstruction: invalid path %q", name)
	}
	if cwd != "" {
		name = path.Join(cwd, name)
	} else {
		name = path.Clean(name)
	}
	root, _, _ := strings.Cut(name, "/")
	if expectedRoot != "" && root != expectedRoot {
		return "", fmt.Errorf("reconstruction: path leaves %s namespace", expectedRoot)
	}
	if root != "app" && root != "tmp" && root != "src" {
		return "", fmt.Errorf("reconstruction: path leaves installation: %q", name)
	}
	if cwd != "" {
		base, _, _ := strings.Cut(cwd, "/")
		if root != base {
			return "", fmt.Errorf("reconstruction: path changes namespace: %q", name)
		}
	}
	if !fs.ValidPath(name) {
		return "", fmt.Errorf("reconstruction: invalid path %q", name)
	}
	return name, nil
}

func (p *reconstructionPlan) store(name string) (*reconstruction, string, error) {
	root, rel, _ := strings.Cut(name, "/")
	switch root {
	case "app":
		return p.app, rel, nil
	case "tmp":
		return p.temp, rel, nil
	default:
		return nil, "", fmt.Errorf("reconstruction: cannot write %s", name)
	}
}
func (p *reconstructionPlan) read(name string) ([]byte, error) {
	if strings.HasPrefix(name, "src/") {
		return fs.ReadFile(p.source, strings.TrimPrefix(name, "src/"))
	}
	s, rel, err := p.store(name)
	if err != nil {
		return nil, err
	}
	return s.require(rel)
}
func (p *reconstructionPlan) put(name string, b []byte) error {
	s, rel, err := p.store(name)
	if err != nil {
		return err
	}
	if rel == "" || !fs.ValidPath(rel) {
		return fmt.Errorf("reconstruction: invalid file %s", name)
	}
	for existing := range s.files {
		if strings.EqualFold(existing, rel) {
			delete(s.files, existing)
		}
	}
	s.files[rel] = b
	return nil
}
func (p *reconstructionPlan) matches(pattern string, dirs bool) ([]string, error) {
	s, rel, err := p.store(pattern)
	if err != nil {
		return nil, err
	}
	root, _, _ := strings.Cut(pattern, "/")
	var out []string
	for name := range s.files {
		ok, err := path.Match(strings.ToLower(rel), strings.ToLower(name))
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, root+"/"+name)
		}
	}
	if dirs {
		for name := range s.dirs {
			ok, err := path.Match(strings.ToLower(rel), strings.ToLower(name))
			if err != nil {
				return nil, err
			}
			if ok {
				out = append(out, root+"/"+name)
			}
		}
	}
	slices.Sort(out)
	return out, nil
}
func (p *reconstructionPlan) remove(name string, tree bool) error {
	s, rel, err := p.store(name)
	if err != nil {
		return err
	}
	if rel == "" {
		return fmt.Errorf("reconstruction: cannot remove namespace root")
	}
	for n := range s.files {
		if strings.EqualFold(n, rel) || (tree && strings.HasPrefix(strings.ToLower(n), strings.ToLower(rel)+"/")) {
			delete(s.files, n)
		}
	}
	for n := range s.dirs {
		if strings.EqualFold(n, rel) || (tree && strings.HasPrefix(strings.ToLower(n), strings.ToLower(rel)+"/")) {
			delete(s.dirs, n)
		}
	}
	return nil
}

func (p *reconstructionPlan) extract(ctx context.Context, op setupdata.Operation) error {
	source, err := virtualPath(op.Source, "")
	if err != nil {
		return err
	}
	dest, err := virtualPath(op.Dest, "")
	if err != nil {
		return err
	}
	store, prefix, err := p.store(dest)
	if err != nil {
		return err
	}
	var staged *reconstruction
	if strings.HasPrefix(source, "src/") {
		name := strings.TrimPrefix(source, "src/")
		v, ok := p.volumes[name]
		if !ok {
			if op.Optional || strings.Contains(strings.ToLower(name), optionalMark) {
				return nil
			}
			return fmt.Errorf("reconstruction: missing required volume %s", name)
		}
		staged = p.decoded[name]
		if staged == nil {
			slog.Info("extract volume", "name", name)
			staged = newStaging()
			if err := extractVolume(ctx, Extractor{Source: p.source, Dest: staged}, v); err != nil {
				return err
			}
			p.decoded[name] = staged
			p.seen[name] = true
		}
		p.remaining[name]--
		if p.remaining[name] <= 0 {
			delete(p.decoded, name)
		}
	} else {
		data, err := p.read(source)
		if err != nil {
			return err
		}
		slog.Info("extract reconstructed archive", "name", source)
		staged = newStaging()
		if err := extractVolumeData(ctx, Extractor{Dest: staged}, source, data); err != nil {
			return err
		}
	}
	filter := strings.Trim(strings.ReplaceAll(op.Filter, "\\", "/"), "/")
	if filter != "" && !fs.ValidPath(filter) {
		return fmt.Errorf("reconstruction: invalid archive filter %q", op.Filter)
	}
	mapName := func(name string) (string, bool) {
		if filter != "" {
			if !strings.HasPrefix(strings.ToLower(name), strings.ToLower(filter)+"/") {
				return "", false
			}
			name = name[len(filter)+1:]
		}
		return path.Join(prefix, name), true
	}
	for name, mode := range staged.dirs {
		if mapped, ok := mapName(name); ok {
			if err := store.MkdirAll(mapped, mode); err != nil {
				return err
			}
		}
	}
	for name, b := range staged.files {
		if mapped, ok := mapName(name); ok {
			if _, err := memberPath(".", mapped); err != nil {
				return err
			}
			store.files[mapped] = b
		}
	}
	return nil
}

func (p *reconstructionPlan) run(ctx context.Context, ops []setupdata.Operation) error {
	for _, op := range ops {
		if op.Kind == "extract" {
			source, err := virtualPath(op.Source, "")
			if err != nil {
				return err
			}
			if strings.HasPrefix(source, "src/") {
				p.remaining[strings.TrimPrefix(source, "src/")]++
			}
		}
	}
	for i, op := range ops {
		if err := ctx.Err(); err != nil {
			return err
		}
		slog.Info("reconstruction operation", "index", i+1, "kind", op.Kind, "source", op.Source, "dest", op.Dest, "program", op.Program, "workdir", op.WorkDir)
		var err error
		switch op.Kind {
		case "extract":
			err = p.extract(ctx, op)
		case "command":
			var cwd string
			cwd, err = virtualPath(op.WorkDir, "")
			if err == nil {
				err = p.command(ctx, op.Program, op.Args, cwd, 0)
			}
		case "remove":
			var name string
			name, err = virtualPath(op.Source, "")
			if err == nil {
				err = p.remove(name, true)
			}
		default:
			err = fmt.Errorf("unknown reconstruction operation %q", op.Kind)
		}
		if err != nil {
			return fmt.Errorf("setup reconstruction record %d: %w", i+1, err)
		}
	}
	for name, v := range p.volumes {
		if !v.Optional && !p.seen[name] {
			return fmt.Errorf("setup reconstruction did not extract required volume %s", name)
		}
	}
	return nil
}
