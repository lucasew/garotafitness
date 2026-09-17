package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/lewtec/lewkit/x/cmd"
	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/taskgroup/progress"
	"github.com/lucasew/garotafitness"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

type root struct {
	taskgroup.Arg `flatten:"" ctx:"taskgroup"`
	Extract       *extractCmd `cmd:"extract"`
}

func (root) Description() string {
	return "Extract a local FitGirl repack into a destination tree."
}

type extractCmd struct {
	Source cmd.WorkDirArg `help:"repack directory with setup.exe and fg-*.bin volumes"`
	Dest   cmd.DataDirArg `help:"destination directory"`
}

func (extractCmd) Description() string {
	return "extract SOURCE into DEST using setup.exe metadata and volume pipelines"
}

func (c *extractCmd) Run(ctx context.Context) error {
	srcPath := c.Source.Value()
	dstPath := c.Dest.Value()
	if srcPath == "" || dstPath == "" {
		return cmd.ErrUsage
	}
	sess, ctx := enterSession(ctx)
	return progress.Run(sess, ctx, func(ctx context.Context) error {
		src, err := lewpath.Open(srcPath)
		if err != nil {
			return err
		}
		defer src.Close()
		dst, err := garotafitness.OpenDirDest(dstPath)
		if err != nil {
			return err
		}
		defer dst.Close()
		slog.Info("extract", "source", src.Name(), "dest", dst.Name())
		return garotafitness.Extractor{Source: src, Dest: dst}.Extract(ctx)
	})
}

func enterSession(ctx context.Context) (*taskgroup.Session, context.Context) {
	if s := taskgroup.FromContext(ctx); s != nil {
		return s, ctx
	}
	if arg, ok := cmd.Lookup[taskgroup.Arg](ctx, "taskgroup"); ok {
		return arg.Enter(ctx, taskgroup.DefaultLimits())
	}
	return taskgroup.New(ctx, taskgroup.DefaultLimits())
}

func run(args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	context.AfterFunc(ctx, func() {
		if err := ctx.Err(); err != nil {
			slog.Warn("interrupt or parent cancel", "err", err, "cause", context.Cause(ctx))
		}
	})
	app, err := cmd.Parse[cmd.App[root]](args...)
	if err != nil {
		return err
	}
	err = app.Run(ctx)
	if err != nil {
		slog.Error("extract failed", "err", err, "cause", context.Cause(ctx))
	}
	return err
}
