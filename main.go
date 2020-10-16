package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/Code-Hex/container-registry/internal/grammar"
	"github.com/Code-Hex/go-router-simple"
	"github.com/google/uuid"
	digest "github.com/opencontainers/go-digest"
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

const hostname = "localhost:5080"

// spec
// https://github.com/opencontainers/distribution-spec/blob/master/spec.md
func main() {
	rs := router.New()

	// https://github.com/opencontainers/distribution-spec/blob/master/spec.md#endpoints
	rs.GET("/v2/", DeterminingSupport())

	// /v2/:name/blobs/:digest
	rs.GET(
		fmt.Sprintf(
			`/v2/{name:%s}/blobs/{digest:%s}`,
			grammar.Name, grammar.DigestHex,
		),
		PullingBlobs(),
	)

	// /v2/:name/manifests/:reference
	rs.GET(
		fmt.Sprintf(
			`/v2/{name:%s}/manifests/{reference:%s}`,
			grammar.Name, grammar.Reference,
		),
		PullingManifests(),
	)

	// /?digest=<digest>
	rs.POST(
		fmt.Sprintf(
			`/v2/{name:%s}/blobs/uploads/`,
			grammar.Name,
		),
		PushBlob(),
	)

	rs.PATCH(
		fmt.Sprintf(
			`/v2/{name:%s}/blobs/uploads/{reference:%s}`,
			grammar.Name, grammar.Reference,
		),
		PushBlobPatch(),
	)

	// /?digest=<digest>
	rs.PUT(
		fmt.Sprintf(
			`/v2/{name:%s}/blobs/uploads/{reference:%s}`,
			grammar.Name, grammar.Reference,
		),
		PushBlobPut(),
	)

	rs.HEAD(
		fmt.Sprintf(
			`/v2/{name:%s}/blobs/{digest:%s}`,
			grammar.Name, grammar.Digest,
		),
		PushBlobHead(),
	)

	rs.PUT(
		fmt.Sprintf(
			`/v2/{name:%s}/manifests/{tag:%s}`,
			grammar.Name, grammar.Tag,
		),
		PushManifestPut(),
	)

	// /?n=<integer>&last=<integer>
	rs.Handle(GET, "/v2/:name/tags/list", nil)

	rs.Handle(DELETE, "/v2/:name/manifests/:reference", nil)

	rs.Handle(DELETE, "/v2/:name/blobs/:digest", nil)

	srv := &http.Server{
		Handler: ServerApply(rs, AccessLogServerAdapter()),
	}
	errCh := make(chan struct{})
	go func() {
		addr := hostname
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
		ctx := r.Context()
		dq := router.ParamFromContext(ctx, "digest")
		dgst, err := digest.Parse(dq)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if dgst == jsonDigest {
			w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("content-type", "application/octet-stream")
			w.Write([]byte(helloworldBlob))
			return
		}
		dgstHex := dgst.Hex()
		path := filepath.Join("testdata", dgstHex, "layer.tar.gz")
		f, err := os.Open(path)
		if errors.Is(err, os.ErrNotExist) {
			writeErrorResponse(w,
				http.StatusNotFound,
				&ErrorResponse{
					Errors: []Error{
						{
							"BLOB_UNKNOWN",
							"blob unknown to registry",
							"sha256:" + dgstHex,
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
		ctx := r.Context()
		name := router.ParamFromContext(ctx, "name")
		ref := router.ParamFromContext(ctx, "reference")

		if name != "library/hello-world" || ref != "latest" {
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

func PushBlob() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uuid := uuid.New().String()
		name := router.ParamFromContext(r.Context(), "name")
		os.MkdirAll(joinWithBasePath(name), 0700)
		location := "/v2/" + name + "/blobs/uploads/" + uuid
		//log.Println(location)
		w.Header().Set("Location", location)
		w.WriteHeader(http.StatusAccepted)
	})
}

func PushBlobPatch() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		name := router.ParamFromContext(ctx, "name")
		reference := router.ParamFromContext(ctx, "reference")

		path := joinWithBasePath(name, reference)
		os.MkdirAll(path, 0700)

		f, err := os.Create(path + "/" + "layer.tar.gz")
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer f.Close()

		size, err := io.Copy(f, r.Body)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		location := "/v2/" + name + "/blobs/uploads/" + reference
		w.Header().Set("Location", location)
		w.Header().Set("Docker-Upload-UUID", reference)
		w.Header().Set("Range", fmt.Sprintf("0-%d", size))
		w.WriteHeader(http.StatusAccepted)
	})
}

func PushBlobPut() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dgst, err := digest.Parse(r.URL.Query().Get("digest"))
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		ctx := r.Context()
		name := router.ParamFromContext(ctx, "name")
		newDir := joinWithBasePath(name, dgst.String())
		os.MkdirAll(newDir, 0700)

		reference := router.ParamFromContext(ctx, "reference")
		uuid := reference
		oldDir := joinWithBasePath(name, uuid)
		oldpath := filepath.Join(oldDir, "layer.tar.gz")
		newpath := filepath.Join(newDir, "layer.tar.gz")
		if err := os.Rename(oldpath, newpath); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		os.Remove(oldDir)

		w.WriteHeader(http.StatusCreated)
	})
}

func PushBlobHead() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		dq := router.ParamFromContext(ctx, "digest")
		dgst, err := digest.Parse(dq)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		name := router.ParamFromContext(ctx, "name")
		dir := joinWithBasePath(name, dgst.String())
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		fis, err := ioutil.ReadDir(dir)
		if err != nil || len(fis) != 1 {
			log.Printf("unexpected directory: %q, err: %v, fis: %q", dir, err, fis)
			w.WriteHeader(http.StatusPreconditionFailed)
			return
		}
		filename := fis[0].Name()
		size := fis[0].Size()
		ext := filepath.Ext(filename)
		if ext == ".json" {
			w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
		} else {
			w.Header().Set("Content-Type", "application/vnd.docker.image.rootfs.diff.tar.gzip")
		}
		w.Header().Set("Docker-Content-Digest", dgst.String())
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		w.WriteHeader(http.StatusAccepted)
	})
}

func PushManifestPut() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var m Manifest
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		name := router.ParamFromContext(ctx, "name")
		path := joinWithBasePath(name, "manifest.json")
		f, err := os.Create(path)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer f.Close()
		if err := json.NewEncoder(f).Encode(&m); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Docker-Content-Digest", m.Config.Digest.String())
		w.WriteHeader(http.StatusCreated)
	})
}

func joinWithBasePath(name string, p ...string) string {
	return filepath.Join(
		append(
			[]string{
				"testdata",
				name,
			},
			p...,
		)...,
	)
}
