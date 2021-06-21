package streamtransport

import (
	"context"
	"io"

	"github.com/jmalloc/harpy"
)

// RequestSetReader is an implementation of harpy.RequestSetReader that reads
// JSON-RPC request sets from an io.Reader.
type RequestSetReader struct {
	Source io.Reader
}

// Read reads the next RequestSet that is to be processed.
//
// It returns ctx.Err() if ctx is canceled while waiting to read the next
// request set. If request set data is read but cannot be parsed a native
// JSON-RPC Error is returned. Any other error indicates an IO error.
func (r *RequestSetReader) Read(ctx context.Context) (harpy.RequestSet, error) {
	return harpy.ParseRequestSet(r.Source)
}
