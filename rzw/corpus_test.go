package rzw

import (
	"io"
	"testing"

	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/test"
)

const rimworld = `/media/downloads/TORRENTS/RimWorld [FitGirl Repack]`

func openCorpus(t *testing.T) *lewpath.Root {
	t.Helper()
	r, err := lewpath.Open(rimworld)
	if err != nil {
		t.Skip("corpus not mounted")
	}
	test.CloseOnCleanup(t, r)
	return r
}

func openCorpusFile(t *testing.T, name string) io.ReadSeeker {
	t.Helper()
	f, err := lewpath.New(name).Open(openCorpus(t))
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

func readCorpusFile(t *testing.T, name string) []byte {
	t.Helper()
	b, err := lewpath.New(name).ReadFile(openCorpus(t))
	if err != nil {
		t.Skip("corpus not mounted")
	}
	return b
}
