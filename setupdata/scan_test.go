package setupdata

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	ulzma "github.com/ulikunitz/xz/lzma"
)

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
	f := openCorpusFile(t, "setup.exe")
	info, err := Scan(f)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(info.InstalledMD5, "\n"); n != 1712 {
		t.Fatalf("installed manifest has %d entries, want 1712", n)
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

func TestInstalledManifestEncoding(t *testing.T) {
	prefix := "900150983cd24fb0d6963f7d28e17f72 *..\\Data\\"
	for _, tt := range []struct {
		name    string
		encoded []byte
		want    string
	}{
		{"UTF8", []byte(prefix + "Я.txt\r\n"), prefix + "Я.txt\r\n"},
		{"Windows1251", append([]byte(prefix), []byte{0xdf, '.', 't', 'x', 't', '\r', '\n'}...), prefix + "Я.txt\r\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := append([]byte("\x00unrelated\x00"), tt.encoded...)
			if got := installedMD5(input); got != tt.want {
				t.Fatalf("got %q; want %q", got, tt.want)
			}
		})
	}
	if got := installedMD5([]byte("900150983cd24fb0d6963f7d28e17f72 *..\\truncated")); got != "" {
		t.Fatalf("accepted unterminated manifest: %q", got)
	}
}
