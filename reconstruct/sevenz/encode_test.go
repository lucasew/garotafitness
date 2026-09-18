package sevenz

import (
	"bytes"
	"testing"
)

func TestEncodeMagic(t *testing.T) {
	t.Parallel()
	out, err := Encode(t.Context(), []File{{Name: "a.txt", Data: []byte("hello sevenz")}})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) < 32 || !bytes.Equal(out[:6], []byte{0x37, 0x7a, 0xbc, 0xaf, 0x27, 0x1c}) {
		t.Fatalf("magic %x", out[:min(6, len(out))])
	}
}
func TestEncodeEmpty(t *testing.T) {
	t.Parallel()
	if _, err := Encode(t.Context(), nil); err == nil {
		t.Fatal("accepted empty")
	}
}
