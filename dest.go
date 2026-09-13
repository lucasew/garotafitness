package garotafitness

import (
	"fmt"
	"io"
	"io/fs"
	"os"

	lewpath "github.com/lewtec/lewkit/x/path"
)

// Dest is the write port for Extract. fs.FS cannot create files.
type Dest interface {
	MkdirAll(name string, perm fs.FileMode) error
	Create(name string) (io.WriteCloser, error)
}

// DirDest writes inside an [lewpath.Root]. OpenDirDest creates the OS directory
// if it is missing, then opens it so member names cannot leave the tree.
type DirDest struct {
	fs *lewpath.Root
}

func OpenDirDest(root string) (DirDest, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return DirDest{}, fmt.Errorf("dest: %w", err)
	}
	fsys, err := lewpath.Open(root)
	if err != nil {
		return DirDest{}, fmt.Errorf("dest: %w", err)
	}
	return DirDest{fs: fsys}, nil
}

func (d DirDest) Name() string {
	if d.fs == nil {
		return ""
	}
	return d.fs.Name()
}

func (d DirDest) Open(name string) (fs.File, error) {
	if d.fs == nil {
		return nil, fmt.Errorf("dest closed")
	}
	return d.fs.Open(name)
}

// SourceArg opens an existing directory as an [lewpath.Root].
type SourceArg struct {
	root *lewpath.Root
}

func (a *SourceArg) Parse(arg string) error {
	r, err := lewpath.Open(arg)
	if err != nil {
		return fmt.Errorf("source: %w", err)
	}
	if a.root != nil {
		a.root.Close()
	}
	a.root = r
	return nil
}

func (a SourceArg) Value() *lewpath.Root { return a.root }

func (a SourceArg) Close() error {
	if a.root == nil {
		return nil
	}
	return a.root.Close()
}

// DestArg creates a directory if needed and opens it as a [DirDest].
type DestArg struct {
	DirDest
}

func (a *DestArg) Parse(arg string) error {
	d, err := OpenDirDest(arg)
	if err != nil {
		return err
	}
	if a.fs != nil {
		a.fs.Close()
	}
	a.DirDest = d
	return nil
}

func (a DestArg) Value() DirDest { return a.DirDest }

func (d DirDest) Close() error {
	if d.fs == nil {
		return nil
	}
	return d.fs.Close()
}

func (d DirDest) MkdirAll(name string, perm fs.FileMode) error {
	p, err := memberName(name)
	if err != nil {
		return err
	}
	return p.MkdirAll(d.fs, perm)
}

func (d DirDest) Create(name string) (io.WriteCloser, error) {
	p, err := memberName(name)
	if err != nil {
		return nil, err
	}
	if parent := p.Parent(); parent.String() != "." {
		if err := parent.MkdirAll(d.fs, 0o755); err != nil {
			return nil, fmt.Errorf("dest mkdir: %w", err)
		}
	}
	f, err := p.Create(d.fs)
	if err != nil {
		return nil, fmt.Errorf("dest create %s: %w", name, err)
	}
	w, ok := f.(io.WriteCloser)
	if !ok {
		f.Close()
		return nil, fmt.Errorf("dest create %s: not writable", name)
	}
	return w, nil
}

func memberName(name string) (lewpath.Path, error) {
	if name == "" {
		return lewpath.Path{}, fmt.Errorf("member path empty")
	}
	p := lewpath.New(name)
	if p.IsAbs() || !p.Valid() || p.String() == "." {
		return lewpath.Path{}, fmt.Errorf("member path leaves dest: %s", name)
	}
	return p, nil
}
