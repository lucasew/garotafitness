package garotafitness

import (
	"context"
	"iter"
	"runtime"

	"golang.org/x/sync/errgroup"
)

// each hands values from in to workers over an unbuffered channel.
// The first error cancels the rest. Algorithm packages do not call this.
func each[T any](ctx context.Context, in iter.Seq[T], fn func(context.Context, T) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	g, ctx := errgroup.WithContext(ctx)
	ch := make(chan T)
	g.Go(func() error {
		defer close(ch)
		for v := range in {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case ch <- v:
			}
		}
		return nil
	})
	for range workers() {
		g.Go(func() error {
			for v := range ch {
				if err := ctx.Err(); err != nil {
					return err
				}
				if err := fn(ctx, v); err != nil {
					return err
				}
			}
			return nil
		})
	}
	return g.Wait()
}

func workers() int {
	return max(1, runtime.GOMAXPROCS(0))
}
