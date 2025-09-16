package blob

import (
	"context"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
)

// FileBlob implements Reader for local filesystem paths with file:// scheme or plain paths.
type FileBlob struct{}

func (f FileBlob) Open(ctx context.Context, uri string) (io.ReadCloser, error) { // ctx unused for file
	if uri == "" {
		return nil, errors.New("empty uri")
	}
	// Support file:// and plain paths
	if u, err := url.Parse(uri); err == nil && u.Scheme == "file" {
		return os.Open(filepath.Clean(u.Path))
	}
	return os.Open(filepath.Clean(uri))
}


