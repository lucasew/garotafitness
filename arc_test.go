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
