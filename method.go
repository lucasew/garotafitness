package garotafitness

import (
	"fmt"
	"strings"
)

// Atom is one step in a FreeArc pipeline: Algo plus its option string.
type Atom struct {
	Algo   Algo
	Params string
	token  string
}

// String reconstructs the on-disk token.
func (a Atom) String() string {
	if a.token != "" {
		return a.token
	}
	if a.Params == "" {
		return a.Algo.String()
	}
	return a.Algo.String() + ":" + a.Params
}

// Known reports whether the atom maps to a defined Algo.
func (a Atom) Known() bool {
	return a.Algo.Known()
}

// Pipeline is a FreeArc method string, last atom decompressed first.
type Pipeline []Atom

func (p Pipeline) String() string {
	if len(p) == 0 {
		return ""
	}
	parts := make([]string, len(p))
	for i, a := range p {
		parts[i] = a.String()
	}
	return strings.Join(parts, "+")
}

// Last is the outer decompressor. Zero Atom if p is empty.
func (p Pipeline) Last() Atom {
	if len(p) == 0 {
		return Atom{}
	}
	return p[len(p)-1]
}

// Algos returns unique Algos in pipeline order.
func (p Pipeline) Algos() []Algo {
	seen := make(map[Algo]struct{}, len(p))
	out := make([]Algo, 0, len(p))
	for _, a := range p {
		if _, ok := seen[a.Algo]; ok {
			continue
		}
		seen[a.Algo] = struct{}{}
		out = append(out, a.Algo)
	}
	return out
}

// ParsePipeline splits a FreeArc compressor string.
func ParsePipeline(s string) Pipeline {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "+")
	out := make(Pipeline, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, parseAtom(p))
	}
	return out
}

func parseAtom(s string) Atom {
	name, params, _ := strings.Cut(s, ":")
	return Atom{
		Algo:   ParseAlgo(name),
		Params: params,
		token:  s,
	}
}

func unknownEncoderError(a Atom) error {
	return fmt.Errorf("unknown encoder %s", a.String())
}
