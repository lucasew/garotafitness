package garotafitness

import (
	"context"
	"io"
)

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

func copyCtx(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	return io.Copy(dst, ctxReader{ctx, src})
}
