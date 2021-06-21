package streamtransport

import (
	"encoding/json"
	"io"

	"github.com/jmalloc/harpy"
)

// ResponseWriter is an implementation of harpy.ResponseWriter that writes
// responses to an io.Writer.
type ResponseWriter struct {
	// Target is the writer used to send JSON-RPC responses.
	Target io.Writer

	// arrayOpen indicates whether the JSON opening array bracket has been
	// written as part of a batch response.
	arrayOpen bool
}

var (
	openArray  = []byte(`[`)
	closeArray = []byte(`]`)
	comma      = []byte(`,`)
)

// WriteError writes an error response that is a result of some problem with the
// request set as a whole.
func (w *ResponseWriter) WriteError(res harpy.ErrorResponse) error {
	return w.write(res)
}

// WriteUnbatched writes a response to an individual request that was not part
// of a batch.
func (w *ResponseWriter) WriteUnbatched(res harpy.Response) error {
	return w.write(res)
}

// WriteBatched writes a response to an individual request that was part of a
// batch.
func (w *ResponseWriter) WriteBatched(res harpy.Response) error {
	separator := comma

	if !w.arrayOpen {
		w.arrayOpen = true
		separator = openArray
	}

	if _, err := w.Target.Write(separator); err != nil {
		return err
	}

	return w.write(res)
}

// Close is called to signal that there are no more responses to be sent.
//
// If batched responses have been written, it writes the closing bracket of the
// array that encapsulates the responses.
func (w *ResponseWriter) Close() error {
	if w.arrayOpen {
		_, err := w.Target.Write(closeArray)
		return err
	}

	return nil
}

// write writes a response to w.Writer.
func (w *ResponseWriter) write(res harpy.Response) error {
	enc := json.NewEncoder(w.Target)
	return enc.Encode(res)
}
