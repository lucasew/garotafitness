# ADR-0005: one Algo package at module root

Status: superseded by ADR-0007

Each Algo lives in `github.com/lucasew/garotafitness/<algo>`.
`NewReader(io.Reader) (io.ReadCloser, error)` is the only decoder
contract. Guests are `go:embed` inside that package. Official C++
lives in `third_party/<name>/` (git submodule). Compilers are mise
`conda:`.

Child packages do not edit `decode.go`, `algo.go`, `extract.go`,
`method.go`, or `SPEC.md`. The root package adds one `Decode` switch
arm after a package lands. `4x4` parses its own Params and takes a
`Decode`-shaped func so it does not import the root package.

Rejected: `internal/algo/`, `init()` registration, a shared
`guests/` tree, fetching `/tmp` clones at build, a second Filter type
for x2/x3/x5/fgpack.

## Ownership (wave 1)

| Agent | May write | Must not write |
|-------|-----------|----------------|
| srep | `srep/`, `third_party/srep/` | root `*.go`, other algo dirs |
| freearc-builtins | `fourx4/`, `delta/`, `dispack/`, `third_party/freearc/` | `srep/`, CLS packages |
| magic2 | `magic2/` | PE exec, other packages |
| mpzz | `mpzz/` | other packages |
| rzw | `rzw/` | other packages |

Wave 2 (after a solid yields those members): `x2/`, `x3/`, `x5/`, `fgpack/`, `mpz/`.
