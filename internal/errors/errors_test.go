package errors

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServeJSON(t *testing.T) {
	err := fmt.Errorf("error")
	tests := []struct {
		name       string
		err        error
		want       string
		wantStatus int
	}{
		{
			name:       "success",
			err:        nil,
			want:       "",
			wantStatus: http.StatusOK,
		},
		{
			name:       "failed if std error",
			err:        err,
			want:       `{"code":"UNKNOWN","message":"unknown error"}`,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "failed if wrapped error",
			err: Wrap(err,
				WithStatusCode(http.StatusPreconditionFailed),
			),
			want:       `{"code":"UNKNOWN","message":"unknown error"}`,
			wantStatus: http.StatusPreconditionFailed,
		},
		{
			name: "failed if blob unknown",
			err: Wrap(err,
				WithCodeBlobUnknown(),
			),
			want:       `{"code":"BLOB_UNKNOWN","message":"blob unknown to registry"}`,
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ServeJSON(w, tt.err)
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
