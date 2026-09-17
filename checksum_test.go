package garotafitness

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestParseMD5(t *testing.T) {
	t.Parallel()
	in := "; comment\n" +
		"2c2800d798bcb735b1a92fdfc41f0443 *..\\fg-01.bin\n" +
		"b8756dfec9afb91b21e8a209f0e0a4fa *..\\fg-optional-bonus-soundtrack.bin\n"
	got, err := parseMD5(strings.NewReader(in))
	require.NoError(t, err)
	require.Equal(t, "2c2800d798bcb735b1a92fdfc41f0443", got["fg-01.bin"])
	require.Contains(t, got, "fg-optional-bonus-soundtrack.bin")
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
	require.NoError(t, err)
	require.NoError(t, verifyChecksums(t.Context(), src, vols, nil))
}

func TestVerifyChecksumsMismatch(t *testing.T) {
	t.Parallel()
	src := fstest.MapFS{
		"fg-01.bin":            {Data: []byte(arcMagic + "storing")},
		"MD5/fitgirl-bins.md5": {Data: []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa *..\\fg-01.bin\n")},
	}
	vols, err := listVolumes(src)
	require.NoError(t, err)
	require.ErrorContains(t, verifyChecksums(t.Context(), src, vols, nil), "mismatch")
}

func TestVerifyChecksumsManyFiles(t *testing.T) {
	t.Parallel()
	src := fstest.MapFS{}
	var lines []string
	for i := 1; i <= 8; i++ {
		name := fmt.Sprintf("fg-%02d.bin", i)
		body := []byte(fmt.Sprintf("%s%d", arcMagic, i))
		sum := md5.Sum(body)
		src[name] = &fstest.MapFile{Data: body}
		lines = append(lines, hex.EncodeToString(sum[:])+" *..\\"+name)
	}
	src[checksumName] = &fstest.MapFile{Data: []byte(strings.Join(lines, "\n") + "\n")}
	vols, err := listVolumes(src)
	require.NoError(t, err)
	require.NoError(t, verifyChecksums(t.Context(), src, vols, nil))
}
