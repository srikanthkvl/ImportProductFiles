package blob

import (
	"context"
	"io"
)

// Reader abstracts reading a single blob object content as a stream.
type Reader interface {
	Open(ctx context.Context, uri string) (io.ReadCloser, error)
}


