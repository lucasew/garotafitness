package magic2

// Static map of official lolz images inside RimWorld setup.exe.
// INV-03: headers, .text strings, and constant pools only. The PEs
// are never mapped executable. Official C++ was never released;
// the reconstructed kernel lives in guest/ and compiles to
// magic2dec.wasm (same wrap as srep: emcc + wazero).
//
// setup.exe overlays
//
//	PE of the Inno stub ends near 0x22600.
//	zlb@0x22600  LZMA2 → 8_306_006 bytes. Holds the codec images.
//	zlb@0x419133 LZMA2 → 187_318 bytes. cls-uni / facompress only.
//	Inno Setup Data @0x42c8c0 (LZMA1) names the extracted files.
//
// Inno file names (FitGirl rename of ProFrager lolz)
//
//	cls-lollypop.dll / cls-lollypop_x64.exe / cls-lollypop_x86.exe
//	    → lolly v20d3 (Oct 30 2017). Older kernel, still in the zlb.
//	cls-magic2.dll / cls-magic2_x64.exe / cls-magic2_x86.exe
//	    → lolz v22c4b, FitGirl method magic2 (non-ldmf).
//	cls-magic2l.dll / cls-magic2l_x64.exe / cls-magic2l_x86.exe
//	    → same v22c4b image, FitGirl method magic2l (ldmf).
//
// Images inside zlb@0x22600 (file offset in the plain overlay)
//
//	+0x269195  x64  138_240 B  entry 0x82c0   lolly v20d3. no exports
//	+0x28ad95  x86  128_000 B  entry 0x8240   lolly v20d3. no exports
//	+0x315d95  x86  cls-uni.dll               export ClsMain
//	+0x319d95  x64  343_040 B  entry 0x43914  v22c4b magic2. no exports
//	+0x36d995  x86  312_320 B  entry 0x3d43b  v22c4b magic2. no exports
//	+0x3bdd95  x64  (copy of +0x319d95)       v22c4b magic2l
//	+0x411995  x86  (copy of +0x36d995)       v22c4b magic2l
//
// The codec EXEs export nothing; they are CLS hosts. The small
// cls-uni.dll in front of each pair exports ClsMain.
//
// Key strings (v22c4b .text; image has no .rdata)
//
//	"lolz" / " v22c4b [Dec 30 2018  16:44:57]"
//	"lolz (ldmf)" ".ldmf"
//	"available options: \n"
//	" for mtt1:  -MaxThreadsUsage, -MaxMemoryUsage\n"
//	" for ldmf1: -ldmfTempPath, -ldmfMaxMemoryUsage, -ldmfDeleteTmp"
//	"transfer data options: -Bufsize, -transfer_ReadBufSize, -transfer_WriteBufSize"
//
// Key strings (v20d3 only)
//
//	"lolly" "lolz" " v20d3 [Oct 30 2017  17:30:22]"
//	"fcm nibble probs" "fcm binary probs" "dictionary" "rolz list buffer"
//
// Immediates
//
//	1<<23 = 0x800000   rANS L. Hundreds of cmp r32, L / jb in .text.
//	0x3420000          VirtualAlloc size when width==4 (nibble FCM).
//	0x992200           VirtualAlloc size when width==2 (binary FCM).
//	x86: cmp eax,2 / je binary; cmp eax,4 / add edi, 0x3420000.
//	x64: same sizes passed to VirtualAlloc (r8d=0x3000, r9d=4).
//
// 16 / 32-byte options
//
//	A 64-byte block ends at "available options": optionTable32,
//	then a 16-byte repeat of that first row, then optionDefaults16
//	= uint32{2,4,4,4}. That matches ProFrager's published defaults
//	-pc2 -bc4 -bm4 -blr4. On/off flags (dt, cm, ldmf, mtt, al, …)
//	are the 16-byte flag rows. Stream prefix is still only DH(n +
//	0x1f; these bytes are the in-memory defaults, not a second
//	on-disk header.
//
// Decode-loop outline (v22c4b x64 .text 0x1656b..0x17920 is the
// FCM extra-class pair 0x140017410 / 0x1400179b0. LZ match class
// + offset/length is decodeMatch @ 0x140039ae0; see match.go.)
//
//	PE map (x64): ImageBase 0x140000000, .text file 0x400 ↔
//	RVA 0x1000 ↔ VA 0x140001000. Entry 0x140043914.
//	x86 twin: ImageBase 0x400000, .text file 0x400 ↔ RVA 0x1000.
//
//	state is a 32-bit rANS register. 0x14002973b does
//	movl (%src),%r10d into 0xe4(%rsp) — little-endian
//	(fg-06 0x00000020, fg-02 0x007703c0). No renorm runs
//	between that store and the first token at 0x140029e04;
//	the first getBit uses the raw dword (slot 0x20).
//	0xb40/0xb48 is a second rANS used by decodeOpt @
//	0x14003afc0 (scale-15 bit + 16-sym, adapt >>5 / 0x2980,
//	ids at 0xa6a7/0xa6b7). Init 0x14003edb0: p0=0x4000,
//	CDF i*0x800 at +0x10/+0x32, [model+0x4a58]=0.
//	Caller 0x140028c3d (rcx=obj, rdx=&opt; E8 thunk
//	0x140063002). After return: movb al, 0x64(%r12) so
//	opt is hist-row n; then 0x5a98[opt] → +0xc39 (pc mask)
//	when opt < 0x25. Only +0xb38 write is 0x14002908e:
//	movq %r9, 0xb38(%r12) with r9 = stream.base+pos (pos
//	zeroed at 0x140028c16). After one fg-06 decodeOpt
//	bit+16-sym, 0xb48 is payload+7 (state 0x20, renorm
//	02 00 25, sym 0 → opt=0); LZ reloads a fresh LE
//	dword from +0xb38 (stream pos 0), not +0xb48.
//
//	renorm:
//	    while state < 1<<23 {
//	        state = state<<8 | *src++
//	    }
//	    // cmp edx/r10d/r15d, 0x800000; jb
//	    // shl r32, 8; movzx tmp, byte [src]; or r32, tmp; inc src
//
//	slot is always state & (M-1), never state % M.
//	Nibble / 8-sym: M=1<<15, scale 15, freq is a CDF of uint16.
//	  find first i where int16(cdf[i]) > int16(slot) via
//	  pcmpgtw/packsswb/pmovmskb + sentinel (0x80 or 0x10000) + bsf.
//	  state' = (cdf[i]-cdf[i-1])*(state>>15) + slot - cdf[i-1]
//	  adapt cdf += (target[sym]-cdf) >> 7   (16-sym, psraw $7,
//	  targets at VA 0x140001c00) or >> 6 (8-sym, VA 0x1400010e0).
//	Binary: M=1<<14, scale 14, single p0 (x64 0x14001af4a,
//	x86 0x415af5). if slot < p0 { state = quo*p0+slot;
//	p0 += (0x4000-p0)>>4 } else { state -= p0*(quo+1); p0 -= p0>>4 }.
//
//	Nibble context hash (0x14001adb0):
//	    h0 = bitlen8(u8((abs(w08-w0c)+abs(w10-w14))>>1))
//	    h1 = bitlen8(u8(abs(w20-w24)))
//	    idx = h0*288 + h1*32
//	Mixer hash (0x140017410):
//	    absb = abs_bytes(pack(w20,w10)-pack(w24,w14))
//	    idx = bitlen(b0|b1|b2)<<7 + bitlen(b5)<<4 + bitlen(((w20+w24)>>9)&0x3f)<<10
//	o1 selectors from -pc2 -bc4 -bm4 -blr4 / -blo8 -bll8: see fcm.go.
//
//	binary FCM (alphabet 2): decode one bit. After the 8-sym token:
//	    cmp ecx, 1 / je  → literal path
//	    add ecx, -2      → match path
//	    (0x14001775e)
//
//	nibble FCM (alphabet 16): two 4-bit symbols make a literal
//	byte. SSE stores of 16-byte prob rows sit next to the
//	0x800000 checks at 0x1656b / 0x16aea (movdqu [rbp], xmm).
//
//	literal: write the byte to the dictionary, advance pos.
//	match:   16-sym class at 0x140039ae0 (add 0x10000 + bsf,
//	         adapt >>6 toward 0x140001f00, CDF at model+0x1240,
//	         jmp 0x14000a8c0).
//	         cmp r15, 0xb / ja 0x14003a3a0: cls 12-15 are
//	         reps[cls-12] rotate-to-front, length 2
//	         (movl $2, ebx at 0x14003a3f6), esi via 0xa280.
//	         class 0 = reuse *rep0 (0x14003a374). There is no
//	         standalone binary bit "0=new / 1=rep0" — that
//	         polarity is not in the PE. class 4..10 index
//	         extraBitsA690 at 0x14000a690 (cls-4) and rotate
//	         the recent-offset array. New distances come from
//	         classes 1/2/3/11; min offset 1.
//	         length @ 0x140039f87: n = 8-sym + 3, escape 10
//	         then +extra (add rbx, 0xa). Class 2 is 3+bit.
//	         -cm1 mixer sits on the lit/match bit (v20
//	         0x14001387f cmp edx,9): a second binary FCM bit
//	         only when the mixer table is below 9.
//	         The 8-sym cmp ecx,1 / add ecx,-2 at 0x14001775e
//	         is the FCM extra-class loop, not the LZ class.
//
//	Nibble at 0x14001adb0 is 9-sym: pmovmskb + add 0x200 + bsf
//	(alphabet 0..8 = bitlen8). Adapt psraw $6 toward 0x140001aa0.
//	CDF row at model+0xb8c00 + h0*288 + h1*32.
//	Nibble extras (0x14001af1b..0x14001b14f): bsf==1 → r10=r13=0.
//	Else base = model+(h1<<11)+((sym-1)<<8)+0xb9620. Tree walk:
//	rdx=0; for i in 0..sym-1 { bit0=getBit(base+i*32+8*rdx);
//	bit1=getBit(that+2+2*bit0); r10=r10*2+bit0; r13=r13*2+bit1;
//	rdx=bit1+2*bit0 }. Mix at 0x14001b15f stores +0x18 only.
//	Caller 3-stage qword rotate: w08←w10←w20←(+0x18).
//
//	ROLZ list is allocated in v20d3 ("rolz list buffer") but
//	-rt is dead since ~v19j. ldmf is the optional long-distance
//	path (magic2l only).
//
//	v22 LZ token @ 0x140029e04: scale 14 adapt >>5. Mixer byte
//	from [obj+0x70][hist] compared to 0x63 (not v20's 9). If
//	below, a second >>5 bit on p0+2: 0=still literal, 1=DXT
//	(al=2). tok==2 is DXT only when mixer<=8 (jmp 0xa880);
//	mixer>8 goes to the error path at 0x14002a620.
//	Init 0x140028b67 sets +0x70 = 0x140005ac0 (mostly 99;
//	nonzero slots are DXT classes 0..13). hist =
//	0x140005b40[min(+0x64,0x24)][pos & +0xc39]. Ctor +0x64=0
//	and 0x5a98[0]=0, so hist=0, mixer=99, no second bit.
//	0xa2b0/0xa260 update esi (p0 context), not the 99-compare.
//	p0 cell is model+0xb50+(hist<<6)+esi*4; ctor +0x60=0 so esi
//	starts at 0. After lit esi=a260[esi]; after match a2b0[esi].
//	First bit 1 = match @ 0x14002a539 → decodeMatch
//	(class CDF model+0x1240+hist*544+esi*34, adapt >>6 / 0x1f00).
//	Class 0 length is the immediate 1 (0x14003a396), no 8-sym.
//	Classes 4-9: a690[cls-4] indexes the recent-offset array
//	(slots 0..3, 17, 18), not extra bits (0x140039dd0).
//	Length 8-sym at +0x3ffea, add 0x100+bsf, adapt >>7 / 0x2f20.
//	Class 1 new offset: two 8-sym, rbp = s1 + s0*8, insert at +0x44.
//	  s0 CDF model+hist*18+0x1c560, adapt >>6 toward 0x1400011a0.
//	  s1 CDF model+hist*288+(prev>=10?144:0)+s0*18+0x1ca82,
//	  adapt >>7 toward 0x140002f20.
//	  lea rbp,[rcx+rbp*8] (0-based). Next 5 bytes f9 99 03 00 00
//	  desync llvm-objdump; the ldmf twin at 0x14003626f is mov edi,2
//	  so length is the immediate 2. Store: movdqu [+0x44]->[+0x48],
//	  movl ebp,[+0x44] (reps[17]).
//	Class 2/3 new offset: call 0x140073fc9 / 0x14007396f (past
//	  .text VSz). x86 twins 0x45ed6f / 0x45e660 are also past.
//	  Same shape as in-image 0x140036b00: 16-sym slot (escape 15).
//	  Both cls2 and cls3 load +0xbe0 from 0xa6e7 (8-byte
//	  {base56, nbits8}). +0xc00/0xa706 has no decode load.
//	  nbits<=5: nbits scale-14 bits, offset=base+extra (0 legal).
//	  nbits>5: s2<<max(nbits,9)-4, raw<<(5) if nbits>9,
//	  then s3 16-sym + bit; offset=rdi+2*s3+bit. No +1.
//	  cls2 length: binary at +0x21ca2+(bsr(off)+1 & ~3), +3.
//	  cls3 length: 0x14006f962 / add $5. A at +0x276a6+
//	  (hist==0)*34+(bsr>>2)*68, B at +0x311c6+(bsr>>2)*34.
//	  Same mix16+esc helper as the length-8 escape; not 8-sym+5.
//	Length 8-sym packed row is esi*32+hist*2+(idx!=0), not esi*16.
//	Length escape 0x140039f8e: cmp $0xa / call 0x14006f8ad / add $0xa.
//	  In-image clone 0x140035ebc → 0x14006b781 (same rel32).
//	  A = +0x4a42a + esi*17<<9 + hist*34 + (idx!=0)*544
//	  B = +0xe562a + (idx!=0)*34. Init uniform i*0x800
//	  (0x14003e612 / 0x14003e662). Caller writes only the 5th
//	  arg (B pointer), so this is mix16 + escape-15, not a6e7
//	  nbits (that floods and cannot hit the 6-byte CRC).
//	  First fg-06 escape: slot 0x7a03 → 15, esc slot 0x4a03 → 9,
//	  extra 24, m=34. Without the esc-15 arm, first cls11 is at
//	  n=41; with it, n=63. c34=0 cannot emit cls11.
//
//	RetDec v5.0 (mise exec github:avast/retdec) on the carved PEs
//	(/tmp/retdec-lolz/*.c). Confirms, does not change, the map:
//	  decode_int 0x140036b00: (obj, state*, src**, A, widx, lookback).
//	    B = A+0x44+bitlen(lookback)*34; w = A+0x484+widx*2.
//	    nbits<6 → 0x14006d0d4; else s2 at A+s*34+0x4a4 >>6/0x1f00,
//	    shift = nbits>=9 ? (nbits+60)&63 : 5; nbits>=10 raw<<5;
//	    s3 at +s2*1088+0x8e4 >>7/0x1c00; bit scale-14 at +0x4ce4;
//	    return base + (s2<<sh) + raw + 2*s3 + bit.
//	  decodeMatch class: A at +0x1240+esi*544+hist*34, unmixed,
//	    jmp 0xa8c0. Length 8-sym ln==7 → 0x14006f8ad.
//	  cls3 length: unknown_14006f962(A=+0x276a6+(hist==0)*34+
//	    68*(bitlen>>2), B=+0x311c6+34*(bitlen>>2)).
//	  cls2 length: p0 +0x21ca2+hist*32+(hist==0)*2+(bitlen&~3).
//	  ctor +0xc34 (3124): lists at +0xc90 and model at +0xb68
//	    (2920) only if c34!=0. Main model +0xb50 always
//	    VirtualAlloc 0xe5fe8. Literal mix matches pe_notes
//	    (hi A hist*2176, B +0x26c80, w +0xc1e80+prev>>bm*2336).
//	  x86 twin 0x42f200: same 0x4a42a+(idx?544)+esi*0x2200.
//	Class 10: 16-sym at +0x364ca, then a697[sym] as rep index
//	  ({4,5,6,7,8,9,10,11,12,13,14,15,16,19,20,21}).
//	Literal @ 0x14002b7e9 mixes two 16-sym CDFs (row stride 34):
//	hi A model+(hist*17<<7)+(prev>>blr)*34
//	hi B model+(hist*17<<9)+(prev>>blo)*34+0x26c80
//	hi w model+0xc1e80+(prev>>bm)*0x920+hist*32+esi*2
//	lo A model+(hist*17<<6)+ctx_lo*34+0xe6680
//	lo B model+(hist*17<<9)+(prev>>bll)*34+0xf9cc0
//	lo w model+0x194ec0+(prev>>bm)*0x920+hist*32+esi*2
//	defaults -blr4 -blo8 -bll8 -bm4. Init CDFs 0x1bc0 (i*0x800),
//	weights 0x8000 (0xa4b0). mixed = (w*A+(uint16)(0-w)*B)>>16,
//	w -= w>>4; if freqA>=freqB { w += 0xfff }.
//	Hi adapt >>6 toward 0x1f00; lo >>7 toward 0x1c00.
//	Store: (hi<<4)|lo at [dict+pos] (0x14002bcf7).
//
//	cls11 @ 0x140039c1d uses model +0xb68 (allocated only if
//	opt[+0x14]/+0xc34 != 0). First helper is a decode_int cousin
//	(mix16+esc, CDF esi*17<<9+hist*34); second is 0x140036730
//	(16-sym at min(bitlen(first),7), then +0xba0 nbits).
//	ROLZ: slot = cursor[prev] + ~idx, wrap by cap; store pos,
//	lookup ~list[slot].
//
//	Who writes +0xc34: ONLY 0x14003b934 movb 0x14(%rbx) (options
//	parse). All other 0xc34 hits are cmpb $0. Parse has no
//	in-image caller (packed). fg-06 bytes after DH(n)+0x1f are
//	NOT that blob (would give width=156, blo=27). Stream does
//	not set +0xc34. 0x14006f513 / 0x14006f7ab are past
//	SizeOfImage 0x57000 — not in this PE.
//
//	decodeOpt common path (s<15) leaves the image:
//	  0x14003b177 jmp 0x140075842          (renorm island)
//	  then the in-image tail at 0x14003b2c6 stores id and
//	  calls TWO more integers (always, unless id==0x3f):
//	    0x140075b29  A=model+0x54     stack {0, nbits=6, 1}
//	    0x140075b60  A=model+0x1900   same stack
//	    0x140075ba2  A=model+0x31ac   only if kA6C7[id]!=0
//	  x86 twins 0x4680ea / 0x468122 / 0x46816b, also past
//	  SizeOfImage 0x4f000. Init 0x14003edb0 zeros +0x54 and
//	  +0x1900; CDF rows for those helpers start at +0x80 /
//	  after +0x190c. Return values become opt[4] / opt[8].
//	  In-image cousin 0x14003b430 is the same model:
//	    +0x08 ctx, +0x0c p0 (scale-15 >>4), +0x2c 16-sym,
//	    +0x46c escape, +0x8ac bit tree, nbits=6 → mask 0x3f.
//	  flag=1 skips the optional presence bit (else fg-06
//	  extraA=extraB=0). With flag=1: extraA=16, extraB=18.
//	After +0xb38 is set to stream.base+pos0, default
//	+0xc24==0 (options blob[4], only written by parse
//	0x14003b901; stream path leaves VirtualAlloc 0)
//	calls 0x14005fc5d(obj, opt[4]) then a packed jmp.
//	x86 twin 0x452748. 0x14005fc5d is past SizeOfImage
//	0x57000. After return the parent reads +0x58 (bytes
//	written). Parent already did stream.pos += extraB
//	while leaving +0xb38 at the old pos (the prefix
//	window). 0x14005fc5d can emit extraA dest bytes from
//	that window (parent later reads +0x58; LZ 0x140029700
//	continues at dest[+0x58] with prev=dest[+0x58-1] and
//	a fresh LE dword from +0xb38). extraB=18 on fg-06:
//	LZ-at-18 first byte is 0x7c '|'.
//	Window hex 20 00 00 00 02 00 25 00 00 00 fa 03 72 05
//	32 1a a8 0f. Fresh rANS there is state 0x20, so every
//	default-init model yields first symbol 0 (16-sym /
//	nibble-pair / FCM r10 / forced LZ lit / b430 skip).
//	Raw extraA slice starts 0x20. Leftover option rANS
//	(x=0x02004000 off=9) first bits are 1 (0x40/0x80).
//	None of those cousins emit '[' / ';' / "[S".
//
//	pe_big1.bin (/tmp/lolzpe/pe_big1.bin, zlb+0x2aa195): 7-section
//	x64, entry 0x1ccf4, SizeOfImage 0x4a000. SREP (FreeArc), not
//	lolz/lolly. "Not an SREP compressed file", transfer_ReadBufsize.
//	0x14005fc5d is past 0x4a000 — no prefix writer there.
//	lolly-v20d3 (/tmp/lolly-v20d3-x64.bin = pe_2527637): 3-section
//	unpacked v20d3, SizeOfImage 0x24000. Default +0x964==0 cousin
//	is call 0x14001ecb4(obj, extraA); that VA is 2 bytes into a
//	movdqu of CDF-init 0x14001ecb0, not a dest writer. +0xc24!=0
//	cousin 0x1400185fc is mid FCM-alloc (0x140018320). v20 does
//	not map 0x14005fc5d. v22 2-section image is section-merged
//	(POGO: .text+.pdata+.rdata), RWX .text, no TLS; far E8/E9
//	(0x14005fc5d / 0x140075842) are past SizeOfImage 0x57000.
//	No zlib/UPX/aPLib overlay. Static unpack cannot recover the
//	body without executing the PE (INV-03). Guest wires the
//	in-image 0x14003b430 cousin extraA times on the +0xb38 window.
//	Init 0x14003edb0 (thunk 0x140066d2e on obj+0xb90, size
//	0x4a5c) copies uniform i*0x800 from 0x140002940 and
//	p0=0x4000 from 0x14000a4c0. Extra models at +0x54 /
//	+0x1900 start uniform; extraA's 16-sym adapts cdf[0]
//	toward 4 (ctxa=5). x86 0x452748 is (eax=obj,
//	edx=extraA) only — model is obj+0xb90. Leftover option
//	rANS + adapted cdf[0] nibble-pairs start 0x77 'w',
//	header p0 (now 0x4200) 8-bit starts 0x22. Still not INI.
//	0x14003b430 does not load stack nbits (mask 0x3f is
//	hardcoded). Init stride 0x44 is one ctx's 16-sym+escape
//	pair; the live row is ctxa*34 at A+0x2c. In-image dest
//	stores are only LZ lit 0x14002bcf7, match copies, and
//	raw memcpy 0x140028d40. 6-bit leftover dest (nbits=6)
//	starts 0x20 / '@' / '`' / '!' / 'A', not '['.
//	0x140036b00 extraA dest (lookback=prev): leftover first
//	mix-16 is s=8, kA6E7 nbits=11 base=2080, low 8 = 0x20
//	space. Window first s=0 nbits=5 base=0 → 0x01. Not INI.
//	In-image LZ @ 0x140029e04 with that reload (state 0x20,
//	p0=0x2000, uniform mix) is forced to first byte 0x04
//	(hi slot 0x0025, lo slot 0x2025). That cannot be INI
//	text; the hole is still 0x14005fc5d (x86 0x452748,
//	past .text), not another nibble-row tweak.
//
//	fg-06 ArC order (one magic2 solid @0x1F csz=93116):
//	3 storing dirs, then steam_emu.ini 2895 fb362bfa,
//	steam_appid.txt 6 f75982bb, two VDFs, rimworld.x3.
//	CRC windows [0:2895] / [2895:2901] are the right members.
//	steam_emu.ini is first; 04 04 08 06 cannot be its text.
//
//	See fcm.go for getBit / getNibble.
//
// DH(n is not stored as a C string in the image (no hit for those
// four ASCII bytes). The on-disk tag is still those four bytes
// (little-endian 0x6e284844) plus ver 0x1f.
