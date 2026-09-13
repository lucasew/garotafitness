package garotafitness

import (
	"bytes"
	"encoding/hex"
	"hash/crc32"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchiveDescriptorCRC(t *testing.T) {
	// The real fg-05 footer descriptor, including its custom CRC at the end.
	raw, err := hex.DecodeString("41724301086c7a6d613a6d666274343a64316d006278a0d4e6fdaa1828fb")
	if err != nil {
		t.Fatal(err)
	}
	d, err := parseLocal(raw)
	if err != nil || d.table != fitgirlCRCTable {
		t.Fatalf("descriptor: %+v %v", d, err)
	}
	standard := bytes.Clone(raw)
	crc := crc32.ChecksumIEEE(standard[:len(standard)-4])
	for i := 0; i < 4; i++ {
		standard[len(standard)-4+i] = byte(crc >> (8 * i))
	}
	d, err = parseLocal(standard)
	if err != nil || d.table != crc32.IEEETable {
		t.Fatalf("standard descriptor: %+v %v", d, err)
	}
	for i := range raw {
		bad := bytes.Clone(raw)
		bad[i] ^= 1
		if _, err := parseLocal(bad); err == nil {
			t.Fatalf("corrupt descriptor accepted at %d", i)
		}
	}
}

func TestExtractRimWorldDecodedVolumes(t *testing.T) {
	for _, name := range []string{"fg-01.bin", "fg-02.bin", "fg-03.bin", "fg-04.bin", "fg-05.bin", "fg-06.bin"} {
		t.Run(name, func(t *testing.T) {
			if _, err := os.Stat(filepath.Join(rimworldCorpus, name)); os.IsNotExist(err) {
				t.Skip("corpus not mounted")
			}
			dst := &reconstruction{files: map[string][]byte{}, dirs: map[string]fs.FileMode{}}
			e := Extractor{Source: os.DirFS(rimworldCorpus), Dest: dst}
			if err := extractVolume(t.Context(), e, Volume{Name: name}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

const rimworldCorpus = `/media/downloads/TORRENTS/RimWorld [FitGirl Repack]`

func TestPackedRoundTripSmall(t *testing.T) {
	t.Parallel()
	// 0x06 → type 3 (DIR)
	v, n, err := readPacked([]byte{0x06}, 0)
	if err != nil || v != 3 || n != 1 {
		t.Fatalf("got %d %d %v", v, n, err)
	}
}

func TestParseRimWorldVolumes(t *testing.T) {
	if _, err := os.Stat(rimworldCorpus); err != nil {
		t.Skip("corpus not mounted")
	}
	cases := []struct {
		file   string
		member string
		method string
		last   Algo
	}{
		{"fg-01.bin", "inner.fgpack", "mpzz+srep:m3f:mem228mb", AlgoSREP},
		{"fg-04.bin", "inner.fgpack.x5", "srep:m3yf+4x4:b128mb:rzw", Algo4x4},
		{"fg-06.bin", "steam_appid.txt", "magic2", AlgoMagic2},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(rimworldCorpus, tc.file))
			if err != nil {
				t.Fatal(err)
			}
			v, err := parseVolume(tc.file, data)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, m := range v.Members {
				if strings.Contains(m.Path, tc.member) {
					found = true
					if m.Pipeline.String() != tc.method {
						t.Fatalf("pipeline %q want %q", m.Pipeline, tc.method)
					}
					if m.Pipeline.Last().Algo != tc.last {
						t.Fatalf("last %v want %v", m.Pipeline.Last().Algo, tc.last)
					}
				}
			}
			if !found {
				t.Fatalf("missing member %s in %+v", tc.member, v.Members)
			}
		})
	}
}

func TestRimWorldPipelineInventory(t *testing.T) {
	if _, err := os.Stat(rimworldCorpus); err != nil {
		t.Skip("corpus not mounted")
	}
	ents, err := os.ReadDir(rimworldCorpus)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]Algo{}
	for _, e := range ents {
		name := e.Name()
		if !strings.HasPrefix(name, "fg-") || !strings.HasSuffix(name, ".bin") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(rimworldCorpus, name))
		if err != nil {
			t.Fatal(err)
		}
		v, err := parseVolume(name, data)
		if err != nil {
			t.Fatal(name, err)
		}
		for _, m := range v.Members {
			if m.Dir || len(m.Pipeline) == 0 {
				continue
			}
			s := m.Pipeline.String()
			if _, ok := seen[s]; ok {
				continue
			}
			last := m.Pipeline.Last()
			seen[s] = last.Algo
			t.Logf("%s %s last=%s", name, s, last.Algo)
			if !last.Known() {
				t.Errorf("%s: unknown last atom %s", name, last)
			}
		}
	}
	if len(seen) == 0 {
		t.Fatal("no pipelines")
	}
}

func TestReconstructionRequiresManifest(t *testing.T) {
	if _, err := os.Stat(filepath.Join(rimworldCorpus, "fg-01.bin")); err != nil {
		t.Skip("corpus not mounted")
	}
	dst, err := OpenDirDest(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	err = extractVolume(t.Context(), Extractor{Source: os.DirFS(rimworldCorpus), Dest: dst}, Volume{Name: "fg-01.bin"})
	if err == nil || !strings.Contains(err.Error(), "checksum manifest") {
		t.Fatalf("got %v", err)
	}
	entries, err := os.ReadDir(dst.Root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatal("wrote intermediates without reconstruction")
	}
}
