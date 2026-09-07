package garotafitness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
