package garotafitness

import (
	"context"

	"github.com/lewtec/lewkit/x/taskgroup"
)

// withSession uses the Session in ctx, or starts one with DefaultLimits.
func withSession(ctx context.Context, fn func(context.Context) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
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
