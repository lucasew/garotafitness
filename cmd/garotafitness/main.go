package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/lewtec/lewkit/x/cmd"
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

func (*root) Run(context.Context) error {
	return fmt.Errorf("usage: garotafitness extract SOURCE DEST")
}

type extractCmd struct {
	Source cmd.StringArg `help:"directory with setup.exe and fg-*.bin volumes"`
	Dest   cmd.StringArg `help:"destination directory (created if missing)"`
}

func (extractCmd) Description() string {
	return "extract SOURCE into DEST using setup.exe metadata and volume pipelines"
}

func (c *extractCmd) Run(ctx context.Context) error {
	srcRoot := c.Source.Value()
	dstRoot := c.Dest.Value()
	if srcRoot == "" || dstRoot == "" {
		return fmt.Errorf("usage: garotafitness extract SOURCE DEST")
	}
	srcInfo, err := os.Stat(srcRoot)
	if err != nil {
		return fmt.Errorf("source: %w", err)
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("source is not a directory")
	}
	dst, err := garotafitness.OpenDirDest(dstRoot)
	if err != nil {
		return err
	}
	slog.Info("extract", "source", srcRoot, "dest", dst.Root)
	ex := garotafitness.Extractor{
		Source: os.DirFS(srcRoot),
		Dest:   dst,
	}
	return ex.Extract(ctx)
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
