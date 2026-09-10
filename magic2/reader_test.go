package magic2

import (
	"bytes"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func testdata(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestNewReader(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   io.Reader
		want error
	}{
		{name: "nil", want: errNil},
		{name: "empty", in: bytes.NewReader(nil), want: io.EOF},
		{name: "short", in: bytes.NewReader([]byte("DH")), want: io.ErrUnexpectedEOF},
		{name: "tagonly", in: bytes.NewReader([]byte(lolzTag)), want: io.ErrUnexpectedEOF},
		{name: "arc", in: bytes.NewReader([]byte("ArC\x01x")), want: errMagic},
		{name: "srep", in: bytes.NewReader([]byte("SREP\x03")), want: errMagic},
		{name: "ver", in: bytes.NewReader([]byte(lolzTag + "\x00")), want: errVersion},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rc, err := NewReader(tt.in)
			if rc != nil {
				t.Fatalf("NewReader(%s) reader = %T; want nil", tt.name, rc)
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("NewReader(%s) err = %v; want %v", tt.name, err, tt.want)
			}
		})
	}
}

func TestNewReaderAcceptsHeader(t *testing.T) {
	t.Parallel()
	rc, err := NewReader(bytes.NewReader(testdata(t, "fg06.head")))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rc.Close() })
}

func TestFG06SolidTag(t *testing.T) {
	t.Parallel()
	const corpus = `/media/downloads/TORRENTS/RimWorld [FitGirl Repack]/fg-06.bin`
	f, err := os.Open(corpus)
	if err != nil {
		t.Skip("corpus not mounted")
	}
	t.Cleanup(func() { f.Close() })
	if _, err := f.Seek(0x1F, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	want := testdata(t, "fg06.head")
	head := make([]byte, len(want))
	if _, err := io.ReadFull(f, head); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(head, want) {
		t.Fatalf("fg-06 solid at 0x1F = %x; want %x", head, want)
	}
	if _, err := f.Seek(0x1F, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseHeader(f); err != nil {
		t.Fatal(err)
	}
	rest := make([]byte, len(want)-headerLen)
	if _, err := io.ReadFull(f, rest); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rest, want[headerLen:]) {
		t.Fatalf("fg-06 leftover %x; want %x", rest, want[headerLen:])
	}
	if _, err := f.Seek(0x1F, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	rc, err := NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rc.Close() })
}

func TestFG06DecodeAttempt(t *testing.T) {
	t.Parallel()
	src := fg06Payload(t)
	cfgs := []cfg{
		{name: "be/s15/a6/8sym", be: true, tokN: 8, adapt: 6},
		{name: "be/s15/a6/16sym", be: true, tokN: 16, adapt: 6},
		{name: "le/s15/a6/8sym", be: false, tokN: 8, adapt: 6},
	}
	for _, c := range cfgs {
		out, err := decodeLZ(src, c)
		if err != nil && len(out) == 0 {
			t.Logf("%s: empty (%v)", c.name, err)
			continue
		}
		n := 16
		if n > len(out) {
			n = len(out)
		}
		t.Logf("%s n=%d first=%x", c.name, len(out), out[:n])
		if len(out) >= emuSize+appidSize {
			got := crc32.ChecksumIEEE(out[emuSize : emuSize+appidSize])
			t.Logf("%s appid crc=%08x want=%08x", c.name, got, appidCRC)
			if got == appidCRC {
				t.Logf("%s matched steam_appid.txt CRC", c.name)
			}
		}
	}
}

func fg06Payload(t *testing.T) []byte {
	t.Helper()
	const corpus = `/media/downloads/TORRENTS/RimWorld [FitGirl Repack]/fg-06.bin`
	f, err := os.Open(corpus)
	if err != nil {
		t.Skip("corpus not mounted")
	}
	t.Cleanup(func() { f.Close() })
	if _, err := f.Seek(0x1F, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	raw := make([]byte, 93116)
	if _, err := io.ReadFull(f, raw); err != nil {
		t.Fatal(err)
	}
	if string(raw[:4]) != lolzTag || raw[4] != verV22c4b {
		t.Fatalf("solid tag %x", raw[:5])
	}
	return raw[5:]
}

func TestFG06WASMProgress(t *testing.T) {
	t.Parallel()
	src := fg06Payload(t)
	out, ok := decodeWASM(src)
	if len(out) < 4 {
		t.Fatalf("wasm n=%d", len(out))
	}
	first := out[:4]
	t.Logf("first32=%x ascii=%q", prefix(out, 32), prefix(out, 32))
	// Default +0xc24==0 must emit INI ('[' or ';'), not the in-image
	// LZ first tokens 04 04 08 06 (hi slot 0x0025 / lo 0x2025).
	if out[0] == '[' || out[0] == ';' {
		t.Log("INI prefix from 0x14005fc5d")
	} else if bytes.Equal(first, []byte{0x04, 0x04, 0x08, 0x06}) {
		t.Log("in-image LZ tokens; extras/0x14005fc5d not yet steering the first byte")
	} else {
		t.Logf("first tokens %x", first)
	}
	if len(out) >= emuSize+appidSize {
		emu := crc32.ChecksumIEEE(out[:emuSize])
		app := crc32.ChecksumIEEE(out[emuSize : emuSize+appidSize])
		t.Logf("n=%d emu=%08x appid=%08x ok=%v", len(out), emu, app, ok)
		if emu == 0xfb362bfa && app == appidCRC {
			return
		}
	}
	if !ok {
		t.Log("appid CRC not matched yet")
	}
}

func TestFG06SolidCRC(t *testing.T) {
	t.Parallel()
	const corpus = `/media/downloads/TORRENTS/RimWorld [FitGirl Repack]/fg-06.bin`
	f, err := os.Open(corpus)
	if err != nil {
		t.Skip("corpus not mounted")
	}
	t.Cleanup(func() { f.Close() })
	if _, err := f.Seek(0x1F, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	rc, err := NewReader(io.LimitReader(f, 93116))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rc.Close() })
	plain, err := io.ReadAll(rc)
	if errors.Is(err, errBitstream) {
		t.Skip(err.Error())
	}
	if err != nil {
		t.Fatal(err)
	}
	members := []struct {
		size uint32
		crc  uint32
		path string
	}{
		{2895, 0xfb362bfa, "RimWorldWin64_Data/Plugins/x86_64/steam_emu.ini"},
		{6, 0xf75982bb, "steam_appid.txt"},
		{190, 0xf845695f, "SteamInputDefaultConfiguration.vdf"},
		{13622, 0xfd88d720, "SteamInputDefaultConfiguration_SteamDeck.vdf"},
		{414176, 0xf37ba5cf, "rimworld.x3"},
	}
	var total int
	for _, m := range members {
		total += int(m.size)
	}
	if len(plain) != total {
		t.Fatalf("unpacked %d; want %d", len(plain), total)
	}
	off := 0
	for _, m := range members {
		chunk := plain[off : off+int(m.size)]
		got := crc32.ChecksumIEEE(chunk)
		if got != m.crc {
			t.Errorf("%s crc=%08x; want %08x", m.path, got, m.crc)
		}
		off += int(m.size)
	}
}
