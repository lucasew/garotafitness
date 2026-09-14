# ADR-0007: packages by pipeline phase

Status: accepted

Stream atoms live in `github.com/lucasew/garotafitness/stream/<algo>`.
Reconstruction programs live in
`github.com/lucasew/garotafitness/reconstruct/<name>`.
`NewReader(io.Reader) (io.ReadCloser, error)` is the decoder contract.
Guests are `go:embed` inside that package. Official C++ lives in
`third_party/<name>/`. Compilers are mise `conda:`.

`Algo` names a volume-pipeline atom. x2, x3, x5, fgpack, and fsb are
reconstruction programs, not `Algo` values. `setupdata` may still scan
those strings in installer text.

Child packages do not edit `decode.go`, `algo.go`, `extract.go`,
`method.go`, or `SPEC.md`. The root package adds one `Decode` switch
arm after a stream package lands. `4x4` parses its own Params and takes
a `Decode`-shaped func so it does not import the root package.

Rejected: `internal/algo/`, `init()` registration, a shared `guests/`
tree, nesting `stream/entropy/` and `stream/media/`, a second Filter
type.

Supersedes ADR-0005.
