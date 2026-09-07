package garotafitness

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
	"testing"
	"testing/fstest"
)

func TestParseMD5(t *testing.T) {
	t.Parallel()
	in := "; comment\n" +
		"2c2800d798bcb735b1a92fdfc41f0443 *..\\fg-01.bin\n" +
		"b8756dfec9afb91b21e8a209f0e0a4fa *..\\fg-optional-bonus-soundtrack.bin\n"
	got, err := parseMD5(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if got["fg-01.bin"] != "2c2800d798bcb735b1a92fdfc41f0443" {
		t.Fatalf("got %#v", got)
	}
	if _, ok := got["fg-optional-bonus-soundtrack.bin"]; !ok {
		t.Fatal("missing optional")
	}
}

func TestVerifyChecksumsSkipMissingOptional(t *testing.T) {
	t.Parallel()
	body := []byte(arcMagic + "storing")
	sum := md5.Sum(body)
	hexsum := hex.EncodeToString(sum[:])
	src := fstest.MapFS{
		"fg-01.bin": {Data: body},
		"MD5/fitgirl-bins.md5": {Data: []byte(
			hexsum + " *..\\fg-01.bin\n" +
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa *..\\fg-optional-ost.bin\n",
		)},
	}
	vols, err := listVolumes(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyChecksums(src, vols); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyChecksumsMismatch(t *testing.T) {
	t.Parallel()
	src := fstest.MapFS{
		"fg-01.bin":            {Data: []byte(arcMagic + "storing")},
		"MD5/fitgirl-bins.md5": {Data: []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa *..\\fg-01.bin\n")},
	}
	vols, err := listVolumes(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyChecksums(src, vols); err == nil {
		t.Fatal("want mismatch")
	}
}
