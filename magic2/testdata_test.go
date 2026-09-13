package magic2

import (
	"io"
	"testing"

	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/test"
)

const rimworld = `/media/downloads/TORRENTS/RimWorld [FitGirl Repack]`

func testdataRoot(t *testing.T) *lewpath.Root {
	t.Helper()
	r, err := lewpath.Open("testdata")
	if err != nil {
		t.Fatal(err)
	}
	test.CloseOnCleanup(t, r)
	return r
}

func openCorpusFile(t *testing.T, name string) io.ReadSeeker {
	t.Helper()
	r, err := lewpath.Open(rimworld)
	if err != nil {
		t.Skip("corpus not mounted")
	}
	test.CloseOnCleanup(t, r)
	f, err := lewpath.New(name).Open(r)
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
