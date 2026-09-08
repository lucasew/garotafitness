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
//	state is a 32-bit rANS register. First payload dword after
//	DH(n 0x1f is that state, big-endian (fg-06 0x20000000,
//	fg-02 0xc0037700).
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
//	(al=2). First bit 1 = match @ 0x14002a539 → decodeMatch.
//	Literal @ 0x14002b7e9 mixes two 16-sym CDFs:
//	mixed = (w*A + (uint16)(0-w)*B)>>16 (pmulhuw+paddw),
//	find on mixed, w -= w>>4; if freqA>=freqB { w += 0xfff }.
//	Hi adapt >>6 toward 0x1f00; lo >>7 toward 0x1c00.
//	Store: (hi<<4)|lo at [dict+pos] (0x14002bcf7).
//
//	See fcm.go for getBit / getNibble.
//
// DH(n is not stored as a C string in the image (no hit for those
// four ASCII bytes). The on-disk tag is still those four bytes
// (little-endian 0x6e284844) plus ver 0x1f.
