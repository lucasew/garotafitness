package dispack

import (
	"encoding/binary"
)

// Stream IDs and opcodes from DisPack.cpp.

const (
	stOp = iota
	stSIB
	stCallIdx
	stDisp8R0
	stDisp8R1
	stDisp8R2
	stDisp8R3
	stDisp8R4
	stDisp8R5
	stDisp8R6
	stDisp8R7
	stJump8
	stImm8
	stImm16
	stImm32
	stDisp32
	stAddr32
	stCall32
	stJump32
	stMax
)

const (
	stModRM        = stOp
	stOp2          = stOp
	stAJump32      = stJump32
	stJumpTblCount = stOp
)

const (
	op2Byte = 0x0f
	opOSize = 0x66
	opCallF = 0x9a
	opRetNI = 0xc2
	opRetN  = 0xc3
	opEnter = 0xc8
	opInt3  = 0xcc
	opInto  = 0xce
	opCallN = 0xe8
	opJmpF  = 0xea
	opIcebp = 0xf1

	escape  = opIcebp
	jumpTab = opInto
)

func moveToFront(table []uint32, pos int, val uint32) uint32 {
	for ; pos > 0; pos-- {
		table[pos] = table[pos-1]
	}
	table[0] = val
	return val
}

func addMTF(mtf []uint32, val uint32) {
	moveToFront(mtf, 255, val)
}

// unfilter is DisUnFilter from DisPack.cpp.
func unfilter(src, dest []byte, memStart uint32) bool {
	if len(src) < stMax*4 {
		return false
	}
	stream := make([][]byte, stMax)
	cur := stMax * 4
	for i := 0; i < stMax; i++ {
		sz := int(binary.LittleEndian.Uint32(src[i*4:]))
		if sz < 0 || cur+sz > len(src) {
			return false
		}
		stream[i] = src[cur : cur+sz]
		cur += sz
	}
	if cur != len(src) {
		return false
	}

	funcTable := make([]uint32, 256)
	nextIsFunc := true
	out := dest[:0]
	destEnd := len(dest)

	for len(stream[stOp]) > 0 {
		start := len(out)
		memory := memStart + uint32(len(out))

		code, ok := fetch8(&stream[stOp])
		if !ok {
			return false
		}
		if code == jumpTab {
			countb, ok := fetch8(&stream[stJumpTblCount])
			if !ok {
				return false
			}
			count := int(countb) + 1
			for i := 0; i < count; i++ {
				ind, ok := fetch8(&stream[stCallIdx])
				if !ok {
					return false
				}
				var target uint32
				if ind != 0 {
					target = moveToFront(funcTable, int(ind)-1, funcTable[int(ind)-1])
				} else {
					target, ok = fetch32B(&stream[stCall32])
					if !ok {
						return false
					}
					addMTF(funcTable, target)
				}
				if len(out)+4 > destEnd {
					return false
				}
				out = appendU32(out, target)
			}
			continue
		}

		if nextIsFunc && code != opInt3 {
			addMTF(funcTable, memory)
			nextIsFunc = false
		}

		if code == escape {
			v, ok := fetch8(&stream[stOp])
			if !ok || len(out)+1 > destEnd {
				return false
			}
			out = append(out, v)
			continue
		}

		if len(out)+1 > destEnd {
			return false
		}
		out = append(out, code)

		flags := 0
		o16 := false
		if code == opOSize {
			o16 = true
			c2, ok := fetch8(&stream[stOp])
			if !ok || len(out)+1 > destEnd {
				return false
			}
			out = append(out, c2)
			code = c2
		}

		if code == opRetNI || code == opRetN || code == opInt3 {
			nextIsFunc = true
		}

		if code == op2Byte {
			c2, ok := fetch8(&stream[stOp2])
			if !ok || len(out)+1 > destEnd {
				return false
			}
			out = append(out, c2)
			flags = int(table2[c2])
		} else {
			flags = int(table1[code])
		}

		if flags == fERR {
			return false
		}

		if code == opCallF || code == opJmpF || code == opEnter {
			if !copy16(&out, &stream[stImm16], destEnd) {
				return false
			}
		}

		if flags&fMR != 0 {
			modrm, ok := fetch8(&stream[stModRM])
			if !ok || len(out)+1 > destEnd {
				return false
			}
			out = append(out, modrm)
			sib := byte(0)
			if flags == fMEXTRA {
				flags = int(tableX[((int(modrm)>>3)&7)|((int(code)&0x01)<<3)|((int(code)&0x08)<<1)])
			}
			if modrm&0x07 == 4 && modrm < 0xc0 {
				s, ok := fetch8(&stream[stSIB])
				if !ok || len(out)+1 > destEnd {
					return false
				}
				out = append(out, s)
				sib = s
			}
			if modrm&0xc0 == 0x40 {
				st := int(modrm&0x07) + stDisp8R0
				if !copy8(&out, &stream[st], destEnd) {
					return false
				}
			}
			if modrm&0xc0 == 0x80 || modrm&0xc7 == 0x05 || (modrm < 0x40 && sib&0x07 == 0x05) {
				st := stDisp32
				if modrm&0xc7 == 5 {
					st = stAddr32
				}
				if !copy32(&out, &stream[st], destEnd) {
					return false
				}
			}
		}

		if flags&fMODE == fAM {
			switch flags & fTYPE {
			case fAD:
				if !copy32(&out, &stream[stAddr32], destEnd) {
					return false
				}
			case fDA:
				if !copy32(&out, &stream[stAJump32], destEnd) {
					return false
				}
			case fBR:
				if !copy8(&out, &stream[stJump8], destEnd) {
					return false
				}
			case fDR:
				var target uint32
				if code == opCallN {
					ind, ok := fetch8(&stream[stCallIdx])
					if !ok {
						return false
					}
					if ind != 0 {
						target = moveToFront(funcTable, int(ind)-1, funcTable[int(ind)-1])
					} else {
						target, ok = fetch32B(&stream[stCall32])
						if !ok {
							return false
						}
						addMTF(funcTable, target)
					}
				} else {
					var ok bool
					target, ok = fetch32B(&stream[stJump32])
					if !ok {
						return false
					}
				}
				target -= uint32(len(out)-start) + 4 + memory
				if len(out)+4 > destEnd {
					return false
				}
				out = appendU32(out, target)
			}
		} else {
			switch flags & fTYPE {
			case fBI:
				if !copy8(&out, &stream[stImm8], destEnd) {
					return false
				}
			case fWI:
				if !copy16(&out, &stream[stImm16], destEnd) {
					return false
				}
			case fDI:
				if !o16 {
					if !copy32(&out, &stream[stImm32], destEnd) {
						return false
					}
				} else if !copy16(&out, &stream[stImm16], destEnd) {
					return false
				}
			}
		}
	}
	return len(out) == destEnd
}

func fetch8(s *[]byte) (byte, bool) {
	if len(*s) < 1 {
		return 0, false
	}
	v := (*s)[0]
	*s = (*s)[1:]
	return v, true
}

func fetch32B(s *[]byte) (uint32, bool) {
	if len(*s) < 4 {
		return 0, false
	}
	v := binary.BigEndian.Uint32(*s)
	*s = (*s)[4:]
	return v, true
}

func copy8(out *[]byte, s *[]byte, destEnd int) bool {
	v, ok := fetch8(s)
	if !ok || len(*out)+1 > destEnd {
		return false
	}
	*out = append(*out, v)
	return true
}

func copy16(out *[]byte, s *[]byte, destEnd int) bool {
	if len(*s) < 2 || len(*out)+2 > destEnd {
		return false
	}
	v := binary.BigEndian.Uint16(*s)
	*s = (*s)[2:]
	*out = append(*out, byte(v), byte(v>>8))
	return true
}

func copy32(out *[]byte, s *[]byte, destEnd int) bool {
	if len(*s) < 4 || len(*out)+4 > destEnd {
		return false
	}
	v := binary.BigEndian.Uint32(*s)
	*s = (*s)[4:]
	*out = appendU32(*out, v)
	return true
}

func appendU32(out []byte, v uint32) []byte {
	return append(out, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}
