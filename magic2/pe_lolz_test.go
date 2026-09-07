package magic2

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"testing"

	ulzma "github.com/ulikunitz/xz/lzma"
)

func TestTablesSelfConsistent(t *testing.T) {
	t.Parallel()
	if ransL != 1<<23 {
		t.Fatalf("ransL=%d", ransL)
	}
	if fcmNibbleBytes != 0x3420000 || fcmBinaryBytes != 0x992200 {
		t.Fatalf("fcm sizes %#x %#x", fcmNibbleBytes, fcmBinaryBytes)
	}
	if !bytes.Contains(versionBanner, []byte("v22c4b")) {
		t.Fatal("version banner")
	}
	if len(optionTable32) != 32 || len(optionDefaults16) != 16 {
		t.Fatal("option layout")
	}
	if binary.LittleEndian.Uint32(optionDefaults16[0:]) != 2 {
		t.Fatal("pc default")
	}
}

func TestCorpusImages(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(rimworldSetup)
	if err != nil {
		t.Skip("corpus not mounted")
	}
	plain := firstZLB(t, raw)
	if !bytes.Contains(plain, versionBanner) {
		t.Fatal("missing v22c4b banner in zlb@0x22600")
	}
	if !bytes.Contains(plain, lollyBanner) {
		t.Fatal("missing v20d3 banner")
	}
	for _, s := range allocNames {
		if !bytes.Contains(plain, s) {
			t.Fatalf("missing %q", s)
		}
	}
	i := bytes.Index(plain, optionHelpPrefix)
	if i < 64 {
		t.Fatal("option help")
	}
	if !bytes.Equal(plain[i-64:i-32], optionTable32) {
		t.Fatalf("optionTable32 mismatch: %x", plain[i-64:i-32])
	}
	if !bytes.Equal(plain[i-32:i-16], optionTable32[:16]) {
		t.Fatalf("option row repeat mismatch: %x", plain[i-32:i-16])
	}
	if !bytes.Equal(plain[i-16:i], optionDefaults16) {
		t.Fatalf("optionDefaults16 mismatch: %x", plain[i-16:i])
	}
	nibble := make([]byte, 4)
	binary.LittleEndian.PutUint32(nibble, fcmNibbleBytes)
	bin := make([]byte, 4)
	binary.LittleEndian.PutUint32(bin, fcmBinaryBytes)
	if !bytes.Contains(plain, nibble) || !bytes.Contains(plain, bin) {
		t.Fatal("missing FCM alloc immediates")
	}

	innos := findInno(raw)
	if len(innos) == 0 {
		t.Fatal("no inno block")
	}
	for _, name := range []string{"cls-magic2_x64.exe", "cls-magic2l_x64.exe", "cls-lollypop_x64.exe"} {
		if !bytes.Contains(innos[0].plain, []byte(name)) {
			t.Fatalf("inno missing %s", name)
		}
	}
}

func firstZLB(t *testing.T, raw []byte) []byte {
	t.Helper()
	i := bytes.Index(raw, []byte(zlbMagic))
	if i < 0 {
		t.Fatal("no zlb")
	}
	plain, err := decodeLZMA2(raw[i+len(zlbMagic):])
	if err != nil {
		t.Fatal(err)
	}
	return plain
}

const rimworldSetup = `/media/downloads/TORRENTS/RimWorld [FitGirl Repack]/setup.exe`

const (
	zlbMagic     = "zlb\x1a"
	innoDataID   = "Inno Setup Setup Data ("
	innoIDLen    = 64
	chunkCRCSize = 4096
	maxStored    = 16 << 20
	maxPlain     = 32 << 20
	maxDict      = 128 << 20
)

func decodeLZMA2(payload []byte) ([]byte, error) {
	if len(payload) < 2 {
		return nil, io.ErrUnexpectedEOF
	}
	dict := lzma2Dict(payload[0])
	if dict <= 0 || dict > maxDict {
		return nil, errors.New("lzma2 dict")
	}
	r, err := ulzma.Reader2Config{DictCap: dict}.NewReader2(bytes.NewReader(payload[1:]))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(io.LimitReader(r, int64(maxPlain)))
}

func lzma2Dict(prop byte) int {
	if prop > 40 {
		return 0
	}
	if prop == 40 {
		return maxDict
	}
	return (2 | int(prop&1)) << (int(prop)/2 + 11)
}

type innoBlob struct {
	off   int
	plain []byte
}

func findInno(data []byte) []innoBlob {
	var out []innoBlob
	start := 0
	for {
		i := bytes.Index(data[start:], []byte(innoDataID))
		if i < 0 {
			return out
		}
		i += start
		if i+innoIDLen > len(data) {
			start = i + 1
			continue
		}
		pos := i + innoIDLen
		for n := 0; n < 2; n++ {
			plain, next, ok := readInnoBlock(data, pos)
			if !ok {
				break
			}
			out = append(out, innoBlob{off: pos, plain: plain})
			pos = next
		}
		start = i + 1
	}
}

func readInnoBlock(data []byte, pos int) ([]byte, int, bool) {
	if pos+9 > len(data) {
		return nil, pos, false
	}
	want := binary.LittleEndian.Uint32(data[pos:])
	stored := binary.LittleEndian.Uint32(data[pos+4:])
	if crc32.ChecksumIEEE(data[pos+4:pos+9]) != want {
		return nil, pos, false
	}
	if stored == 0 || stored > maxStored || pos+9+int(stored) > len(data) {
		return nil, pos, false
	}
	next := pos + 9 + int(stored)
	raw := stripChunkCRC(data[pos+9 : next])
	if data[pos+8] == 0 {
		return raw, next, true
	}
	plain, err := decodeLZMA1(raw)
	if err != nil {
		return nil, pos, false
	}
	return plain, next, true
}

func stripChunkCRC(stored []byte) []byte {
	out := make([]byte, 0, len(stored))
	for len(stored) >= 4 {
		stored = stored[4:]
		n := chunkCRCSize
		if n > len(stored) {
			n = len(stored)
		}
		out = append(out, stored[:n]...)
		stored = stored[n:]
	}
	return out
}

func decodeLZMA1(payload []byte) ([]byte, error) {
	if len(payload) < 5 {
		return nil, io.ErrUnexpectedEOF
	}
	dict := binary.LittleEndian.Uint32(payload[1:5])
	if dict == 0 || int(dict) > maxDict {
		return nil, errors.New("lzma1 dict")
	}
	hdr := make([]byte, 13)
	copy(hdr[:5], payload[:5])
	for i := 5; i < 13; i++ {
		hdr[i] = 0xff
	}
	r, err := ulzma.ReaderConfig{DictCap: int(dict)}.NewReader(
		io.MultiReader(bytes.NewReader(hdr), bytes.NewReader(payload[5:])),
	)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(io.LimitReader(r, int64(maxPlain)))
}
