package bz2

import (
	"unsafe"
)

func fallbackSimpleSort(fmap *UInt32, eclass *UInt32, lo Int32, hi Int32) {
	var (
		i      Int32
		j      Int32
		tmp    Int32
		ec_tmp UInt32
	)
	if lo == hi {
		return
	}
	if hi-lo > 3 {
		for i = hi - 4; i >= lo; i-- {
			tmp = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(i))))
			ec_tmp = *(*UInt32)(unsafe.Add(unsafe.Pointer(eclass), unsafe.Sizeof(UInt32(0))*uintptr(tmp)))
			for j = i + 4; j <= hi && ec_tmp > *(*UInt32)(unsafe.Add(unsafe.Pointer(eclass), unsafe.Sizeof(UInt32(0))*uintptr(*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(j)))))); j += 4 {
				*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(j-4))) = *(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(j)))
			}
			*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(j-4))) = UInt32(tmp)
		}
	}
	for i = hi - 1; i >= lo; i-- {
		tmp = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(i))))
		ec_tmp = *(*UInt32)(unsafe.Add(unsafe.Pointer(eclass), unsafe.Sizeof(UInt32(0))*uintptr(tmp)))
		for j = i + 1; j <= hi && ec_tmp > *(*UInt32)(unsafe.Add(unsafe.Pointer(eclass), unsafe.Sizeof(UInt32(0))*uintptr(*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(j)))))); j++ {
			*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(j-1))) = *(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(j)))
		}
		*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(j-1))) = UInt32(tmp)
	}
}
func fallbackQSort3(fmap *UInt32, eclass *UInt32, loSt Int32, hiSt Int32) {
	var (
		unLo    Int32
		unHi    Int32
		ltLo    Int32
		gtHi    Int32
		n       Int32
		m       Int32
		sp      Int32
		lo      Int32
		hi      Int32
		med     UInt32
		r       UInt32
		r3      UInt32
		stackLo [100]Int32
		stackHi [100]Int32
	)
	r = 0
	sp = 0
	{
		stackLo[sp] = loSt
		stackHi[sp] = hiSt
		sp++
	}
	for sp > 0 {
		{
			if sp >= Int32(int32(FALLBACK_QSORT_STACK_SIZE-1)) {
				bz_internal_error(1004)
			}
		}
		{
			sp--
			lo = stackLo[sp]
			hi = stackHi[sp]
		}
		if hi-lo < FALLBACK_QSORT_SMALL_THRESH {
			fallbackSimpleSort(fmap, eclass, lo, hi)
			continue
		}
		r = ((r * 7621) + 1) % 32768
		r3 = r % 3
		if r3 == 0 {
			med = *(*UInt32)(unsafe.Add(unsafe.Pointer(eclass), unsafe.Sizeof(UInt32(0))*uintptr(*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(lo))))))
		} else if r3 == 1 {
			med = *(*UInt32)(unsafe.Add(unsafe.Pointer(eclass), unsafe.Sizeof(UInt32(0))*uintptr(*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr((lo+hi)>>1))))))
		} else {
			med = *(*UInt32)(unsafe.Add(unsafe.Pointer(eclass), unsafe.Sizeof(UInt32(0))*uintptr(*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(hi))))))
		}
		unLo = func() Int32 {
			ltLo = lo
			return ltLo
		}()
		unHi = func() Int32 {
			gtHi = hi
			return gtHi
		}()
		for {
			for {
				if unLo > unHi {
					break
				}
				n = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(eclass), unsafe.Sizeof(UInt32(0))*uintptr(*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(unLo))))))) - Int32(med)
				if n == 0 {
					{
						var zztmp Int32 = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(unLo))))
						*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(unLo))) = *(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(ltLo)))
						*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(ltLo))) = UInt32(zztmp)
					}
					ltLo++
					unLo++
					continue
				}
				if n > 0 {
					break
				}
				unLo++
			}
			for {
				if unLo > unHi {
					break
				}
				n = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(eclass), unsafe.Sizeof(UInt32(0))*uintptr(*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(unHi))))))) - Int32(med)
				if n == 0 {
					{
						var zztmp Int32 = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(unHi))))
						*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(unHi))) = *(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(gtHi)))
						*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(gtHi))) = UInt32(zztmp)
					}
					gtHi--
					unHi--
					continue
				}
				if n < 0 {
					break
				}
				unHi--
			}
			if unLo > unHi {
				break
			}
			{
				var zztmp Int32 = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(unLo))))
				*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(unLo))) = *(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(unHi)))
				*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(unHi))) = UInt32(zztmp)
			}
			unLo++
			unHi--
		}
		for {
			if true {
				break
			}
		}
		if gtHi < ltLo {
			continue
		}
		if (ltLo - lo) < (unLo - ltLo) {
			n = ltLo - lo
		} else {
			n = unLo - ltLo
		}
		{
			var (
				yyp1 Int32 = lo
				yyp2 Int32 = (unLo - n)
				yyn  Int32 = n
			)
			for yyn > 0 {
				{
					var zztmp Int32 = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(yyp1))))
					*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(yyp1))) = *(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(yyp2)))
					*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(yyp2))) = UInt32(zztmp)
				}
				yyp1++
				yyp2++
				yyn--
			}
		}
		if (hi - gtHi) < (gtHi - unHi) {
			m = hi - gtHi
		} else {
			m = gtHi - unHi
		}
		{
			var (
				yyp1 Int32 = unLo
				yyp2 Int32 = (hi - m + 1)
				yyn  Int32 = m
			)
			for yyn > 0 {
				{
					var zztmp Int32 = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(yyp1))))
					*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(yyp1))) = *(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(yyp2)))
					*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(yyp2))) = UInt32(zztmp)
				}
				yyp1++
				yyp2++
				yyn--
			}
		}
		n = lo + unLo - ltLo - 1
		m = hi - (gtHi - unHi) + 1
		if n-lo > hi-m {
			stackLo[sp] = lo
			stackHi[sp] = n
			sp++
			stackLo[sp] = m
			stackHi[sp] = hi
			sp++
		} else {
			{
				stackLo[sp] = m
				stackHi[sp] = hi
				sp++
			}
			{
				stackLo[sp] = lo
				stackHi[sp] = n
				sp++
			}
		}
	}
}
func fallbackSort(fmap *UInt32, eclass *UInt32, bhtab *UInt32, nblock Int32, verb Int32) {
	var (
		ftab     [257]Int32
		ftabCopy [256]Int32
		H        Int32
		i        Int32
		j        Int32
		k        Int32
		l        Int32
		r        Int32
		cc       Int32
		cc1      Int32
		nNotDone Int32
		nBhtab   Int32
		eclass8  *UChar = (*UChar)(unsafe.Pointer(eclass))
	)
	if verb >= 4 {
		for {
			if true {
				break
			}
		}
	}
	for i = 0; i < 257; i++ {
		ftab[i] = 0
	}
	for i = 0; i < nblock; i++ {
		ftab[*(*UChar)(unsafe.Add(unsafe.Pointer(eclass8), i))]++
	}
	for i = 0; i < 256; i++ {
		ftabCopy[i] = ftab[i]
	}
	for i = 1; i < 257; i++ {
		ftab[i] += ftab[i-1]
	}
	for i = 0; i < nblock; i++ {
		j = Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(eclass8), i)))
		k = ftab[j] - 1
		ftab[j] = k
		*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(k))) = UInt32(i)
	}
	nBhtab = (nblock / 32) + 2
	for i = 0; i < nBhtab; i++ {
		*(*UInt32)(unsafe.Add(unsafe.Pointer(bhtab), unsafe.Sizeof(UInt32(0))*uintptr(i))) = 0
	}
	for i = 0; i < 256; i++ {
		*(*UInt32)(unsafe.Add(unsafe.Pointer(bhtab), unsafe.Sizeof(UInt32(0))*uintptr((ftab[i])>>5))) |= UInt32(1 << ((ftab[i]) & 31))
	}
	for i = 0; i < 32; i++ {
		*(*UInt32)(unsafe.Add(unsafe.Pointer(bhtab), unsafe.Sizeof(UInt32(0))*uintptr((nblock+i*2)>>5))) |= UInt32(1 << ((nblock + i*2) & 31))
		*(*UInt32)(unsafe.Add(unsafe.Pointer(bhtab), unsafe.Sizeof(UInt32(0))*uintptr((nblock+i*2+1)>>5))) &= UInt32(int32(^(1 << ((nblock + i*2 + 1) & 31))))
	}
	H = 1
	for {
		if verb >= 4 {
			for {
				if true {
					break
				}
			}
		}
		j = 0
		for i = 0; i < nblock; i++ {
			if (*(*UInt32)(unsafe.Add(unsafe.Pointer(bhtab), unsafe.Sizeof(UInt32(0))*uintptr(i>>5))) & UInt32(1<<(i&31))) != 0 {
				j = i
			}
			k = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(i))) - UInt32(H))
			if k < 0 {
				k += nblock
			}
			*(*UInt32)(unsafe.Add(unsafe.Pointer(eclass), unsafe.Sizeof(UInt32(0))*uintptr(k))) = UInt32(j)
		}
		nNotDone = 0
		r = -1
		for {
			k = r + 1
			for (*(*UInt32)(unsafe.Add(unsafe.Pointer(bhtab), unsafe.Sizeof(UInt32(0))*uintptr(k>>5)))&UInt32(1<<(k&31))) != 0 && (k&0x1F) != 0 {
				k++
			}
			if (*(*UInt32)(unsafe.Add(unsafe.Pointer(bhtab), unsafe.Sizeof(UInt32(0))*uintptr(k>>5))) & UInt32(1<<(k&31))) != 0 {
				for *(*UInt32)(unsafe.Add(unsafe.Pointer(bhtab), unsafe.Sizeof(UInt32(0))*uintptr(k>>5))) == 0xFFFFFFFF {
					k += 32
				}
				for (*(*UInt32)(unsafe.Add(unsafe.Pointer(bhtab), unsafe.Sizeof(UInt32(0))*uintptr(k>>5))) & UInt32(1<<(k&31))) != 0 {
					k++
				}
			}
			l = k - 1
			if l >= nblock {
				break
			}
			for (*(*UInt32)(unsafe.Add(unsafe.Pointer(bhtab), unsafe.Sizeof(UInt32(0))*uintptr(k>>5)))&UInt32(1<<(k&31))) == 0 && (k&0x1F) != 0 {
				k++
			}
			if (*(*UInt32)(unsafe.Add(unsafe.Pointer(bhtab), unsafe.Sizeof(UInt32(0))*uintptr(k>>5))) & UInt32(1<<(k&31))) == 0 {
				for *(*UInt32)(unsafe.Add(unsafe.Pointer(bhtab), unsafe.Sizeof(UInt32(0))*uintptr(k>>5))) == 0x0 {
					k += 32
				}
				for (*(*UInt32)(unsafe.Add(unsafe.Pointer(bhtab), unsafe.Sizeof(UInt32(0))*uintptr(k>>5))) & UInt32(1<<(k&31))) == 0 {
					k++
				}
			}
			r = k - 1
			if r >= nblock {
				break
			}
			if r > l {
				nNotDone += r - l + 1
				fallbackQSort3(fmap, eclass, l, r)
				cc = -1
				for i = l; i <= r; i++ {
					cc1 = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(eclass), unsafe.Sizeof(UInt32(0))*uintptr(*(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(i)))))))
					if cc != cc1 {
						*(*UInt32)(unsafe.Add(unsafe.Pointer(bhtab), unsafe.Sizeof(UInt32(0))*uintptr(i>>5))) |= UInt32(1 << (i & 31))
						cc = cc1
					}
				}
			}
		}
		if verb >= 4 {
			for {
				if true {
					break
				}
			}
		}
		H *= 2
		if H > nblock || nNotDone == 0 {
			break
		}
	}
	if verb >= 4 {
		for {
			if true {
				break
			}
		}
	}
	j = 0
	for i = 0; i < nblock; i++ {
		for ftabCopy[j] == 0 {
			j++
		}
		ftabCopy[j]--
		*(*UChar)(unsafe.Add(unsafe.Pointer(eclass8), *(*UInt32)(unsafe.Add(unsafe.Pointer(fmap), unsafe.Sizeof(UInt32(0))*uintptr(i))))) = UChar(int8(j))
	}
	{
		if j >= 256 {
			bz_internal_error(1005)
		}
	}
}
func mainGtU(i1 UInt32, i2 UInt32, block *UChar, quadrant *UInt16, nblock UInt32, budget *Int32) Bool {
	var (
		k  Int32
		c1 UChar
		c2 UChar
		s1 UInt16
		s2 UInt16
	)
	for {
		if true {
			break
		}
	}
	c1 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i1))
	c2 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i2))
	if c1 != c2 {
		return cbool(c1 > c2)
	}
	i1++
	i2++
	c1 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i1))
	c2 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i2))
	if c1 != c2 {
		return cbool(c1 > c2)
	}
	i1++
	i2++
	c1 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i1))
	c2 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i2))
	if c1 != c2 {
		return cbool(c1 > c2)
	}
	i1++
	i2++
	c1 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i1))
	c2 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i2))
	if c1 != c2 {
		return cbool(c1 > c2)
	}
	i1++
	i2++
	c1 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i1))
	c2 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i2))
	if c1 != c2 {
		return cbool(c1 > c2)
	}
	i1++
	i2++
	c1 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i1))
	c2 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i2))
	if c1 != c2 {
		return cbool(c1 > c2)
	}
	i1++
	i2++
	c1 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i1))
	c2 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i2))
	if c1 != c2 {
		return cbool(c1 > c2)
	}
	i1++
	i2++
	c1 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i1))
	c2 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i2))
	if c1 != c2 {
		return cbool(c1 > c2)
	}
	i1++
	i2++
	c1 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i1))
	c2 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i2))
	if c1 != c2 {
		return cbool(c1 > c2)
	}
	i1++
	i2++
	c1 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i1))
	c2 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i2))
	if c1 != c2 {
		return cbool(c1 > c2)
	}
	i1++
	i2++
	c1 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i1))
	c2 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i2))
	if c1 != c2 {
		return cbool(c1 > c2)
	}
	i1++
	i2++
	c1 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i1))
	c2 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i2))
	if c1 != c2 {
		return cbool(c1 > c2)
	}
	i1++
	i2++
	k = Int32(nblock + 8)
	for {
		c1 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i1))
		c2 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i2))
		if c1 != c2 {
			return cbool(c1 > c2)
		}
		s1 = *(*UInt16)(unsafe.Add(unsafe.Pointer(quadrant), unsafe.Sizeof(UInt16(0))*uintptr(i1)))
		s2 = *(*UInt16)(unsafe.Add(unsafe.Pointer(quadrant), unsafe.Sizeof(UInt16(0))*uintptr(i2)))
		if s1 != s2 {
			return cbool(s1 > s2)
		}
		i1++
		i2++
		c1 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i1))
		c2 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i2))
		if c1 != c2 {
			return cbool(c1 > c2)
		}
		s1 = *(*UInt16)(unsafe.Add(unsafe.Pointer(quadrant), unsafe.Sizeof(UInt16(0))*uintptr(i1)))
		s2 = *(*UInt16)(unsafe.Add(unsafe.Pointer(quadrant), unsafe.Sizeof(UInt16(0))*uintptr(i2)))
		if s1 != s2 {
			return cbool(s1 > s2)
		}
		i1++
		i2++
		c1 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i1))
		c2 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i2))
		if c1 != c2 {
			return cbool(c1 > c2)
		}
		s1 = *(*UInt16)(unsafe.Add(unsafe.Pointer(quadrant), unsafe.Sizeof(UInt16(0))*uintptr(i1)))
		s2 = *(*UInt16)(unsafe.Add(unsafe.Pointer(quadrant), unsafe.Sizeof(UInt16(0))*uintptr(i2)))
		if s1 != s2 {
			return cbool(s1 > s2)
		}
		i1++
		i2++
		c1 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i1))
		c2 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i2))
		if c1 != c2 {
			return cbool(c1 > c2)
		}
		s1 = *(*UInt16)(unsafe.Add(unsafe.Pointer(quadrant), unsafe.Sizeof(UInt16(0))*uintptr(i1)))
		s2 = *(*UInt16)(unsafe.Add(unsafe.Pointer(quadrant), unsafe.Sizeof(UInt16(0))*uintptr(i2)))
		if s1 != s2 {
			return cbool(s1 > s2)
		}
		i1++
		i2++
		c1 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i1))
		c2 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i2))
		if c1 != c2 {
			return cbool(c1 > c2)
		}
		s1 = *(*UInt16)(unsafe.Add(unsafe.Pointer(quadrant), unsafe.Sizeof(UInt16(0))*uintptr(i1)))
		s2 = *(*UInt16)(unsafe.Add(unsafe.Pointer(quadrant), unsafe.Sizeof(UInt16(0))*uintptr(i2)))
		if s1 != s2 {
			return cbool(s1 > s2)
		}
		i1++
		i2++
		c1 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i1))
		c2 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i2))
		if c1 != c2 {
			return cbool(c1 > c2)
		}
		s1 = *(*UInt16)(unsafe.Add(unsafe.Pointer(quadrant), unsafe.Sizeof(UInt16(0))*uintptr(i1)))
		s2 = *(*UInt16)(unsafe.Add(unsafe.Pointer(quadrant), unsafe.Sizeof(UInt16(0))*uintptr(i2)))
		if s1 != s2 {
			return cbool(s1 > s2)
		}
		i1++
		i2++
		c1 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i1))
		c2 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i2))
		if c1 != c2 {
			return cbool(c1 > c2)
		}
		s1 = *(*UInt16)(unsafe.Add(unsafe.Pointer(quadrant), unsafe.Sizeof(UInt16(0))*uintptr(i1)))
		s2 = *(*UInt16)(unsafe.Add(unsafe.Pointer(quadrant), unsafe.Sizeof(UInt16(0))*uintptr(i2)))
		if s1 != s2 {
			return cbool(s1 > s2)
		}
		i1++
		i2++
		c1 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i1))
		c2 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i2))
		if c1 != c2 {
			return cbool(c1 > c2)
		}
		s1 = *(*UInt16)(unsafe.Add(unsafe.Pointer(quadrant), unsafe.Sizeof(UInt16(0))*uintptr(i1)))
		s2 = *(*UInt16)(unsafe.Add(unsafe.Pointer(quadrant), unsafe.Sizeof(UInt16(0))*uintptr(i2)))
		if s1 != s2 {
			return cbool(s1 > s2)
		}
		i1++
		i2++
		if i1 >= nblock {
			i1 -= nblock
		}
		if i2 >= nblock {
			i2 -= nblock
		}
		k -= 8
		(*budget)--
		if k < 0 {
			break
		}
	}
	return 0
}

var incs [14]Int32 = [14]Int32{1, 4, 13, 40, 121, 364, 1093, 3280, 9841, 29524, 88573, 265720, 797161, 2391484}

func mainSimpleSort(ptr *UInt32, block *UChar, quadrant *UInt16, nblock Int32, lo Int32, hi Int32, d Int32, budget *Int32) {
	var (
		i    Int32
		j    Int32
		h    Int32
		bigN Int32
		hp   Int32
		v    UInt32
	)
	bigN = hi - lo + 1
	if bigN < 2 {
		return
	}
	hp = 0
	for incs[hp] < bigN {
		hp++
	}
	hp--
	for ; hp >= 0; hp-- {
		h = incs[hp]
		i = lo + h
		for {
			if i > hi {
				break
			}
			v = *(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(i)))
			j = i
			for mainGtU(*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(j-h)))+UInt32(d), v+UInt32(d), block, quadrant, UInt32(nblock), budget) != 0 {
				*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(j))) = *(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(j-h)))
				j = j - h
				if j <= (lo + h - 1) {
					break
				}
			}
			*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(j))) = v
			i++
			if i > hi {
				break
			}
			v = *(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(i)))
			j = i
			for mainGtU(*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(j-h)))+UInt32(d), v+UInt32(d), block, quadrant, UInt32(nblock), budget) != 0 {
				*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(j))) = *(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(j-h)))
				j = j - h
				if j <= (lo + h - 1) {
					break
				}
			}
			*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(j))) = v
			i++
			if i > hi {
				break
			}
			v = *(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(i)))
			j = i
			for mainGtU(*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(j-h)))+UInt32(d), v+UInt32(d), block, quadrant, UInt32(nblock), budget) != 0 {
				*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(j))) = *(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(j-h)))
				j = j - h
				if j <= (lo + h - 1) {
					break
				}
			}
			*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(j))) = v
			i++
			if *budget < 0 {
				return
			}
		}
	}
}
func mmed3(a UChar, b UChar, c UChar) UChar {
	var t UChar
	if a > b {
		t = a
		a = b
		b = t
	}
	if b > c {
		b = c
		if a > b {
			b = a
		}
	}
	return b
}
func mainQSort3(ptr *UInt32, block *UChar, quadrant *UInt16, nblock Int32, loSt Int32, hiSt Int32, dSt Int32, budget *Int32) {
	var (
		unLo    Int32
		unHi    Int32
		ltLo    Int32
		gtHi    Int32
		n       Int32
		m       Int32
		med     Int32
		sp      Int32
		lo      Int32
		hi      Int32
		d       Int32
		stackLo [100]Int32
		stackHi [100]Int32
		stackD  [100]Int32
		nextLo  [3]Int32
		nextHi  [3]Int32
		nextD   [3]Int32
	)
	sp = 0
	{
		stackLo[sp] = loSt
		stackHi[sp] = hiSt
		stackD[sp] = dSt
		sp++
	}
	for sp > 0 {
		{
			if sp >= Int32(int32(MAIN_QSORT_STACK_SIZE-2)) {
				bz_internal_error(1001)
			}
		}
		{
			sp--
			lo = stackLo[sp]
			hi = stackHi[sp]
			d = stackD[sp]
		}
		if hi-lo < MAIN_QSORT_SMALL_THRESH || d > Int32(int32(BZ_N_RADIX+BZ_N_QSORT)) {
			mainSimpleSort(ptr, block, quadrant, nblock, lo, hi, d, budget)
			if *budget < 0 {
				return
			}
			continue
		}
		med = Int32(mmed3(*(*UChar)(unsafe.Add(unsafe.Pointer(block), *(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(lo)))+UInt32(d))), *(*UChar)(unsafe.Add(unsafe.Pointer(block), *(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(hi)))+UInt32(d))), *(*UChar)(unsafe.Add(unsafe.Pointer(block), *(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr((lo+hi)>>1)))+UInt32(d)))))
		unLo = func() Int32 {
			ltLo = lo
			return ltLo
		}()
		unHi = func() Int32 {
			gtHi = hi
			return gtHi
		}()
		for {
			for {
				if unLo > unHi {
					break
				}
				n = (Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(block), *(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(unLo)))+UInt32(d))))) - med
				if n == 0 {
					{
						var zztmp Int32 = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(unLo))))
						*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(unLo))) = *(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(ltLo)))
						*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(ltLo))) = UInt32(zztmp)
					}
					ltLo++
					unLo++
					continue
				}
				if n > 0 {
					break
				}
				unLo++
			}
			for {
				if unLo > unHi {
					break
				}
				n = (Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(block), *(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(unHi)))+UInt32(d))))) - med
				if n == 0 {
					{
						var zztmp Int32 = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(unHi))))
						*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(unHi))) = *(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(gtHi)))
						*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(gtHi))) = UInt32(zztmp)
					}
					gtHi--
					unHi--
					continue
				}
				if n < 0 {
					break
				}
				unHi--
			}
			if unLo > unHi {
				break
			}
			{
				var zztmp Int32 = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(unLo))))
				*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(unLo))) = *(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(unHi)))
				*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(unHi))) = UInt32(zztmp)
			}
			unLo++
			unHi--
		}
		for {
			if true {
				break
			}
		}
		if gtHi < ltLo {
			stackLo[sp] = lo
			stackHi[sp] = hi
			stackD[sp] = d + 1
			sp++
			continue
		}
		if (ltLo - lo) < (unLo - ltLo) {
			n = ltLo - lo
		} else {
			n = unLo - ltLo
		}
		{
			var (
				yyp1 Int32 = lo
				yyp2 Int32 = (unLo - n)
				yyn  Int32 = n
			)
			for yyn > 0 {
				{
					var zztmp Int32 = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(yyp1))))
					*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(yyp1))) = *(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(yyp2)))
					*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(yyp2))) = UInt32(zztmp)
				}
				yyp1++
				yyp2++
				yyn--
			}
		}
		if (hi - gtHi) < (gtHi - unHi) {
			m = hi - gtHi
		} else {
			m = gtHi - unHi
		}
		{
			var (
				yyp1 Int32 = unLo
				yyp2 Int32 = (hi - m + 1)
				yyn  Int32 = m
			)
			for yyn > 0 {
				{
					var zztmp Int32 = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(yyp1))))
					*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(yyp1))) = *(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(yyp2)))
					*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(yyp2))) = UInt32(zztmp)
				}
				yyp1++
				yyp2++
				yyn--
			}
		}
		n = lo + unLo - ltLo - 1
		m = hi - (gtHi - unHi) + 1
		nextLo[0] = lo
		nextHi[0] = n
		nextD[0] = d
		nextLo[1] = m
		nextHi[1] = hi
		nextD[1] = d
		nextLo[2] = n + 1
		nextHi[2] = m - 1
		nextD[2] = d + 1
		if (nextHi[0] - nextLo[0]) < (nextHi[1] - nextLo[1]) {
			var tz Int32
			tz = nextLo[0]
			nextLo[0] = nextLo[1]
			nextLo[1] = tz
			tz = nextHi[0]
			nextHi[0] = nextHi[1]
			nextHi[1] = tz
			tz = nextD[0]
			nextD[0] = nextD[1]
			nextD[1] = tz
		}
		if (nextHi[1] - nextLo[1]) < (nextHi[2] - nextLo[2]) {
			var tz Int32
			tz = nextLo[1]
			nextLo[1] = nextLo[2]
			nextLo[2] = tz
			tz = nextHi[1]
			nextHi[1] = nextHi[2]
			nextHi[2] = tz
			tz = nextD[1]
			nextD[1] = nextD[2]
			nextD[2] = tz
		}
		if (nextHi[0] - nextLo[0]) < (nextHi[1] - nextLo[1]) {
			var tz Int32
			tz = nextLo[0]
			nextLo[0] = nextLo[1]
			nextLo[1] = tz
			tz = nextHi[0]
			nextHi[0] = nextHi[1]
			nextHi[1] = tz
			tz = nextD[0]
			nextD[0] = nextD[1]
			nextD[1] = tz
		}
		for {
			if true {
				break
			}
		}
		for {
			if true {
				break
			}
		}
		{
			stackLo[sp] = nextLo[0]
			stackHi[sp] = nextHi[0]
			stackD[sp] = nextD[0]
			sp++
		}
		{
			stackLo[sp] = nextLo[1]
			stackHi[sp] = nextHi[1]
			stackD[sp] = nextD[1]
			sp++
		}
		{
			stackLo[sp] = nextLo[2]
			stackHi[sp] = nextHi[2]
			stackD[sp] = nextD[2]
			sp++
		}
	}
}
func mainSort(ptr *UInt32, block *UChar, quadrant *UInt16, ftab *UInt32, nblock Int32, verb Int32, budget *Int32) {
	var (
		i            Int32
		j            Int32
		k            Int32
		ss           Int32
		sb           Int32
		runningOrder [256]Int32
		bigDone      [256]Bool
		copyStart    [256]Int32
		copyEnd      [256]Int32
		c1           UChar
		numQSorted   Int32
	)
	_ = numQSorted
	var s UInt16
	if verb >= 4 {
		for {
			if true {
				break
			}
		}
	}
	for i = 65536; i >= 0; i-- {
		*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr(i))) = 0
	}
	j = Int32(*block) << 8
	i = nblock - 1
	for ; i >= 3; i -= 4 {
		*(*UInt16)(unsafe.Add(unsafe.Pointer(quadrant), unsafe.Sizeof(UInt16(0))*uintptr(i))) = 0
		j = (j >> 8) | Int32((UInt16(*(*UChar)(unsafe.Add(unsafe.Pointer(block), i))))<<8)
		*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr(j)))++
		*(*UInt16)(unsafe.Add(unsafe.Pointer(quadrant), unsafe.Sizeof(UInt16(0))*uintptr(i-1))) = 0
		j = (j >> 8) | Int32((UInt16(*(*UChar)(unsafe.Add(unsafe.Pointer(block), i-1))))<<8)
		*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr(j)))++
		*(*UInt16)(unsafe.Add(unsafe.Pointer(quadrant), unsafe.Sizeof(UInt16(0))*uintptr(i-2))) = 0
		j = (j >> 8) | Int32((UInt16(*(*UChar)(unsafe.Add(unsafe.Pointer(block), i-2))))<<8)
		*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr(j)))++
		*(*UInt16)(unsafe.Add(unsafe.Pointer(quadrant), unsafe.Sizeof(UInt16(0))*uintptr(i-3))) = 0
		j = (j >> 8) | Int32((UInt16(*(*UChar)(unsafe.Add(unsafe.Pointer(block), i-3))))<<8)
		*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr(j)))++
	}
	for ; i >= 0; i-- {
		*(*UInt16)(unsafe.Add(unsafe.Pointer(quadrant), unsafe.Sizeof(UInt16(0))*uintptr(i))) = 0
		j = (j >> 8) | Int32((UInt16(*(*UChar)(unsafe.Add(unsafe.Pointer(block), i))))<<8)
		*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr(j)))++
	}
	for i = 0; i < Int32(int32(BZ_N_RADIX+BZ_N_QSORT)+BZ_N_SHELL+2); i++ {
		*(*UChar)(unsafe.Add(unsafe.Pointer(block), nblock+i)) = *(*UChar)(unsafe.Add(unsafe.Pointer(block), i))
		*(*UInt16)(unsafe.Add(unsafe.Pointer(quadrant), unsafe.Sizeof(UInt16(0))*uintptr(nblock+i))) = 0
	}
	if verb >= 4 {
		for {
			if true {
				break
			}
		}
	}
	for i = 1; i <= 65536; i++ {
		*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr(i))) += *(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr(i-1)))
	}
	s = UInt16(*block) << 8
	i = nblock - 1
	for ; i >= 3; i -= 4 {
		s = (s >> 8) | (UInt16(*(*UChar)(unsafe.Add(unsafe.Pointer(block), i))) << 8)
		j = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr(s))) - 1)
		*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr(s))) = UInt32(j)
		*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(j))) = UInt32(i)
		s = (s >> 8) | (UInt16(*(*UChar)(unsafe.Add(unsafe.Pointer(block), i-1))) << 8)
		j = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr(s))) - 1)
		*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr(s))) = UInt32(j)
		*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(j))) = UInt32(i - 1)
		s = (s >> 8) | (UInt16(*(*UChar)(unsafe.Add(unsafe.Pointer(block), i-2))) << 8)
		j = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr(s))) - 1)
		*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr(s))) = UInt32(j)
		*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(j))) = UInt32(i - 2)
		s = (s >> 8) | (UInt16(*(*UChar)(unsafe.Add(unsafe.Pointer(block), i-3))) << 8)
		j = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr(s))) - 1)
		*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr(s))) = UInt32(j)
		*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(j))) = UInt32(i - 3)
	}
	for ; i >= 0; i-- {
		s = (s >> 8) | (UInt16(*(*UChar)(unsafe.Add(unsafe.Pointer(block), i))) << 8)
		j = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr(s))) - 1)
		*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr(s))) = UInt32(j)
		*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(j))) = UInt32(i)
	}
	for i = 0; i <= 255; i++ {
		bigDone[i] = 0
		runningOrder[i] = i
	}
	{
		var (
			vv Int32
			h  Int32 = 1
		)
		for {
			h = h*3 + 1
			if h > 256 {
				break
			}
		}
		for {
			h = h / 3
			for i = h; i <= 255; i++ {
				vv = runningOrder[i]
				j = i
				for (*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr(((runningOrder[j-h])+1)<<8))) - *(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr((runningOrder[j-h])<<8)))) > (*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr((vv+1)<<8))) - *(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr(vv<<8)))) {
					runningOrder[j] = runningOrder[j-h]
					j = j - h
					if j <= (h - 1) {
						goto zero
					}
				}
			zero:
				runningOrder[j] = vv
			}
			if h == 1 {
				break
			}
		}
	}
	numQSorted = 0
	for i = 0; i <= 255; i++ {
		ss = runningOrder[i]
		for j = 0; j <= 255; j++ {
			if j != ss {
				sb = (ss << 8) + j
				if (*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr(sb))) & (1 << 21)) == 0 {
					var (
						lo Int32 = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr(sb))) & (^UInt32(1 << 21)))
						hi Int32 = Int32((*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr(sb+1))) & (^UInt32(1 << 21))) - 1)
					)
					if hi > lo {
						if verb >= 4 {
							for {
								if true {
									break
								}
							}
						}
						mainQSort3(ptr, block, quadrant, nblock, lo, hi, BZ_N_RADIX, budget)
						numQSorted += hi - lo + 1
						if *budget < 0 {
							return
						}
					}
				}
				*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr(sb))) |= 1 << 21
			}
		}
		{
			if bigDone[ss] != 0 {
				bz_internal_error(1006)
			}
		}
		{
			for j = 0; j <= 255; j++ {
				copyStart[j] = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr((j<<8)+ss))) & (^UInt32(1 << 21)))
				copyEnd[j] = Int32((*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr((j<<8)+ss+1))) & (^UInt32(1 << 21))) - 1)
			}
			for j = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr(ss<<8))) & (^UInt32(1 << 21))); j < copyStart[ss]; j++ {
				k = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(j))) - 1)
				if k < 0 {
					k += nblock
				}
				c1 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), k))
				if bigDone[c1] == 0 {
					*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(func() Int32 {
						p_ := &copyStart[c1]
						x := *p_
						*p_++
						return x
					}()))) = UInt32(k)
				}
			}
			for j = Int32((*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr((ss+1)<<8))) & (^UInt32(1 << 21))) - 1); j > copyEnd[ss]; j-- {
				k = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(j))) - 1)
				if k < 0 {
					k += nblock
				}
				c1 = *(*UChar)(unsafe.Add(unsafe.Pointer(block), k))
				if bigDone[c1] == 0 {
					*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(func() Int32 {
						p_ := &copyEnd[c1]
						x := *p_
						*p_--
						return x
					}()))) = UInt32(k)
				}
			}
		}
		{
			if copyStart[ss]-1 != copyEnd[ss] && (copyStart[ss] != 0 || copyEnd[ss] != nblock-1) {
				bz_internal_error(1007)
			}
		}
		for j = 0; j <= 255; j++ {
			*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr((j<<8)+ss))) |= 1 << 21
		}
		bigDone[ss] = 1
		if i < 255 {
			var (
				bbStart Int32 = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr(ss<<8))) & (^UInt32(1 << 21)))
				bbSize  Int32 = Int32((*(*UInt32)(unsafe.Add(unsafe.Pointer(ftab), unsafe.Sizeof(UInt32(0))*uintptr((ss+1)<<8))) & (^UInt32(1 << 21))) - UInt32(bbStart))
				shifts  Int32 = 0
			)
			for (bbSize >> shifts) > 65534 {
				shifts++
			}
			for j = bbSize - 1; j >= 0; j-- {
				var (
					a2update Int32  = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(bbStart+j))))
					qVal     UInt16 = UInt16(int16(j >> shifts))
				)
				*(*UInt16)(unsafe.Add(unsafe.Pointer(quadrant), unsafe.Sizeof(UInt16(0))*uintptr(a2update))) = qVal
				if a2update < Int32(int32(BZ_N_RADIX+BZ_N_QSORT)+BZ_N_SHELL+2) {
					*(*UInt16)(unsafe.Add(unsafe.Pointer(quadrant), unsafe.Sizeof(UInt16(0))*uintptr(a2update+nblock))) = qVal
				}
			}
			if ((bbSize - 1) >> shifts) > 65535 {
				bz_internal_error(1002)
			}
		}
	}
	if verb >= 4 {
		for {
			if true {
				break
			}
		}
	}
}
func BZ2_blockSort(s *EState) {
	var (
		ptr        *UInt32 = s.Ptr
		block      *UChar  = s.Block
		ftab       *UInt32 = s.Ftab
		nblock     Int32   = s.Nblock
		verb       Int32   = s.Verbosity
		wfact      Int32   = s.WorkFactor
		quadrant   *UInt16
		budget     Int32
		budgetInit Int32
		i          Int32
	)
	if nblock < 10000 {
		fallbackSort(s.Arr1, s.Arr2, ftab, nblock, verb)
	} else {
		i = nblock + Int32(int32(BZ_N_RADIX+BZ_N_QSORT)+BZ_N_SHELL+2)
		if i&1 != 0 {
			i++
		}
		quadrant = (*UInt16)(unsafe.Pointer((*UChar)(unsafe.Add(unsafe.Pointer(block), i))))
		if wfact < 1 {
			wfact = 1
		}
		if wfact > 100 {
			wfact = 100
		}
		budgetInit = nblock * ((wfact - 1) / 3)
		budget = budgetInit
		mainSort(ptr, block, quadrant, ftab, nblock, verb, &budget)
		if verb >= 3 {
			for {
				if true {
					break
				}
			}
		}
		if budget < 0 {
			if verb >= 2 {
				for {
					if true {
						break
					}
				}
			}
			fallbackSort(s.Arr1, s.Arr2, ftab, nblock, verb)
		}
	}
	s.OrigPtr = -1
	for i = 0; i < s.Nblock; i++ {
		if *(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(i))) == 0 {
			s.OrigPtr = i
			break
		}
	}
	{
		if s.OrigPtr == -1 {
			bz_internal_error(1003)
		}
	}
}
