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
	Source garotafitness.SourceArg `help:"directory with setup.exe and fg-*.bin volumes"`
	Dest   garotafitness.DestArg   `help:"destination directory (created if missing)"`
}

func (extractCmd) Description() string {
	return "extract SOURCE into DEST using setup.exe metadata and volume pipelines"
}

func (c *extractCmd) Run(ctx context.Context) error {
	src := c.Source.Value()
	dst := c.Dest.Value()
	if src == nil || dst.Name() == "" {
		return fmt.Errorf("usage: garotafitness extract SOURCE DEST")
	}
	defer c.Source.Close()
	defer c.Dest.Close()
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
