package x3

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

func TestApplyInstructions(t *testing.T) {
	tests := []struct {
		name string
		old  string
		code []byte
		want string
	}{
		{"literal gaps around source", "abcd", []byte{0x15, 0, 0x13, 2, 1, 2, 0x12, 'X', 'Y', 'Z', 0x16}, "XYbcZ"},
		{"repeated template", "abcdef", []byte{0x15, 0, 0xf, 1, 3, 0xe, 0, 0xd, 1, 0, 0x12, '!', 0x16}, "bcd!bcd"},
		{"fill pattern and zero", "", []byte{0x15, 0, 0x5, 'a', 'b', 5, 0xb, 1, 2, 0x12, '!', 0x16}, "ababa!\x00\x00"},
		{"delta with signed seeks", "abcd", []byte{0x15, 0, 0x14, 0, 4, 0x9, 1, 2, 2, 0x82, 0x11, 1, 255, 0x16}, "badd"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := applyCode(t.Context(), []byte(tt.old), tt.code, len(tt.want))
			if err != nil || string(got) != tt.want {
				t.Fatalf("got %q, %v; want %q", got, err, tt.want)
			}
			for n := 0; n < len(tt.code); n++ {
				if _, err := applyCode(t.Context(), []byte(tt.old), tt.code[:n], len(tt.want)); err == nil {
					t.Fatalf("accepted truncation at %d", n)
				}
			}
		})
	}
}

func TestApplyRejectsInvalidInstructions(t *testing.T) {
	for _, code := range [][]byte{
		{0x16}, {0x15, 1, 0x16}, {0x15, 0, 0x15, 0}, {0x15, 0, 0xff},
		{0x15, 0, 0x14, 3, 2}, {0x15, 0, 0x14, 0, 5}, {0x15, 0, 0xc, 5},
		{0x15, 0, 0xe, 0}, {0x15, 0, 0x11, 4, 1}, {0x15, 0, 0xc, 4, 0x16, 0},
	} {
		if _, err := applyCode(t.Context(), []byte("abcd"), code, 4); err == nil {
			t.Fatalf("accepted %x", code)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := applyCode(ctx, nil, []byte{0x15, 0, 0x16}, 0); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestRecordChecksBothVersions(t *testing.T) {
	old, want := []byte("abcd"), []byte("bcde")
	a, b := checksum(old)
	c, d := checksum(want)
	r := Record{Source: "a", Target: "b", old: fileVersion{size: 4, w1: a, w2: b}, new: fileVersion{size: 4, w1: c, w2: d}, code: []byte{0x15, 0, 0x14, 0, 4, 0x9, 1, 4, 0, 1, 1, 1, 0x16}}
	got, err := r.Apply(t.Context(), old)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := r.Apply(t.Context(), []byte("abce")); err == nil {
		t.Fatal("accepted wrong source")
	}
	r.new.w2 ^= 1
	if _, err := r.Apply(t.Context(), old); err == nil {
		t.Fatal("accepted wrong target checksum")
	}
}

func TestFilePath(t *testing.T) {
	for _, name := range []string{"../a", "a/../b", "/a", "C:\\a", "a\x00b"} {
		if _, err := filePath(name); err == nil {
			t.Fatalf("accepted %q", name)
		}
	}
	if got, err := filePath("Data\\file"); got != "Data/file" || err != nil {
		t.Fatalf("%q %v", got, err)
	}
}
