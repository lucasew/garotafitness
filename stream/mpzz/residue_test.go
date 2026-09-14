package mpzz

import (
	"bytes"
	"testing"
)

func TestBookCacheUnusedEntries(t *testing.T) {
	for _, tt := range []struct {
		name string
		used int
		want int
	}{
		{"compact", 1, 0},
		{"quarter occupancy", 2, 1},
		{"dense", 7, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := bookCache{capacity: 2}
			b := codebook{lengths: make([]byte, 8), quant: []int{0, 1}}
			copy(b.lengths, bytes.Repeat([]byte{3}, tt.used))
			if got := c.selectBook(&b, nil); got != 0 {
				t.Fatalf("first context %d", got)
			}
			b.lengths = bytes.Repeat([]byte{3}, 8)
			if got := c.selectBook(&b, nil); got != tt.want {
				t.Fatalf("context %d, want %d", got, tt.want)
			}
		})
	}
}
