package main

import (
	"log"
	"net/http"
)

// AccessLogServerAdapter logs access log
func AccessLogServerAdapter() ServerAdapter {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Println(r.URL.String())
			next.ServeHTTP(w, r)
		})
	}
}
