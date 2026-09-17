package garotafitness

import (
	"context"
	"errors"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEachEmpty(t *testing.T) {
	t.Parallel()
	require.NoError(t, each(t.Context(), "test", slices.Values([]int(nil)), func(context.Context, int) error {
		t.Fatal("ran")
		return nil
	}))
}

func TestEachOne(t *testing.T) {
	t.Parallel()
	var n atomic.Int32
	err := each(t.Context(), "test", slices.Values([]int{7}), func(_ context.Context, v int) error {
		n.Add(int32(v))
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, int32(7), n.Load())
}

func TestEachMany(t *testing.T) {
	t.Parallel()
	var n atomic.Int32
	seq := func(yield func(int) bool) {
		for i := 0; i < 16; i++ {
			if !yield(i) {
				return
			}
		}
	}
	require.NoError(t, each(t.Context(), "test", seq, func(context.Context, int) error {
		n.Add(1)
		return nil
	}))
	require.Equal(t, int32(16), n.Load())
}

func TestEachFirstError(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	err := each(t.Context(), "test", slices.Values([]int{1, 2, 3, 4}), func(_ context.Context, v int) error {
		if v == 1 {
			return boom
		}
		return nil
	})
	require.ErrorIs(t, err, boom)
}

func TestEachCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := each(ctx, "test", slices.Values([]int{1}), func(context.Context, int) error {
		return errors.New("should not run")
	})
	require.ErrorIs(t, err, context.Canceled)
}
