package bz2

import "unsafe"

func BZ2_hbMakeCodeLengths(len_ *UChar, freq *Int32, alphaSize Int32, maxLen Int32) {
	var (
		nNodes  Int32
		nHeap   Int32
		n1      Int32
		n2      Int32
		i       Int32
		j       Int32
		k       Int32
		tooLong Bool
		heap    [260]Int32
		weight  [516]Int32
		parent  [516]Int32
	)
	for i = 0; i < alphaSize; i++ {
		weight[i+1] = (func() Int32 {
			if *(*Int32)(unsafe.Add(unsafe.Pointer(freq), unsafe.Sizeof(Int32(0))*uintptr(i))) == 0 {
				return 1
			}
			return *(*Int32)(unsafe.Add(unsafe.Pointer(freq), unsafe.Sizeof(Int32(0))*uintptr(i)))
		}()) << 8
	}
	for {
		nNodes = alphaSize
		nHeap = 0
		heap[0] = 0
		weight[0] = 0
		parent[0] = -2
		for i = 1; i <= alphaSize; i++ {
			parent[i] = -1
			nHeap++
			heap[nHeap] = i
			{
				var (
					zz  Int32
					tmp Int32
				)
				zz = nHeap
				tmp = heap[zz]
				for weight[tmp] < weight[heap[zz>>1]] {
					heap[zz] = heap[zz>>1]
					zz >>= 1
				}
				heap[zz] = tmp
			}
		}
		{
			if nHeap >= Int32(int32(BZ_MAX_ALPHA_SIZE+2)) {
				bz_internal_error(2001)
			}
		}
		for nHeap > 1 {
			n1 = heap[1]
			heap[1] = heap[nHeap]
			nHeap--
			{
				var (
					zz  Int32
					yy  Int32
					tmp Int32
				)
				zz = 1
				tmp = heap[zz]
				for {
					yy = zz << 1
					if yy > nHeap {
						break
					}
					if yy < nHeap && weight[heap[yy+1]] < weight[heap[yy]] {
						yy++
					}
					if weight[tmp] < weight[heap[yy]] {
						break
					}
					heap[zz] = heap[yy]
					zz = yy
				}
				heap[zz] = tmp
			}
			n2 = heap[1]
			heap[1] = heap[nHeap]
			nHeap--
			{
				var (
					zz  Int32
					yy  Int32
					tmp Int32
				)
				zz = 1
				tmp = heap[zz]
				for {
					yy = zz << 1
					if yy > nHeap {
						break
					}
					if yy < nHeap && weight[heap[yy+1]] < weight[heap[yy]] {
						yy++
					}
					if weight[tmp] < weight[heap[yy]] {
						break
					}
					heap[zz] = heap[yy]
					zz = yy
				}
				heap[zz] = tmp
			}
			nNodes++
			parent[n1] = func() Int32 {
				p_ := &parent[n2]
				*p_ = nNodes
				return *p_
			}()
			weight[nNodes] = (((weight[n1]) & (-256)) + ((weight[n2]) & (-256))) | ((func() Int32 {
				if ((weight[n1]) & 0xFF) > ((weight[n2]) & 0xFF) {
					return (weight[n1]) & 0xFF
				}
				return (weight[n2]) & 0xFF
			}()) + 1)
			parent[nNodes] = -1
			nHeap++
			heap[nHeap] = nNodes
			{
				var (
					zz  Int32
					tmp Int32
				)
				zz = nHeap
				tmp = heap[zz]
				for weight[tmp] < weight[heap[zz>>1]] {
					heap[zz] = heap[zz>>1]
					zz >>= 1
				}
				heap[zz] = tmp
			}
		}
		{
			if nNodes >= Int32(int32(BZ_MAX_ALPHA_SIZE*2)) {
				bz_internal_error(2002)
			}
		}
		tooLong = 0
		for i = 1; i <= alphaSize; i++ {
			j = 0
			k = i
			for parent[k] >= 0 {
				k = parent[k]
				j++
			}
			*(*UChar)(unsafe.Add(unsafe.Pointer(len_), i-1)) = UChar(int8(j))
			if j > maxLen {
				tooLong = 1
			}
		}
		if tooLong == 0 {
			break
		}
		for i = 1; i <= alphaSize; i++ {
			j = weight[i] >> 8
			j = (j / 2) + 1
			weight[i] = j << 8
		}
	}
}
func BZ2_hbAssignCodes(code *Int32, length *UChar, minLen Int32, maxLen Int32, alphaSize Int32) {
	var (
		n   Int32
		vec Int32
		i   Int32
	)
	vec = 0
	for n = minLen; n <= maxLen; n++ {
		for i = 0; i < alphaSize; i++ {
			if Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(length), i))) == n {
				*(*Int32)(unsafe.Add(unsafe.Pointer(code), unsafe.Sizeof(Int32(0))*uintptr(i))) = vec
				vec++
			}
		}
		vec <<= 1
	}
}
func BZ2_hbCreateDecodeTables(limit *Int32, base *Int32, perm *Int32, length *UChar, minLen Int32, maxLen Int32, alphaSize Int32) {
	var (
		pp  Int32
		i   Int32
		j   Int32
		vec Int32
	)
	pp = 0
	for i = minLen; i <= maxLen; i++ {
		for j = 0; j < alphaSize; j++ {
			if Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(length), j))) == i {
				*(*Int32)(unsafe.Add(unsafe.Pointer(perm), unsafe.Sizeof(Int32(0))*uintptr(pp))) = j
				pp++
			}
		}
	}
	for i = 0; i < BZ_MAX_CODE_LEN; i++ {
		*(*Int32)(unsafe.Add(unsafe.Pointer(base), unsafe.Sizeof(Int32(0))*uintptr(i))) = 0
	}
	for i = 0; i < alphaSize; i++ {
		*(*Int32)(unsafe.Add(unsafe.Pointer(base), unsafe.Sizeof(Int32(0))*uintptr(*(*UChar)(unsafe.Add(unsafe.Pointer(length), i))+1)))++
	}
	for i = 1; i < BZ_MAX_CODE_LEN; i++ {
		*(*Int32)(unsafe.Add(unsafe.Pointer(base), unsafe.Sizeof(Int32(0))*uintptr(i))) += *(*Int32)(unsafe.Add(unsafe.Pointer(base), unsafe.Sizeof(Int32(0))*uintptr(i-1)))
	}
	for i = 0; i < BZ_MAX_CODE_LEN; i++ {
		*(*Int32)(unsafe.Add(unsafe.Pointer(limit), unsafe.Sizeof(Int32(0))*uintptr(i))) = 0
	}
	vec = 0
	for i = minLen; i <= maxLen; i++ {
		vec += *(*Int32)(unsafe.Add(unsafe.Pointer(base), unsafe.Sizeof(Int32(0))*uintptr(i+1))) - *(*Int32)(unsafe.Add(unsafe.Pointer(base), unsafe.Sizeof(Int32(0))*uintptr(i)))
		*(*Int32)(unsafe.Add(unsafe.Pointer(limit), unsafe.Sizeof(Int32(0))*uintptr(i))) = vec - 1
		vec <<= 1
	}
	for i = minLen + 1; i <= maxLen; i++ {
		*(*Int32)(unsafe.Add(unsafe.Pointer(base), unsafe.Sizeof(Int32(0))*uintptr(i))) = ((*(*Int32)(unsafe.Add(unsafe.Pointer(limit), unsafe.Sizeof(Int32(0))*uintptr(i-1))) + 1) << 1) - *(*Int32)(unsafe.Add(unsafe.Pointer(base), unsafe.Sizeof(Int32(0))*uintptr(i)))
	}
}
