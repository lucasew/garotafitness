package bz2

import (
	"syscall"
	"unsafe"
)

const (
	FALLBACK_QSORT_STACK_SIZE   = 100
	FALLBACK_QSORT_SMALL_THRESH = 10
	MAIN_QSORT_STACK_SIZE       = 100
	MAIN_QSORT_SMALL_THRESH     = 20
)

func bz_internal_error(code int32) {
	panic(code)
}

func cbool(v bool) Bool {
	if v {
		return 1
	}
	return 0
}

var bzVersion = [...]byte{'1', '.', '0', '.', '6', ',', ' ', '6', '-', 'S', 'e', 'p', 't', '-', '2', '0', '1', '0', 0}

// bmalloc is outside the Go heap. The translated compressor keeps C pointers
// in EState, and those must not be collected.
func bmalloc(n int) unsafe.Pointer {
	if n < 0 {
		return nil
	}
	total := n + 16
	b, err := syscall.Mmap(-1, 0, total, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		return nil
	}
	*(*int)(unsafe.Pointer(&b[0])) = len(b)
	return unsafe.Pointer(&b[16])
}

func bfree(p unsafe.Pointer) {
	if p == nil {
		return
	}
	base := unsafe.Add(p, -16)
	size := *(*int)(base)
	_ = syscall.Munmap(unsafe.Slice((*byte)(base), size))
}
