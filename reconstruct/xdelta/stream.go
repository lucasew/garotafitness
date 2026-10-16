package xdelta

import (
	"fmt"
	"io"
)

// SeqWriter turns an append-only Writer into a WriterAt for ApplyStream.
func SeqWriter(w io.Writer) *SeqWriterAt {
	return &SeqWriterAt{w: w}
}

// SeqWriterAt records the number of bytes written.
type SeqWriterAt struct {
	w   io.Writer
	off int64
}

func (s *SeqWriterAt) Size() int64 { return s.off }

func (s *SeqWriterAt) WriteAt(p []byte, pos int64) (int, error) {
	if pos != s.off {
		return 0, fmt.Errorf("xdelta: non-sequential write at %d want %d", pos, s.off)
	}
	n, err := s.w.Write(p)
	s.off += int64(n)
	return n, err
}
