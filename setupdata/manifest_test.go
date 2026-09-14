package setupdata

import (
	"bytes"
	"encoding/binary"
	"testing"
	"unicode/utf16"

	"github.com/stretchr/testify/require"
)

func TestScanManifestWithDifferentDestinations(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ dest, member string }{
		{`{app}\verify.md5`, `Assets\payload.dat`},
		{`{app}\Checks\files.md5`, `..\Assets\payload.dat`},
		{`{app}\Verify\Nested\checksums.md5`, `../../Assets/payload.dat`},
	} {
		t.Run(tc.dest, func(t *testing.T) {
			var encoded []byte
			for _, c := range utf16.Encode([]rune(tc.dest)) {
				encoded = binary.LittleEndian.AppendUint16(encoded, c)
			}
			for _, mode := range []string{"*", " "} {
				data := binary.LittleEndian.AppendUint32(nil, uint32(len(encoded)))
				data = append(data, encoded...)
				data = append(data, 0)
				manifest := "900150983cd24fb0d6963f7d28e17f72 " + mode + tc.member + "\r\n"
				data = append(data, manifest...)
				info, err := Scan(bytes.NewReader(data))
				require.NoError(t, err)
				require.Equal(t, tc.dest, info.ManifestPath)
				require.Equal(t, manifest, info.InstalledMD5)
			}
		})
	}
}
