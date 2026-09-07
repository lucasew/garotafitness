package magic2

// Static map of official lolz images inside RimWorld setup.exe.
// INV-03: headers, .text strings, and constant pools only. The PEs
// are never mapped executable.
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
// Decode-loop outline (v22c4b x64 .text 0x1656b..0x17920)
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
//	binary FCM (alphabet 2): decode one bit. After the symbol:
//	    cmp ecx, 1 / je  → literal path
//	    add ecx, -2      → match path
//
//	nibble FCM (alphabet 16): two 4-bit symbols make a literal
//	byte. SSE stores of 16-byte prob rows sit next to the
//	0x800000 checks at 0x1656b / 0x16aea (movdqu [rbp], xmm).
//
//	literal: write the byte to the dictionary, advance pos.
//	match:   decode length; if the rep0 bit is set reuse the last
//	         offset, else decode a new offset into rep0; copy
//	         dict[pos-rep0 : pos-rep0+len] forward.
//
//	ROLZ list is allocated in v20d3 ("rolz list buffer") but
//	-rt is dead since ~v19j. ldmf is the optional long-distance
//	path (magic2l only).
//
// DH(n is not stored as a C string in the image (no hit for those
// four ASCII bytes). The on-disk tag is still those four bytes
// (little-endian 0x6e284844) plus ver 0x1f.
