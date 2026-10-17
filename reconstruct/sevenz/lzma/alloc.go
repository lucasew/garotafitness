package lzma

import (
	"sync"
	"syscall"
	"unsafe"
)

// C one-past-end pointers and an explicit free of a live *T are both fatal
// to the Go garbage collector. Allocator blocks therefore come from mmap,
// which the collector does not scan. keeps holds Go values (match-finder
// closures, input bytes) that those blocks point at.

var (
	pinMu sync.Mutex
	pins  = map[uintptr][]byte{}
	keeps = map[uintptr][]any{}
)

func MyAlloc(size uint32) unsafe.Pointer {
	if size == 0 {
		return nil
	}
	n := int(size) + 4096
	if n < int(size) {
		return nil
	}
	b, err := syscall.Mmap(-1, 0, n, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		return nil
	}
	p := unsafe.Pointer(&b[0])
	pinMu.Lock()
	pins[uintptr(p)] = b
	pinMu.Unlock()
	return p
}

func MyFree(address unsafe.Pointer) {
	if address == nil {
		return
	}
	pinMu.Lock()
	b := pins[uintptr(address)]
	delete(pins, uintptr(address))
	delete(keeps, uintptr(address))
	pinMu.Unlock()
	if b != nil {
		_ = syscall.Munmap(b)
	}
}

// pinValue keeps v reachable for as long as the mmap block at key is alive.
func pinValue(key unsafe.Pointer, v any) {
	if key == nil || v == nil {
		return
	}
	pinMu.Lock()
	keeps[uintptr(key)] = append(keeps[uintptr(key)], v)
	pinMu.Unlock()
}

func unpinAll(key unsafe.Pointer) {
	if key == nil {
		return
	}
	pinMu.Lock()
	delete(keeps, uintptr(key))
	pinMu.Unlock()
}

func SzAlloc(p ISzAllocPtr, size uint32) unsafe.Pointer { return MyAlloc(size) }

func SzFree(p ISzAllocPtr, address unsafe.Pointer) { MyFree(address) }

var g_Alloc = ISzAlloc{Alloc: SzAlloc, Free: SzFree}
var g_BigAlloc = ISzAlloc{Alloc: SzAlloc, Free: SzFree}

func AllocPtr() ISzAllocPtr { return &g_Alloc }
