package corpus

import (
	"io"
	"os"
	"testing"

	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/test"
)

func Dir(t testing.TB) string {
	t.Helper()
	dir := os.Getenv("GAROTAFITNESS_CORPUS")
	if dir == "" {
		t.Skip("set GAROTAFITNESS_CORPUS")
	}
	return dir
}

func Open(t testing.TB) *lewpath.Root {
	t.Helper()
	r, err := lewpath.Open(Dir(t))
	if err != nil {
		t.Skip("corpus not mounted")
	}
	test.CloseOnCleanup(t, r)
	return r
}

func File(t testing.TB, name string) io.ReadSeeker {
	t.Helper()
	f, err := lewpath.New(name).Open(Open(t))
	if err != nil {
		t.Skip("corpus not mounted")
	}
	test.CloseOnCleanup(t, f)
	rs, ok := f.(io.ReadSeeker)
	if !ok {
		t.Fatalf("%s: %T is not a ReadSeeker", name, f)
	}
	return rs
}

func ReadFile(t testing.TB, name string) []byte {
	t.Helper()
	b, err := lewpath.New(name).ReadFile(Open(t))
	if err != nil {
		t.Skip("corpus not mounted")
	}
	return b
}
