package x2

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestApply(t *testing.T) {
	patch := binary.LittleEndian.AppendUint64(nil, 260)
	patch = binary.LittleEndian.AppendUint64(patch, 3)
	patch = append(patch, 0) // 256 bytes
	patch = append(patch, bytes.Repeat([]byte{42}, 256)...)
	patch = binary.LittleEndian.AppendUint64(patch, 4)
	patch = append(patch, 1, 17) // overlapping replacement wins
	old := []byte{1, 2, 3, 4, 5}
	out, err := Apply(old, patch)
	if err != nil {
		t.Fatal(err)
	}
	want := append([]byte{1, 2, 3}, bytes.Repeat([]byte{42}, 256)...)
	want[4] = 17
	want = append(want, 0)
	if !bytes.Equal(out, want) || old[4] != 5 {
		t.Fatal("incorrect sparse patch result")
	}
	for _, n := range []int{0, 7, 9, 16, 17, len(patch) - 1} {
		if _, err := Apply(old, patch[:n]); err == nil {
			t.Fatalf("accepted truncated patch at %d", n)
		}
	}
	binary.LittleEndian.PutUint64(patch[8:], 259)
	if _, err := Apply(old, patch); err == nil {
		t.Fatal("accepted out of bounds write")
	}
	out, err = Apply(old, binary.LittleEndian.AppendUint64(nil, 2))
	if err != nil || !bytes.Equal(out, []byte{1, 2}) {
		t.Fatal("truncate", out, err)
	}
}
