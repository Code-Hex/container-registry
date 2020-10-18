package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Code-Hex/container-registry/internal/errors"
)

func TestResponse_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	r := newResponse(w)
	if r.statusCode != 0 {
		t.Fatalf("want %q, but got %q", 0, r.statusCode)
	}
	r.WriteHeader(http.StatusOK)
	if r.statusCode != http.StatusOK {
		t.Fatalf("want %q, but got %q", http.StatusOK, r.statusCode)
	}
	// ignore
	r.WriteHeader(http.StatusInternalServerError)
	if r.statusCode != http.StatusOK {
		t.Fatalf("want %q, but got %q", http.StatusOK, r.statusCode)
	}
}

func TestHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name       string
		h          Handler
		want       string
		wantStatus int
	}{
		{
			name: "success",
			h: func(w http.ResponseWriter, _ *http.Request) error {
				return nil
			},
			want:       "",
			wantStatus: http.StatusOK,
		},
		{
			name: "failed if std error",
			h: func(w http.ResponseWriter, _ *http.Request) error {
				return fmt.Errorf("error")
			},
			want:       `{"code":"UNKNOWN","message":"unknown error"}`,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "failed if wrapped error",
			h: func(w http.ResponseWriter, _ *http.Request) error {
				err := fmt.Errorf("error")
				return errors.Wrap(err,
					errors.WithStatusCode(http.StatusPreconditionFailed),
				)
			},
			want:       `{"code":"UNKNOWN","message":"unknown error"}`,
			wantStatus: http.StatusPreconditionFailed,
		},
		{
			name: "failed if blob unknown",
			h: func(w http.ResponseWriter, _ *http.Request) error {
				err := fmt.Errorf("error")
				return errors.Wrap(err,
					errors.WithCodeBlobUnknown(),
				)
			},
			want:       `{"code":"BLOB_UNKNOWN","message":"blob unknown to registry"}`,
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tt.h.ServeHTTP(w, nil)
			gotBody := strings.TrimSpace(w.Body.String())
			if gotBody != tt.want {
				t.Fatalf("want body %q, but got %q", tt.want, w.Body.String())
			}
			if w.Code != tt.wantStatus {
				t.Fatalf("want statuscode %q, but got %q", tt.wantStatus, w.Code)
			}
		})
	}
}
