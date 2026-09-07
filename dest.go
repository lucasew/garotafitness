package garotafitness

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Dest is the write port for Extract. fs.FS cannot create files.
type Dest interface {
	MkdirAll(name string, perm fs.FileMode) error
	Create(name string) (io.WriteCloser, error)
}

// DirDest writes under Root. Create Dest if missing happens in OpenDirDest.
type DirDest struct {
	Root string
}

func OpenDirDest(root string) (DirDest, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return DirDest{}, fmt.Errorf("dest: %w", err)
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return DirDest{}, fmt.Errorf("dest abs: %w", err)
	}
	return DirDest{Root: abs}, nil
}

func (d DirDest) MkdirAll(name string, perm fs.FileMode) error {
	full, err := memberPath(d.Root, name)
	if err != nil {
		return err
	}
	return os.MkdirAll(full, perm)
}

func (d DirDest) Create(name string) (io.WriteCloser, error) {
	full, err := memberPath(d.Root, name)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return nil, fmt.Errorf("dest mkdir: %w", err)
	}
	f, err := os.Create(full)
	if err != nil {
		return nil, fmt.Errorf("dest create %s: %w", name, err)
	}
	return f, nil
}

func memberPath(root, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("member path empty")
	}
	rel := filepath.Clean(name)
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("member path leaves dest: %s", name)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("member path leaves dest: %s", name)
	}
	return filepath.Join(root, rel), nil
}
