package main

import (
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestParseExtract(t *testing.T) {
	t.Parallel()
	_, err := cmd.Parse[cmd.App[root]]()
	require.NoError(t, err)

	app, err := cmd.Parse[cmd.App[root]]("extract")
	require.NoError(t, err)
	require.NotNil(t, app.Args.Extract)
	require.Error(t, app.Args.Extract.Run(t.Context()))

	src := t.TempDir()
	dst := t.TempDir()
	app, err = cmd.Parse[cmd.App[root]]("extract", src, dst)
	require.NoError(t, err)
	require.NotNil(t, app.Args.Extract)
	test.CloseOnCleanup(t, app.Args.Extract.Source)
	test.CloseOnCleanup(t, app.Args.Extract.Dest)
	require.NotNil(t, app.Args.Extract.Source.Value())
	require.Equal(t, src, app.Args.Extract.Source.Value().Name())
	require.Equal(t, dst, app.Args.Extract.Dest.Name())
}
