package garotafitness

import (
	"context"
	"runtime"

	"golang.org/x/sync/errgroup"
)

// runParallel runs independent extract units. The first error cancels the rest.
// Algorithm packages keep a synchronous NewReader and do not call this.
func runParallel(ctx context.Context, fns []func(context.Context) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	switch len(fns) {
	case 0:
		return nil
	case 1:
		return fns[0](ctx)
	}
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(workers())
	for _, fn := range fns {
		g.Go(func() error {
			if err := ctx.Err(); err != nil {
				return err
			}
			return fn(ctx)
		})
	}
	return g.Wait()
}

func workers() int {
	return max(1, runtime.GOMAXPROCS(0))
}
