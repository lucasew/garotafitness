package srep

import (
	"errors"
	"io"
)

// Blocked: v3 Future-LZ is not a wasm32-wasip1 Guest. Official srep.cpp
// includes MultiThreading.cpp (pthread_create, -lpthread). The v3 path
// file_seek()s the output and may fopen() srep-virtual-memory.tmp.
// WASI preview1 has no pthreads; wazero does not implement wasi-threads.

var (
	errWASI = errors.New("srep: v3 Future-LZ needs pthreads and a seekable tempfile; wasm32-wasip1/wazero provide neither")
	errNil  = errors.New("srep: nil reader")
)

// NewReader wraps an official SREP stream as compress/gzip does.
func NewReader(r io.Reader) (io.ReadCloser, error) {
	if r == nil {
		return nil, errNil
	}
	if _, err := ParseHeader(r); err != nil {
		return nil, err
	}
	return nil, errWASI
}
