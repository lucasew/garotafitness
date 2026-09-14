package dispack

// Opcode tables from third_party/freearc/Compression/DisPack/DisPack.cpp.

const (
	fNM     = 0x0
	fAM     = 0x1
	fMR     = 0x2
	fMEXTRA = 0x3
	fMODE   = 0x3
	fNI     = 0x0
	fBI     = 0x4
	fWI     = 0x8
	fDI     = 0xc
	fTYPE   = 0xc
	fAD     = 0x0
	fDA     = 0x4
	fBR     = 0x8
	fDR     = 0xc
	fERR    = 0xf
)

var table1 = [256]byte{
	fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fNM | fBI, fNM | fDI, fNM | fNI, fNM | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fNM | fBI, fNM | fDI, fNM | fNI, fNM | fNI,
	fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fNM | fBI, fNM | fDI, fNM | fNI, fNM | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fNM | fBI, fNM | fDI, fNM | fNI, fNM | fNI,
	fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fNM | fBI, fNM | fDI, fNM | fNI, fNM | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fNM | fBI, fNM | fDI, fNM | fNI, fNM | fNI,
	fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fNM | fBI, fNM | fDI, fNM | fNI, fNM | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fNM | fBI, fNM | fDI, fNM | fNI, fNM | fNI,
	fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI,
	fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI,
	fNM | fNI, fNM | fNI, fMR | fNI, fMR | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fDI, fMR | fDI, fNM | fBI, fMR | fBI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI,
	fAM | fBR, fAM | fBR, fAM | fBR, fAM | fBR, fAM | fBR, fAM | fBR, fAM | fBR, fAM | fBR, fAM | fBR, fAM | fBR, fAM | fBR, fAM | fBR, fAM | fBR, fAM | fBR, fAM | fBR, fAM | fBR,
	fMR | fBI, fMR | fDI, fMR | fBI, fMR | fBI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI,
	fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fAM | fDA, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI,
	fAM | fAD, fAM | fAD, fAM | fAD, fAM | fAD, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fBI, fNM | fDI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI,
	fNM | fBI, fNM | fBI, fNM | fBI, fNM | fBI, fNM | fBI, fNM | fBI, fNM | fBI, fNM | fBI, fNM | fDI, fNM | fDI, fNM | fDI, fNM | fDI, fNM | fDI, fNM | fDI, fNM | fDI, fNM | fDI,
	fMR | fBI, fMR | fBI, fNM | fWI, fNM | fNI, fMR | fNI, fMR | fNI, fMR | fBI, fMR | fDI, fNM | fBI, fNM | fNI, fNM | fWI, fNM | fNI, fNM | fNI, fNM | fBI, fERR, fNM | fNI,
	fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fNM | fBI, fNM | fBI, fNM | fNI, fNM | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI,
	fAM | fBR, fAM | fBR, fAM | fBR, fAM | fBR, fNM | fBI, fNM | fBI, fNM | fBI, fNM | fBI, fAM | fDR, fAM | fDR, fAM | fAD, fAM | fBR, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI,
	fNM | fNI, fERR, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fMEXTRA, fMEXTRA, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fMEXTRA, fMEXTRA,
}

var table2 = [256]byte{
	fERR, fERR, fERR, fERR, fERR, fERR, fNM | fNI, fERR, fNM | fNI, fNM | fNI, fERR, fERR, fERR, fERR, fERR, fERR,
	fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fERR, fERR, fERR, fERR, fERR, fERR, fERR,
	fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fERR, fERR, fERR, fERR, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI,
	fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fERR, fNM | fNI, fERR, fERR, fERR, fERR, fERR, fERR, fERR, fERR,
	fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI,
	fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI,
	fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI,
	fMR | fBI, fMR | fBI, fMR | fBI, fMR | fBI, fMR | fNI, fMR | fNI, fMR | fNI, fNM | fNI, fERR, fERR, fERR, fERR, fERR, fERR, fMR | fNI, fMR | fNI,
	fAM | fDR, fAM | fDR, fAM | fDR, fAM | fDR, fAM | fDR, fAM | fDR, fAM | fDR, fAM | fDR, fAM | fDR, fAM | fDR, fAM | fDR, fAM | fDR, fAM | fDR, fAM | fDR, fAM | fDR, fAM | fDR,
	fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI,
	fNM | fNI, fNM | fNI, fNM | fNI, fMR | fNI, fMR | fBI, fMR | fNI, fMR | fNI, fMR | fNI, fERR, fERR, fERR, fMR | fNI, fMR | fBI, fMR | fNI, fERR, fMR | fNI,
	fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fERR, fERR, fERR, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI,
	fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI, fNM | fNI,
	fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI,
	fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI,
	fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fERR,
}

var tableX = [32]byte{
	fMR | fBI, fERR, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI,
	fMR | fDI, fERR, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI, fMR | fNI,
	fMR | fNI, fMR | fNI, fERR, fERR, fERR, fERR, fERR, fERR,
	fMR | fNI, fMR | fNI, fMR | fNI, fERR, fMR | fNI, fERR, fMR | fNI, fERR,
}
