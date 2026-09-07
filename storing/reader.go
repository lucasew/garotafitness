package storing

import (
	"io"
)

// NewReader returns r as an uncompressed stream.
func NewReader(r io.Reader) (io.ReadCloser, error) {
	if r == nil {
		return nil, errNil
	}
	if rc, ok := r.(io.ReadCloser); ok {
		return rc, nil
	}
	return io.NopCloser(r), nil
}

type errString string

func (e errString) Error() string { return string(e) }

const errNil = errString("storing: nil reader")
