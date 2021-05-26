package harpy

import (
	"context"
	"encoding/json"
	"mime"
	"net/http"
)

// HTTPHandler is an implementation of http.Handler that provides an HTTP-based
// transport for a JSON-RPC server.
type HTTPHandler struct {
	// Exchanger is the Exchange that handles JSON-RPC requests.
	Exchanger Exchanger
}

// ServeHTTP handles the HTTP request.
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rw := &httpResponseWriter{
		w:   w,
		enc: json.NewEncoder(w),
	}

	if r.Method != http.MethodPost {
		rw.writeError(
			http.StatusMethodNotAllowed,
			NewErrorResponse(
				nil,
				NewErrorWithReservedCode(
					InvalidRequestCode,
					WithMessage("JSON-RPC requests must use the POST method"),
				),
			),
		)
		return
	}

	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != httpMediaType {
		rw.writeError(
			http.StatusUnsupportedMediaType,
			NewErrorResponse(
				nil,
				NewErrorWithReservedCode(
					InvalidRequestCode,
					WithMessage("JSON-RPC requests must use the application/json content type"),
				),
			),
		)
		return
	}

	ctx := r.Context()

	rs, err := ParseRequestSet(r.Body)

	switch err.(type) {
	case nil:
		err = Exchange(
			ctx,
			rs,
			h.Exchanger,
			rw,
		)
	case Error:
		err = rw.WriteError(
			ctx,
			RequestSet{},
			NewErrorResponse(nil, err),
		)
	default:
		rw.writeError(
			http.StatusInternalServerError,
			NewErrorResponse(
				nil,
				NewErrorWithReservedCode(
					InternalErrorCode,
					WithMessage("unable to read request body"),
				),
			),
		)
		return
	}
}

// httpMediaType is the MIME media-type for JSON-RPC requests and responses when
// delivered over HTTP.
const httpMediaType = "application/json"

var (
	openArray  = []byte(`[`)
	closeArray = []byte(`]`)
	comma      = []byte(`,`)
)

// httpResponseWriter is an implementation of ResponseWriter that sends
// responses to HTTP requests.
type httpResponseWriter struct {
	w       http.ResponseWriter
	enc     *json.Encoder
	isBatch bool
}

// WriteError writes an error response that is a result of some problem with the
// request set as a whole.
//
// It immediately writes the HTTP response headers followed by the HTTP body.
//
// If the error uses one of the error codes reserved by the JSON-RPC
// specification the HTTP status code is set to the most appropriate equivalent.
// Application-defined JSON-RPC errors always result in a HTTP 200 OK, as they
// considered part of normal operation of the transport.
func (w *httpResponseWriter) WriteError(_ context.Context, _ RequestSet, res ErrorResponse) error {
	return w.writeError(0, res)
}

// WriteUnbatched writes a response to an individual request that was not part
// of a batch.
//
// It immediately writes the HTTP response headers followed by the HTTP body.
//
// If res is an ErrorResponse and its error code is one of the error codes
// reserved by the JSON-RPC specification the HTTP status code is set to the
// most appropriate equivalent. Application-defined JSON-RPC errors always
// result in a HTTP 200 OK, as they considered part of normal operation of the
// transport.
func (w *httpResponseWriter) WriteUnbatched(_ context.Context, _ Request, res Response) error {
	if e, ok := res.(ErrorResponse); ok {
		return w.writeError(0, e)
	}

	w.w.Header().Set("Content-Type", httpMediaType)
	return w.enc.Encode(res)
}

// WriteBatched writes a response to an individual request that was part of a
// batch.
//
// If this is the first response of the batch, it immediately writes the HTTP
// response headers and the opening bracket of the array that encapsulates the
// batch of responses.
//
// The HTTP status is always HTTP 200 OK, as even if res is an ErrorResponse,
// other responses in the batch may indicate a success.
func (w *httpResponseWriter) WriteBatched(_ context.Context, _ Request, res Response) error {
	separator := comma

	if !w.isBatch {
		w.w.Header().Set("Content-Type", httpMediaType)
		w.isBatch = true
		separator = openArray
	}

	if _, err := w.w.Write(separator); err != nil {
		return err
	}

	return w.enc.Encode(res)
}

// Close is called to signal that there are no more responses to be sent.
//
// If batched responses have been written, it writes the closing bracket of the
// array that encapsulates the responses.
func (w *httpResponseWriter) Close() error {
	if w.isBatch {
		_, err := w.w.Write(closeArray)
		return err
	}

	return nil
}

// writeError writes a JSON-RPC error response to the HTTP response.
func (w *httpResponseWriter) writeError(code int, res ErrorResponse) error {
	if code == 0 {
		code = httpStatusFromErrorCode(res.Error.Code)
	}

	w.w.Header().Set("Content-Type", httpMediaType)
	w.w.WriteHeader(code)
	return w.enc.Encode(res)
}

// httpStatusFromErrorCode returns the appropriate HTTP status code to send in
// response to a specific JSON-RPC error code.
func httpStatusFromErrorCode(c ErrorCode) int {
	if !c.IsReserved() {
		// If the error code is not "reserved" that means its an
		// application-defined error. We do write the response using an OK
		// status as even though an error occurred there was no problem with the
		// request or the HTTP encapsulation itself.
		return http.StatusOK
	}

	switch c {
	case ParseErrorCode:
		return http.StatusBadRequest
	case InvalidRequestCode:
		return http.StatusBadRequest
	case InvalidParametersCode:
		return http.StatusBadRequest
	case MethodNotFoundCode:
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}
