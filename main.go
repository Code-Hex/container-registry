package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/julienschmidt/httprouter"
)

const (
	// GET represents GET method
	GET = http.MethodGet
	// POST represents POST method
	POST = http.MethodPost
	// PATCH represents PATCH method
	PATCH = http.MethodPatch
	// PUT represents PUT method
	PUT = http.MethodPut
	// DELETE represents DELETE method
	DELETE = http.MethodDelete
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

// spec
// https://github.com/opencontainers/distribution-spec/blob/master/spec.md
func main() {

	router := httprouter.New()
	// https://github.com/opencontainers/distribution-spec/blob/master/spec.md#endpoints
	router.Handler(GET, "/v2/", DeterminingSupport())

	// /v2/:name/blobs/:digest
	router.Handler(GET, "/v2/:name/blobs/:digest", PullingBlobs())
	//router.Handler(GET, "/v2/:name/:name2/blobs/:digest", PullingBlobs())

	// /v2/:name/manifests/:reference
	router.Handler(GET, "/v2/:name/manifests/:reference", PullingManifests())
	//router.Handler(GET, "/v2/:name/:name2/manifests/:reference", PullingManifests())

	// /?digest=<digest>
	router.Handler(POST, "/v2/:name/blobs/uploads/", nil)

	router.Handler(PATCH, "/v2/:name/blobs/uploads/:reference", nil)

	// /?digest=<digest>
	router.Handler(PUT, "/v2/:name/blobs/uploads/:reference", nil)

	router.Handler(PUT, "/v2/:name/manifests/:reference", nil)

	// /?n=<integer>&last=<integer>
	router.Handler(GET, "/v2/:name/tags/list", nil)

	router.Handler(DELETE, "/v2/:name/manifests/:reference", nil)

	router.Handler(DELETE, "/v2/:name/blobs/:digest", nil)

	_ = router

	srv := &http.Server{
		Handler: ServerApply(router, AccessLogServerAdapter()),
	}
	errCh := make(chan struct{})
	go func() {
		addr := "localhost:5080"
		log.Printf("running %q", addr)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			log.Printf("error: %v", err)
			close(errCh)
			return
		}
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sig:
	case <-errCh:
	}

	if err := srv.Shutdown(context.Background()); err != nil {
		log.Printf("shutdown error: %v\n", err)
	}
}

// DeterminingSupport to check whether or not the registry implements this specification.
// If the response is 200 OK, then the registry implements this specification.
// This endpoint MAY be used for authentication/authorization purposes, but this is out of the purview of this specification.
func DeterminingSupport() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Important/Required HTTP-Headers
		// https://docs.docker.com/registry/deploying/#importantrequired-http-headers
		w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte{'{', '}'})
	})
}

const jsonDigest = "bf756fb1ae65adf866bd8c456593cd24beb6a0a061dedf42b26a993176745f6b"

// PullingBlobs to pull a blob.
func PullingBlobs() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params := httprouter.ParamsFromContext(r.Context())
		digest := params.ByName("digest")
		if index := strings.Index(digest, ":"); index != -1 {
			digest = digest[index+1:]
		}

		if digest == jsonDigest {
			w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("content-type", "application/octet-stream")
			w.Write([]byte(helloworldBlob))
			return
		}
		path := filepath.Join("testdata", digest, "layer.tar.gz")
		f, err := os.Open(path)
		if errors.Is(err, os.ErrNotExist) {
			writeErrorResponse(w,
				http.StatusNotFound,
				&ErrorResponse{
					Errors: []Error{
						{
							"BLOB_UNKNOWN",
							"blob unknown to registry",
							"sha256:" + digest,
						},
					},
				},
			)
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer f.Close()

		w.Header().Set("accept-ranges", "bytes")
		w.Header().Set("content-type", "application/octet-stream")
		io.Copy(w, f)
	})
}

// PullingManifests to pull a manifest.
func PullingManifests() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params := httprouter.ParamsFromContext(r.Context())
		name := params.ByName("name")
		name2 := params.ByName("name2")
		if name2 != "" {
			name = name + "/" + name2
		}
		ref := params.ByName("reference")

		if name != "hello-world" || ref != "latest" {
			writeErrorResponse(w,
				http.StatusNotFound,
				&ErrorResponse{
					Errors: []Error{
						{
							"MANIFEST_UNKNOWN",
							"manifest unknown",
							fmt.Sprintf(`{"name":"%s","tag":"%s"}`, name, ref),
						},
					},
				},
			)
			return
		}
		w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
		w.Write([]byte(helloworldManifest))
	})
}
