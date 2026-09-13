package magic2

import (
	"bytes"
	"io"
	"testing"

	lewpath "github.com/lewtec/lewkit/x/path"
)

func TestHeader(t *testing.T) {
	for _, tt := range []struct {
		name       string
		dictionary uint32
	}{{"fg06.head", 16 << 20}, {"fg02.head", 480 << 20}} {
		t.Run(tt.name, func(t *testing.T) {
			b, err := lewpath.New(tt.name).ReadFile(testdataRoot(t))
			if err != nil {
				t.Fatal(err)
			}
			r := bytes.NewReader(b)
			h, err := ParseHeader(r)
			if err != nil {
				t.Fatal(err)
			}
			if h.DictionarySize != tt.dictionary || !h.Mixed || h.Workers != 1 || h.Independent || h.ROLZ || h.LongDistance {
				t.Fatalf("options: %+v", h)
			}
			if h.ClassShift != 4 || h.PredictionShift != 4 || h.HighShift != 0 || h.LowShift != 0 || h.WeightShift != 4 {
				t.Fatalf("contexts: %+v", h)
			}
			rest, _ := io.ReadAll(r)
			if !bytes.Equal(rest, b[9:]) {
				t.Fatal("wrong header boundary")
			}
		})
	}
}
func TestHeaderTruncated(t *testing.T) {
	for n := 0; n < 9; n++ {
		if _, err := ParseHeader(bytes.NewReader(make([]byte, n))); err == nil {
			t.Fatalf("accepted %d bytes", n)
		}
	}
}
