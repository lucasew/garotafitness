package rzw

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestCopyHistory(t *testing.T) {
	d := newDec(nil, 32)
	if !d.copyHistory(0xff0, 4) || !bytes.Equal(d.out, make([]byte, 4)) {
		t.Fatal("initial zero history")
	}
	d.emit('a')
	d.emit('b')
	if !d.copyHistory(2, 6) || string(d.out[4:]) != "abababab" {
		t.Fatalf("overlapping match: %q", d.out)
	}
	before := bytes.Clone(d.out)
	for _, match := range [][2]int{{0, 1}, {d.pos + 0xff1, 1}, {1, 33}} {
		if d.copyHistory(match[0], match[1]) || !bytes.Equal(d.out, before) {
			t.Fatalf("accepted invalid match %v", match)
		}
	}
}

func TestEntropyOutputLimit(t *testing.T) {
	packet, err := hex.DecodeString("b38da51816bf8d82b6ba93adbb5b30c177e4a778822f0b6353c6c67a150cd4c724377a6eeb0fe176a09c753e5d03b9ae3900")
	if err != nil {
		t.Fatal(err)
	}
	for limit := 1; limit < 75; limit++ {
		if _, err := decodeFrames([][]byte{packet}, limit); err == nil {
			t.Fatalf("accepted truncated output with limit %d", limit)
		}
	}
	got, err := decodeFrames([][]byte{packet}, 75)
	if err != nil || len(got) != 75 {
		t.Fatalf("complete packet: %d %v", len(got), err)
	}
}
