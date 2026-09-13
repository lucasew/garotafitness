package garotafitness

import (
	"testing"

	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/test"
)

const rimworldCorpus = `/media/downloads/TORRENTS/RimWorld [FitGirl Repack]`

func openCorpus(t *testing.T) *lewpath.Root {
	t.Helper()
	r, err := lewpath.Open(rimworldCorpus)
	if err != nil {
		t.Skip("corpus not mounted")
	}
	test.CloseOnCleanup(t, r)
	return r
}
