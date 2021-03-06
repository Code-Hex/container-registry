package main

import (
	"log"
	"net/http"
)

// ServerAdapter represents a apply middleware type for http server.
type ServerAdapter func(http.Handler) http.Handler

// ServerApply applies http server middlewares
func ServerApply(h http.Handler, adapters ...ServerAdapter) http.Handler {
	// To process from left to right, iterate from the last one.
	for i := len(adapters) - 1; i >= 0; i-- {
		h = adapters[i](h)
	}
	return h
}

// AccessLogServerAdapter logs access log
func AccessLogServerAdapter() ServerAdapter {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wrapped := newResponse(w)
			next.ServeHTTP(wrapped, r)
			log.Println(r.Method, wrapped.statusCode, r.URL.String())
		})
	}
}

func SetHeaderServerAdapter() ServerAdapter {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Important/Required HTTP-Headers
			// https://docs.docker.com/registry/deploying/#importantrequired-http-headers
			w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			next.ServeHTTP(w, r)
		})
	}
}
