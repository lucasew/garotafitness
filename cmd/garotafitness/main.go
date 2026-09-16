package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/lewtec/lewkit/x/cmd"
	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lucasew/garotafitness"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

type root struct {
	Extract *extractCmd `cmd:"extract"`
}

func (root) Description() string {
	return "Extract a local FitGirl repack into a destination tree."
}

type extractCmd struct {
	taskgroup.Arg
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
	sess, ctx := c.Enter(ctx, taskgroup.DefaultLimits())
	defer sess.Wait()
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
}

func run(args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	app, err := cmd.Parse[cmd.App[root]](args...)
	if err != nil {
		return err
	}
	return app.Run(ctx)
}
