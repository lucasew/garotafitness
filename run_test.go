package garotafitness

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunParallelEmpty(t *testing.T) {
	t.Parallel()
	require.NoError(t, runParallel(t.Context(), nil))
}

func TestRunParallelOne(t *testing.T) {
	t.Parallel()
	var n atomic.Int32
	err := runParallel(t.Context(), []func(context.Context) error{
		func(context.Context) error {
			n.Add(1)
			return nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, int32(1), n.Load())
}

func TestRunParallelMany(t *testing.T) {
	t.Parallel()
	const want = 16
	var n atomic.Int32
	fns := make([]func(context.Context) error, want)
	for i := range fns {
		fns[i] = func(context.Context) error {
			n.Add(1)
			return nil
		}
	}
	require.NoError(t, runParallel(t.Context(), fns))
	require.Equal(t, int32(want), n.Load())
}

func TestRunParallelFirstError(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	err := runParallel(t.Context(), []func(context.Context) error{
		func(context.Context) error { return boom },
		func(ctx context.Context) error { return ctx.Err() },
	})
	require.ErrorIs(t, err, boom)
}

func TestRunParallelCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := runParallel(ctx, []func(context.Context) error{
		func(context.Context) error { return errors.New("should not run") },
	})
	require.ErrorIs(t, err, context.Canceled)
}
