package garotafitness

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"
)

const checksumName = "MD5/fitgirl-bins.md5"

func verifyChecksums(src fs.FS, vols []Volume) error {
	f, err := src.Open(checksumName)
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
		have[path.Base(v.Name)] = v
	}
	for file, sum := range want {
		v, ok := have[file]
		if !ok {
			if strings.Contains(strings.ToLower(file), optionalMark) {
				continue
			}
			return fmt.Errorf("checksum: missing %s", file)
		}
		got, err := hashFile(src, v.Name)
		if err != nil {
			return err
		}
		if got != sum {
			return fmt.Errorf("checksum: %s mismatch", file)
		}
	}
	return nil
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
		name = path.Base(strings.ReplaceAll(name, "\\", "/"))
		out[name] = strings.ToLower(sum)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("checksum: %w", err)
	}
	return out, nil
}

func hashFile(src fs.FS, name string) (string, error) {
	f, err := src.Open(name)
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
