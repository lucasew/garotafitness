package garotafitness

import (
	"context"
	"fmt"
	"strings"

	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/taskgroup"
)

func (p *reconstructionPlan) resetStamps() {
	p.lastWrite = map[string]taskgroup.ID{}
	p.lastUse = map[string]taskgroup.ID{}
	p.prior = nil
	p.unknown = nil
	p.schedErr = nil
}

func (p *reconstructionPlan) waitScheduled() error {
	p.pending.Wait()
	p.mu.Lock()
	err := p.schedErr
	p.mu.Unlock()
	return err
}

func (p *reconstructionPlan) failSched(err error) {
	if err == nil {
		return
	}
	p.mu.Lock()
	if p.schedErr == nil {
		p.schedErr = err
	}
	p.mu.Unlock()
}

func (p *reconstructionPlan) scheduleWords(ctx context.Context, w []string, cwd string, depth int) error {
	if len(w) == 0 {
		return nil
	}
	if depth > 32 {
		return fmt.Errorf("recursive reconstruction recipe")
	}
	name := strings.ToLower(lewpath.New(strings.ReplaceAll(w[0], "\\", "/")).Name())
	a := w[1:]
	switch name {
	case "echo", "rem", "flushfilecache.exe":
		return nil
	case "call":
		if len(a) == 0 {
			return fmt.Errorf("missing called recipe")
		}
		return p.scheduleWords(ctx, a, cwd, depth+1)
	}
	if name == "run.exe" || strings.HasSuffix(name, ".bat") || strings.HasSuffix(name, ".cmd") {
		path := w[0]
		if name == "run.exe" {
			if len(a) != 4 {
				return fmt.Errorf("unsupported command-list parameters")
			}
			path = a[2]
		} else if len(a) != 0 {
			return fmt.Errorf("parameterized batch recipes are unsupported")
		}
		resolved, err := virtualPath(path, cwd)
		if err != nil {
			return err
		}
		if b, err := p.read(resolved); err == nil {
			return p.recipeLines(ctx, string(b), cwd, depth+1)
		}
		return p.scheduleLeaf(ctx, name, w, cwd, depth, []string{resolved}, nil, true, taskgroup.Control)
	}
	reads, writes, glob, err := recipeFiles(name, w, cwd)
	if err != nil {
		return err
	}
	return p.scheduleLeaf(ctx, name, w, cwd, depth, reads, writes, glob, taskgroup.CPU)
}

func leafLabel(prog string, reads, writes []string) string {
	focus := writes
	if len(focus) == 0 {
		focus = reads
	}
	if len(focus) == 0 {
		return prog
	}
	return prog + " " + strings.Join(focus, " ")
}

func (p *reconstructionPlan) scheduleLeaf(ctx context.Context, name string, w []string, cwd string, depth int, reads, writes []string, glob bool, pool taskgroup.PoolKind) error {
	deps := p.fileDeps(reads, writes, glob)
	label := leafLabel(name, reads, writes)
	p.pending.Add(1)
	id := taskgroup.Go(ctx, label, pool, func(ctx context.Context, s *taskgroup.Status) error {
		defer p.pending.Done()
		var err error
		if pool == taskgroup.Control {
			err = p.expandRecipe(ctx, w, cwd, depth)
		} else {
			err = p.words(ctx, w, cwd, depth)
		}
		p.failSched(err)
		return err
	}, deps...)
	p.recordLeaf(id, reads, writes, glob)
	return nil
}

func (p *reconstructionPlan) expandRecipe(ctx context.Context, w []string, cwd string, depth int) error {
	name := strings.ToLower(lewpath.New(strings.ReplaceAll(w[0], "\\", "/")).Name())
	path := w[0]
	if name == "run.exe" {
		path = w[3]
	}
	resolved, err := virtualPath(path, cwd)
	if err != nil {
		return err
	}
	b, err := p.read(resolved)
	if err != nil {
		return err
	}
	return p.recipeLines(ctx, string(b), cwd, depth+1)
}

func (p *reconstructionPlan) fileDeps(reads, writes []string, glob bool) []taskgroup.ID {
	if glob {
		return append(append([]taskgroup.ID{}, p.prior...), p.unknown...)
	}
	seen := map[taskgroup.ID]bool{}
	var deps []taskgroup.ID
	add := func(id taskgroup.ID) {
		if id != 0 && !seen[id] {
			seen[id] = true
			deps = append(deps, id)
		}
	}
	for _, path := range reads {
		add(p.lastWrite[path])
	}
	for _, path := range writes {
		add(p.lastWrite[path])
		add(p.lastUse[path])
	}
	for _, id := range p.unknown {
		add(id)
	}
	return deps
}

func (p *reconstructionPlan) recordLeaf(id taskgroup.ID, reads, writes []string, glob bool) {
	if p.lastWrite == nil {
		p.lastWrite = map[string]taskgroup.ID{}
		p.lastUse = map[string]taskgroup.ID{}
	}
	p.prior = append(p.prior, id)
	if glob {
		p.unknown = append(p.unknown, id)
		return
	}
	for _, path := range reads {
		p.lastUse[path] = id
	}
	for _, path := range writes {
		p.lastWrite[path] = id
		delete(p.lastUse, path)
	}
}

func recipeFiles(name string, w []string, cwd string) (reads, writes []string, glob bool, err error) {
	resolve := func(s string) (string, error) { return virtualPath(s, cwd) }
	add := func(s string, write bool) error {
		if strings.ContainsAny(s, "*?[") {
			glob = true
		}
		n, e := resolve(s)
		if e != nil {
			return e
		}
		if write {
			writes = append(writes, n)
		} else {
			reads = append(reads, n)
		}
		return nil
	}
	a := w[1:]
	stripFlags := func(args []string) []string {
		out := []string{}
		for _, s := range args {
			if !strings.HasPrefix(s, "/") {
				out = append(out, s)
			}
		}
		return out
	}
	switch name {
	case "del", "erase", "rd", "rmdir":
		if name == "rd" || name == "rmdir" {
			glob = true
		}
		for _, pattern := range stripFlags(a) {
			if err = add(pattern, true); err != nil {
				return
			}
		}
	case "move", "ren", "rename":
		a = stripFlags(a)
		if len(a) != 2 {
			return
		}
		if err = add(a[0], false); err != nil {
			return
		}
		if err = add(a[0], true); err != nil {
			return
		}
		if err = add(a[1], true); err != nil {
			return
		}
	case "copy":
		a = stripFlags(a)
		if len(a) < 1 || len(a) > 2 {
			return
		}
		parts := strings.Split(a[0], "+")
		dest := parts[0]
		if len(a) == 2 {
			dest = a[1]
		}
		for _, pattern := range parts {
			if err = add(pattern, false); err != nil {
				return
			}
		}
		err = add(dest, true)
	case "fart.exe":
		if len(a) == 4 {
			err = add(a[1], true)
			reads = append(reads, writes...)
		}
	case "fsb.exe":
		if len(a) == 2 {
			if err = add(a[0], false); err != nil {
				return
			}
			err = add(a[1], true)
		}
	case "x2.exe":
		if len(a) == 2 {
			if err = add(a[0], false); err != nil {
				return
			}
			if err = add(a[1], false); err != nil {
				return
			}
			err = add(a[0], true)
		}
	case "x5.exe", "hpatchz.exe":
		if len(a) > 0 && strings.HasPrefix(a[0], "-s-") {
			a = a[1:]
		}
		if len(a) == 3 {
			if err = add(a[0], false); err != nil {
				return
			}
			if err = add(a[1], false); err != nil {
				return
			}
			err = add(a[2], true)
		}
	case "fgpack.exe":
		_, source, dest, e := packingOptions(a)
		if e != nil {
			return
		}
		if err = add(source, false); err != nil {
			return
		}
		err = add(dest, true)
	case "x.exe", "xdelta.exe", "xdelta3.exe":
		glob = true
	case "x3.exe":
		if len(a) == 1 {
			if err = add(a[0], false); err != nil {
				return
			}
		}
		glob = true
	default:
		if strings.HasSuffix(name, ".bat") || strings.HasSuffix(name, ".cmd") {
			err = add(w[0], false)
			glob = true
		}
	}
	return
}
