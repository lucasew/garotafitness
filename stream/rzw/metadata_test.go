package rzw

import (
	"encoding/hex"
	"reflect"
	"testing"
)

func TestRimWorldMetadata(t *testing.T) {
	// The complete 50-byte entropy payload from fg-05's metadata frame.
	packet, err := hex.DecodeString("b38da51816bf8d82b6ba93adbb5b30c177e4a778822f0b6353c6c67a150cd4c724377a6eeb0fe176a09c753e5d03b9ae3900")
	if err != nil {
		t.Fatal(err)
	}
	got, err := readMetadata([][]byte{packet})
	if err != nil {
		t.Fatal(err)
	}
	want := metadata{window: 1 << 20, files: []fileRecord{
		{name: "rzr3878", time: 0x31c2ad0a5, attributes: 0x10, parent: -1},
		{name: "f1", size: 0x835e1, time: 0x31c2ad0a5, crc: 0x88ecbedd, attributes: 0x20, parent: 0},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("metadata = %+v; want %+v", got, want)
	}
	for n := 0; n < len(packet); n++ {
		if _, err := readMetadata([][]byte{packet[:n]}); err == nil {
			t.Fatalf("accepted truncation at %d", n)
		}
	}
}
