package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/lucasew/garotafitness"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if err := run(os.Args[1:]); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func parseArgs(args []string) (source, dest string, err error) {
	if len(args) != 3 || args[0] != "extract" {
		return "", "", fmt.Errorf("usage: garotafitness extract SOURCE DEST")
	}
	return args[1], args[2], nil
}

func run(args []string) error {
	srcRoot, dstRoot, err := parseArgs(args)
	if err != nil {
		return err
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
	ex := garotafitness.Extractor{
		Source: os.DirFS(srcRoot),
		Dest:   dst,
	}
	return ex.Extract(context.Background())
}
