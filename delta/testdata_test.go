package delta

import (
	"testing"

	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/test"
)

func testdataRoot(t *testing.T) *lewpath.Root {
	t.Helper()
	r, err := lewpath.Open("testdata")
	if err != nil {
		t.Fatal(err)
	}
	test.CloseOnCleanup(t, r)
	return r
}
