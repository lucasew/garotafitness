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
	for file := range want {
		if _, ok := have[file]; ok {
			continue
		}
		flag, known := optional[file]
		if !known {
			flag = strings.Contains(strings.ToLower(file), optionalMark)
		}
		if flag {
			continue
		}
		return fmt.Errorf("checksum: missing %s", file)
	}
	return each(ctx, func(yield func(job) bool) {
		for file, sum := range want {
			v, ok := have[file]
			if !ok {
				continue
			}
			if !yield(job{file: file, sum: sum, name: v.Name}) {
				return
			}
		}
	}, func(ctx context.Context, j job) error {
		got, err := hashFile(ctx, src, j.name)
		if err != nil {
			return err
		}
		if got != j.sum {
			return fmt.Errorf("checksum: %s mismatch", j.file)
		}
		return nil
	})
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

func hashFile(ctx context.Context, src fs.FS, name string) (string, error) {
	f, err := lewpath.New(name).Open(src)
	if err != nil {
		return "", fmt.Errorf("checksum open %s: %w", name, err)
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, ctxReader{ctx, f}); err != nil {
		return "", fmt.Errorf("checksum hash %s: %w", name, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (c ctxReader) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	return c.r.Read(p)
}
