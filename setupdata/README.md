The reconstruction metadata reader parses the IFPS container used by Inno's
compiled PascalScript. Container types, procedure declarations, operands, and
branch offsets follow `TPSExec.LoadData`, `ReadVariable`, and the opcode tables
in [RemObjects PascalScript](https://github.com/remobjects/pascalscript).

Constant propagation starts at `CurStepChanged(ssInstall)`, follows control-flow
edges, joins values at branches, and inspects referenced procedures. It records
ISDone extraction and command calls, plus file cleanup. Known string helpers
preserve Inno path constants instead of resolving them to host directories.
Arguments needed for reconstruction must resolve to constants. No DLL import
is resolved, and no installer script or executable is run.

Inno file-table strings provide the installed checksum file's destination.
Archive records keep their source path, destination, subdirectory filter, and
component flag. The extractor uses these records to separate installed and
temporary files and reads compression and movement recipes from the selected
volume members. Supported commands are mapped to in-process transformations.
There is no game-name or installed-file table in this reader.
