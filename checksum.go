package garotafitness

import (
	"bufio"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"

	lewpath "github.com/lewtec/lewkit/x/path"
)

const checksumName = "MD5/fitgirl-bins.md5"

func verifyChecksums(ctx context.Context, src fs.FS, vols []Volume, optional map[string]bool) error {
	f, err := lewpath.New(checksumName).Open(src)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("checksum: %w", err)
	}
	defer f.Close()

	want, err := parseMD5(f)
	if err != nil {
		return err
	}
	have := make(map[string]Volume, len(vols))
	for _, v := range vols {
		have[lewpath.New(v.Name).Name()] = v
	}
	type job struct {
		file, sum, name string
	}
	jobs := make([]job, 0, len(want))
	for file, sum := range want {
		v, ok := have[file]
		if !ok {
			flag, known := optional[file]
			if !known {
				flag = strings.Contains(strings.ToLower(file), optionalMark)
			}
			if flag {
				continue
			}
			return fmt.Errorf("checksum: missing %s", file)
		}
		jobs = append(jobs, job{file: file, sum: sum, name: v.Name})
	}
	fns := make([]func(context.Context) error, len(jobs))
	for i, j := range jobs {
		fns[i] = func(context.Context) error {
			got, err := hashFile(src, j.name)
			if err != nil {
				return err
			}
			if got != j.sum {
				return fmt.Errorf("checksum: %s mismatch", j.file)
			}
			return nil
		}
	}
	return runParallel(ctx, fns)
}

func parseMD5(r io.Reader) (map[string]string, error) {
	out := make(map[string]string)
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}
		sum, rest, ok := strings.Cut(line, " ")
		if !ok || len(sum) != 32 {
			return nil, fmt.Errorf("checksum: bad line %q", line)
		}
		if _, err := hex.DecodeString(sum); err != nil {
			return nil, fmt.Errorf("checksum: bad digest %q", sum)
		}
		name := strings.TrimSpace(rest)
		name = strings.TrimPrefix(name, "*")
		name = lewpath.New(strings.ReplaceAll(name, "\\", "/")).Name()
		out[name] = strings.ToLower(sum)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("checksum: %w", err)
	}
	return out, nil
}

func hashFile(src fs.FS, name string) (string, error) {
	f, err := lewpath.New(name).Open(src)
	if err != nil {
		return "", fmt.Errorf("checksum open %s: %w", name, err)
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("checksum hash %s: %w", name, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
