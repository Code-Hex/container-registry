package main

import (
	"log"
	"net/http"

	"github.com/Code-Hex/container-registry/internal/errors"
)

// Response is a wrapper of http.ResponseWriter
type Response struct {
	http.ResponseWriter
	statusCode int
}

func newResponse(w http.ResponseWriter) *Response {
	return &Response{
		ResponseWriter: w,
	}
}

var _ http.ResponseWriter = (*Response)(nil)

// WriteHeader wraps http.ResponseWriter.WriteHeader and
// store status code which is written.
func (r *Response) WriteHeader(statusCode int) {
	if r.statusCode != 0 {
		return
	}
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(r.statusCode)
}

// Handler handles http handler and error which is caused in it.
type Handler func(w http.ResponseWriter, r *http.Request) error

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h(w, r)
	if err == nil {
		return
	}
	if err := errors.ServeJSON(w, err); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}
