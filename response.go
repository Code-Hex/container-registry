package main

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse :
// error codes: https://github.com/opencontainers/distribution-spec/blob/master/spec.md#error-codes
type ErrorResponse struct {
	Errors []Error `json:"errors"`
}

// Error :
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail"`
}

func writeErrorResponse(w http.ResponseWriter, code int, er *ErrorResponse) error {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
	w.WriteHeader(http.StatusNotFound)
	return json.NewEncoder(w).Encode(er)
}
