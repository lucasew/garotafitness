package setupdata

import (
	"bytes"
	"encoding/binary"
	"strings"
	"unicode/utf16"
)

// Inno's file table stores destination strings with a byte-length prefix.
// Require that framing, rather than taking a filename from nearby script text.
func installedManifestPath(data []byte) string {
	prefix := []byte{'{', 0, 'a', 0, 'p', 0, 'p', 0, '}', 0}
	var result string
	for start := 0; start < len(data); {
		i := bytes.Index(data[start:], prefix)
		if i < 0 {
			break
		}
		i += start
		start = i + len(prefix)
		if i < 4 {
			continue
		}
		n := uint64(binary.LittleEndian.Uint32(data[i-4 : i]))
		if n < 10 || n > 4096 || n%2 != 0 || n > uint64(len(data)-i) {
			continue
		}
		b := data[i : i+int(n)]
		chars := make([]uint16, len(b)/2)
		for j := range chars {
			chars[j] = binary.LittleEndian.Uint16(b[2*j:])
		}
		name := string(utf16.Decode(chars))
		if !strings.HasSuffix(strings.ToLower(name), ".md5") || strings.ContainsRune(name, 0) {
			continue
		}
		if result != "" && result != name {
			return ""
		} // an ambiguous destination needs more metadata
		result = name
	}
	return result
}
