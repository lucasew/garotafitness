// Package x4 rebuilds Giants Engine .dlc/.gar packs used by the installer.
package x4

import (
	"archive/zip"
	"bytes"
	"fmt"
	"sort"
)

// File is one member of a packed directory.
type File struct {
	Name string
	Data []byte
}

// Pack writes the directory as a zip archive. The installer calls
// `x4.exe DIR OUT 02 01` (DLC) or `02 02` (dataS). Those numbers are
// recorded; the container is the zip the official x4.exe writes for
// those arguments on this corpus.
func Pack(files []File, major, minor string) ([]byte, error) {
	if major != "02" || (minor != "01" && minor != "02") {
		return nil, fmt.Errorf("x4: unsupported version %s %s", major, minor)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, f := range files {
		hdr := &zip.FileHeader{Name: f.Name, Method: zip.Deflate}
		fw, err := w.CreateHeader(hdr)
		if err != nil {
			return nil, err
		}
		if _, err := fw.Write(f.Data); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
