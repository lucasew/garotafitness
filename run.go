package garotafitness

import (
	"context"
	"iter"

	"github.com/lewtec/lewkit/x/taskgroup"
)

// withSession uses the Session in ctx, or starts one with DefaultLimits.
func withSession(ctx context.Context, fn func(context.Context) error) error {
	if taskgroup.FromContext(ctx) != nil {
		return fn(ctx)
	}
	sess, ctx := taskgroup.New(ctx, taskgroup.DefaultLimits())
	err := fn(ctx)
	if werr := sess.Wait(); err == nil {
		err = werr
	}
	return err
}

// each runs independent extract units on pool. Algorithm packages do not call this.
func each[T any](ctx context.Context, in iter.Seq[T], fn func(context.Context, T) error) error {
	return eachPool(ctx, taskgroup.CPU, in, fn)
}

func eachPool[T any](ctx context.Context, pool taskgroup.PoolKind, in iter.Seq[T], fn func(context.Context, T) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return withSession(ctx, func(ctx context.Context) error {
		var items []T
		for v := range in {
			items = append(items, v)
		}
		return taskgroup.Each[T]{
			PoolKind: pool,
			Items:    items,
			Fn: func(ctx context.Context, _ *taskgroup.Status, item T) error {
				return fn(ctx, item)
			},
		}.Run(ctx)
	})
}
