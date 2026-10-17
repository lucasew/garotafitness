// Package fsb remuxes chained Ogg Vorbis streams into an FSB5 sound bank.
package fsb

import (
	"context"
	"fmt"
	"io"
)

// Remux preserves the Vorbis packets and constructs the FSB5 headers and padding.
// Arguments match the WASI guest ("fsb", "-", "-"), so no loop-point chunk is written.
func Remux(ctx context.Context, dst io.Writer, src io.Reader) error {
	if src == nil || dst == nil {
		return fmt.Errorf("fsb: nil input or output")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := io.ReadAll(ctxReader{ctx: ctx, r: src})
	if err != nil {
		return fmt.Errorf("fsb: remux: %w", err)
	}
	out, err := remux(data)
	if err != nil {
		return fmt.Errorf("fsb: remux: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := dst.Write(out); err != nil {
		return fmt.Errorf("fsb: remux: %w", err)
	}
	return nil
}

// NewReader returns FSB5 bytes. Close cancels the remux.
func NewReader(ctx context.Context, src io.Reader) (io.ReadCloser, error) {
	if src == nil {
		return nil, fmt.Errorf("fsb: nil reader")
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

type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (c ctxReader) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	return c.r.Read(p)
}
