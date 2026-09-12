package garotafitness

import "hash/crc32"

// The RimWorld installer's unarc DLL derives this reflected polynomial in
// CrcGenerateTable (VA 0x61090b44). Its byte update, initialization, and final
// complement are otherwise standard CRC-32. Select it only when it validates
// the archive descriptor; use that same table for all control blocks and files.
var fitgirlCRCTable = crc32.MakeTable(0x0895171b)
