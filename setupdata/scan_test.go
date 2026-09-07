package setupdata

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	ulzma "github.com/ulikunitz/xz/lzma"
)

const rimworldSetup = `/media/downloads/TORRENTS/RimWorld [FitGirl Repack]/setup.exe`

func TestScanNil(t *testing.T) {
	t.Parallel()
	if _, err := Scan(nil); !errors.Is(err, errNil) {
		t.Fatalf("got %v", err)
	}
}

func TestScanPlain(t *testing.T) {
	t.Parallel()
	in := []byte("" +
		"[External compressor:srep]\r\n" +
		"unpackcmd = srep d\r\n" +
		"\r\n" +
		"[External compressor:mpzz]\r\n" +
		"header = 0\r\n")
	info, err := Scan(bytes.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if !hasAll(info.Encoders, "srep", "mpzz") {
		t.Fatalf("encoders %v", info.Encoders)
	}
	if !strings.Contains(info.ArcINI, "[External compressor:srep]") {
		t.Fatalf("arc.ini %q", info.ArcINI)
	}
}

func TestScanZLB(t *testing.T) {
	t.Parallel()
	plain := []byte("" +
		"[External compressor:rzw]\r\n" +
		"unpackcmd = rzw d f2 f1 128 128\r\n")
	info, err := Scan(bytes.NewReader(packZLB(t, plain)))
	if err != nil {
		t.Fatal(err)
	}
	if !hasAll(info.Encoders, "rzw") {
		t.Fatalf("encoders %v", info.Encoders)
	}
	if !strings.Contains(info.ArcINI, "[External compressor:rzw]") {
		t.Fatalf("arc.ini %q", info.ArcINI)
	}
}

func TestScanCorpus(t *testing.T) {
	t.Parallel()
	f, err := os.Open(rimworldSetup)
	if err != nil {
		t.Skip("corpus not mounted")
	}
	t.Cleanup(func() { f.Close() })
	info, err := Scan(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Encoders) == 0 {
		t.Fatal("empty encoder list")
	}
	if !hasAll(info.Encoders, "srep") {
		t.Fatalf("missing srep in %v", info.Encoders)
	}
	if !hasAny(info.Encoders, "mpzz", "magic2", "rzw") {
		t.Fatalf("missing mpzz/magic2/rzw in %v", info.Encoders)
	}
	if !strings.Contains(info.ArcINI, "[External compressor:") {
		t.Fatalf("arc.ini missing: %q", clip(info.ArcINI, 120))
	}
}

func packZLB(t *testing.T, plain []byte) []byte {
	t.Helper()
	const dict = 1 << 16
	var buf bytes.Buffer
	buf.WriteString(zlbMagic)
	buf.WriteByte(8) // LZMA2 prop for 64 KiB
	w, err := ulzma.Writer2Config{DictCap: dict}.NewWriter2(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(plain); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func hasAll(got []string, want ...string) bool {
	for _, w := range want {
		if !hasAny(got, w) {
			return false
		}
	}
	return true
}

func hasAny(got []string, want ...string) bool {
	for _, g := range got {
		for _, w := range want {
			if g == w {
				return true
			}
		}
	}
	return false
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
