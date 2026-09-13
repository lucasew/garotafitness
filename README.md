# garotafitness

Extract the local RimWorld FitGirl repack into its installed game tree:

```sh
go build -o bin/garotafitness ./cmd/garotafitness
bin/garotafitness extract '/path/to/RimWorld [FitGirl Repack]' /path/to/output
```

Keep `setup.exe`, the required `fg-*.bin` volumes, and `MD5/fitgirl-bins.md5`
together in the source directory. The optional soundtrack volume is extracted
when present. `setup.exe` supplies metadata and the installed-file checksum
manifest; it is read as data. Extraction runs entirely inside the Go process.

The RimWorld reconstruction restores the inner archive, four DLC asset bundles,
and the update files. It checks archive CRCs, patch checksums, and every installed
file's MD5 before writing the final tree. Temporary reconstruction files stay in
memory. The soundtrack's MPZ decoding takes several minutes.

Compiled WASM modules are included. To rebuild the reconstruction modules, use
the compilers in `mise.toml` and run `mise run build:reconstruction`. The MPZ
translation is documented in [mpz/guest/README.md](mpz/guest/README.md), and the
RTPatch format in [x3/README.md](x3/README.md).

Run regression tests with `go test -p 1 ./...`. With the corpus mounted at the
path used in `arc_test.go`, the full extraction test can also be enabled:

```sh
GAROTAFITNESS_FULL_EXTRACT=all go test . -run '^TestExtractInstalledRimWorld$' -timeout=1h
GAROTAFITNESS_FULL_EXTRACT=required go test . -run '^TestExtractInstalledRimWorld$' -timeout=1h
```
