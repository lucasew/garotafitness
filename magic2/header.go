package magic2

import (
	"errors"
	"io"
)

// lolzTag is the 4-byte tag on RimWorld fg-02 and fg-06 solids.
const lolzTag = "DH(n"

// verV22c4b is the byte after the tag on both RimWorld solids
// (FitGirl magic2 = lolz v22c4b).
const verV22c4b = 0x1f

// headerLen is the proven on-disk prefix. Unproven bytes stay in the
// bitstream for decodeStream.
const headerLen = 5

// On-disk layout (lolz v22c4b), live RimWorld solids at volume 0x1F.
//
//	off  n   field     end.  status
//	0    4   tag       ASCII proved: both heads "DH(n"
//	4    1   version         proved: both heads 0x1f; PE "lolz v22c4b [Dec 30 2018]"
//	5    …   bitstream       unproved: ParseHeader does not consume
//
// After +5 (testdata leftover; not a parsed field):
//
//	fg-06  20 00 00 00 02 00 25 00 00 00 fa …
//	fg-02  c0 03 77 00 ac 03 04 61 00 00 07 …
//
// Shared zeros at +8, +13, +14 are two-sample coincidence, not a field.
//
// Guessed, not consumed:
//
//	+5 u32be rANS state  fg-06 0x20000000  fg-02 0xc0037700
//	  Both >= L=1<<23 (PE cmp + shl-8/movzx renormalize). The LE
//	  readings 0x20 and 0x007703c0 are < L, so a 32-bit rANS init
//	  here would have to be big-endian. No init site is pinned to +5.
//
// Disproved on these two heads:
//
//	v20 32-byte options blob (dict<<24, -blo/-bll/-blr/-bm/-bc).
//	  A LE store of dict<<24 is 00 00 00 XX; live first dword is not.
//	  The two solids do not share a 32-byte options prefix.
//	  optionTable32 in tables.go is the in-memory default block
//	  before "available options", not this prefix.
//	Plain sizes 93116, 430889, 2895, 6, 190, 13622, 414176, count 5,
//	  and the five IEEE CRCs: none appear as u16/u32/u64 LE/BE in the
//	  fg-06 solid (whole csz). orig/csz live in the ArC directory.
//	Method strings are "magic2" and "srep:m3yf+magic2" (no :d/:blo).
//
// cls-magic2 ReadFile(9) when [ctx+0x254]==1 is the CLS transfer
// record, not this prefix.

// Header is the identified prefix of a lolz v22c4b stream.
type Header struct {
	Tag [4]byte
	Ver byte
}

// ParseHeader reads the 5-byte tag+version prefix.
func ParseHeader(r io.Reader) (Header, error) {
	var b [headerLen]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return Header{}, err
	}
	if string(b[:4]) != lolzTag {
		return Header{}, errMagic
	}
	if b[4] != verV22c4b {
		return Header{}, errVersion
	}
	var h Header
	copy(h.Tag[:], b[:4])
	h.Ver = b[4]
	return h, nil
}

var errVersion = errors.New("magic2: unsupported version")
