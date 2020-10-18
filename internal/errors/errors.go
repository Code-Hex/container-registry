package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Error for using wrapped error.
type Error struct {
	Err        error `json:"-"`
	StatusCode int   `json:"-"`

	// https://github.com/opencontainers/distribution-spec/blob/master/spec.md#error-codes
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Detail  interface{} `json:"detail,omitempty"`
}

// WrapOption represents option for Wrap function.
type WrapOption func(e *Error)

// WithStatusCode wraps error with http status code.
func WithStatusCode(sc int) WrapOption {
	return func(e *Error) {
		e.StatusCode = sc
	}
}

// WithDetail wraps error with detail.
func WithDetail(detail interface{}) WrapOption {
	return func(e *Error) {
		e.Detail = detail
	}
}

// Wrap wraps error which is also sets other fields.
func Wrap(err error, opts ...WrapOption) *Error {
	wrapped := &Error{Err: err}
	WithCodeUnknown()(wrapped) // initialize
	for _, wo := range opts {
		wo(wrapped)
	}
	return wrapped
}

func (e *Error) Error() string {
	if e.Err == nil {
		return "<nil>"
	}
	if e.Detail == nil {
		return e.Err.Error()
	}
	// best effort
	v, _ := json.Marshal(e.Detail)
	if len(v) != 0 {
		return fmt.Sprintf("err: %q, detail: %v", e.Err, string(v))
	}
	return e.Err.Error()
}

// ServeJSON attempts to serve the errcode in a JSON envelope. It marshals err
// and sets the content-type header to 'application/json'. It will handle
// Error and some errors which is converted to Error, and if necessary will create an envelope.
func ServeJSON(w http.ResponseWriter, err error) error {
	if err == nil {
		return nil
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	e := func(err error) *Error {
		switch e := err.(type) {
		case *Error:
			return e
		}
		return Wrap(err)
	}(err)

	w.WriteHeader(e.StatusCode)

	return json.NewEncoder(w).Encode(e)
}
