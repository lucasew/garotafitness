// Package fsb remuxes chained Ogg Vorbis streams into an FSB5 sound bank.
package fsb

import (
	"context"
	_ "embed"
	"fmt"
	"io"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed fsb.wasm
var guest []byte

// Remux preserves the Vorbis packets and constructs the FSB5 headers and padding.
// It runs the vendored oggvorbis2fsb5 implementation in-process.
func Remux(ctx context.Context, dst io.Writer, src io.Reader) error {
	if src == nil || dst == nil {
		return fmt.Errorf("fsb: nil input or output")
	}
	rt := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().WithCloseOnContextDone(true))
	defer rt.Close(context.Background())
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		return fmt.Errorf("fsb: wasi: %w", err)
	}
	if _, err := rt.NewHostModuleBuilder("env").NewFunctionBuilder().
		WithFunc(func(uint32) {}).Export("emscripten_notify_memory_growth").Instantiate(ctx); err != nil {
		return fmt.Errorf("fsb: env: %w", err)
	}
	_, err := rt.InstantiateWithConfig(ctx, guest, wazero.NewModuleConfig().
		WithArgs("fsb", "-", "-").WithStdin(src).WithStdout(dst).WithStderr(io.Discard))
	if err != nil {
		return fmt.Errorf("fsb: remux: %w", err)
	}
	return nil
}

// NewReader returns FSB5 bytes. Close cancels the remux and releases its Guest.
func NewReader(ctx context.Context, src io.Reader) (io.ReadCloser, error) {
	if src == nil {
		return nil, fmt.Errorf("fsb: nil reader")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	r, w := io.Pipe()
	done := make(chan struct{})
	go func() {
		defer close(done)
		w.CloseWithError(Remux(ctx, w, src))
	}()
	return &reader{PipeReader: r, cancel: cancel, done: done}, nil
}

type reader struct {
	*io.PipeReader
	cancel context.CancelFunc
	done   <-chan struct{}
}

func (r *reader) Close() error {
	r.cancel()
	err := r.PipeReader.Close()
	<-r.done
	return err
}
