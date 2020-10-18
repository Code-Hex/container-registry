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
	committed  bool
}

func newResponse(w http.ResponseWriter) *Response {
	return &Response{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

var _ http.ResponseWriter = (*Response)(nil)

// WriteHeader wraps http.ResponseWriter.WriteHeader and
// store status code which is written.
func (r *Response) WriteHeader(statusCode int) {
	if r.committed {
		return
	}
	r.committed = true
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
	log.Println("error:", err)
	if err := errors.ServeJSON(w, err); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}
