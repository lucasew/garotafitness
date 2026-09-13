The decoder is a static C translation of the MPZ 5.4.5.1 implementation in
RimWorld's MpzSlimmer.dll. No PE code is loaded or interpreted during extraction.
The WASM module contains compiled C routines with fixed control flow, bounded
model memory, and memory streams in place of CRT file operations.

The installer stores this DLL at offsets 0x46f873–0x49b873 of its decompressed
payload. Restore Inno's E8/E9 filter before disassembly. `lift.py` checks the
restored DLL's SHA-256 and rejects other versions. The translation includes:

- Stream initialization at 0x10016dd0, the frame/literal loop at 0x100133d0,
  and final literal copying at 0x10016c7c.
- Probability table initialization at 0x100158cc, which normally runs during
  DLL initialization.
- The reachable MP3 model, Huffman reconstruction, and frame writing routines.
- Fixed input/output callbacks at 0x10018e24 and 0x10018e10.

`mpz-tables.inc` contains constant data, including MP3 Huffman tables. Dynamic
probability tables are initialized by the translated routines. Arithmetic
uses explicit 32-bit wraparound. Dead arithmetic flags are removed by the
static translator. Unsupported callbacks and invalid memory accesses abort
the WASM call; wazero reports an error to the Go reader.

To reproduce the C translation with the local corpus and LLVM tools:

```sh
llvm-objdump -d --no-show-raw-insn MpzSlimmer.dll > mpzdll.asm
python3 mpz/guest/lift.py MpzSlimmer.dll mpzdll.asm mpz/guest
make -C mpz
```

`decoder.c` and the tables are checked in, so ordinary builds need neither the
installer nor the offline translation tools. Compile with the toolchain in
`mise.toml`. The Go reader separately supports the 5.4.5.0 complemented literal
format, bounds input/output sizes, and checks the reconstructed output length.
