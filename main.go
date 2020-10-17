package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/Code-Hex/container-registry/internal/grammar"
	"github.com/Code-Hex/container-registry/internal/registry"
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
			grammar.Name, grammar.Digest,
		),
		PullingBlobs(),
	)

	// /v2/:name/manifests/:reference
	rs.GET(
		fmt.Sprintf(
			`/v2/{name:%s}/manifests/{tag:%s}`,
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
		name := router.ParamFromContext(ctx, "name")
		dir := registry.PathJoinWithBase(name, dgst.String())
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			writeErrorResponse(w,
				http.StatusNotFound,
				&ErrorResponse{
					Errors: []Error{
						{
							"BLOB_UNKNOWN",
							"blob unknown to registry",
							dgst.String(),
						},
					},
				},
			)
			return
		}

		fi, err := registry.PickupFileinfo(dir)
		if err != nil {
			log.Printf("unexpected directory: %q, err: %v", dir, err)
			w.WriteHeader(http.StatusPreconditionFailed)
			return
		}
		filename := fi.Name()
		if strings.HasSuffix(filename, ".json") {
			w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("content-type", "application/octet-stream")
		} else {
			w.Header().Set("accept-ranges", "bytes")
			w.Header().Set("content-type", "application/octet-stream")
		}

		path := filepath.Join(dir, filename)
		f, err := os.Open(path)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer f.Close()
		io.Copy(w, f)
	})
}

// PullingManifests to pull a manifest.
func PullingManifests() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		name := router.ParamFromContext(ctx, "name")
		tag := router.ParamFromContext(ctx, "tag")

		manifest := registry.PathJoinWithBase(name, tag, "manifest.json")
		if _, err := os.Stat(manifest); os.IsNotExist(err) {
			writeErrorResponse(w,
				http.StatusNotFound,
				&ErrorResponse{
					Errors: []Error{
						{
							"MANIFEST_UNKNOWN",
							"manifest unknown",
							tag,
						},
					},
				},
			)
			return
		}

		f, err := os.Open(manifest)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer f.Close()

		w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
		io.Copy(w, f)
	})
}

func PushBlob() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		uuid := uuid.New().String()
		name := router.ParamFromContext(r.Context(), "name")
		os.MkdirAll(registry.PathJoinWithBase(name), 0700)
		location := "/v2/" + name + "/blobs/uploads/" + uuid
		//log.Println(location)
		w.Header().Set("Location", location)
		w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusAccepted)
	})
}

func PushBlobPatch() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		name := router.ParamFromContext(ctx, "name")
		reference := router.ParamFromContext(ctx, "reference")

		path := registry.PathJoinWithBase(name, reference)
		os.MkdirAll(path, 0700)

		size, err := registry.CreateLayer(r.Body, path)
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
		newDir := registry.PathJoinWithBase(name, dgst.String())
		os.MkdirAll(newDir, 0700)

		reference := router.ParamFromContext(ctx, "reference")
		uuid := reference
		oldDir := registry.PathJoinWithBase(name, uuid)
		fi, err := registry.PickupFileinfo(oldDir)
		if err != nil {
			log.Printf("unexpected directory: %q, err: %v", oldDir, err)
			w.WriteHeader(http.StatusPreconditionFailed)
			return
		}
		filename := fi.Name()

		oldpath := filepath.Join(oldDir, filename)
		newpath := filepath.Join(newDir, filename)
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
		dir := registry.PathJoinWithBase(name, dgst.String())
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		fi, err := registry.PickupFileinfo(dir)
		if err != nil {
			log.Printf("unexpected directory: %q, err: %v", dir, err)
			w.WriteHeader(http.StatusPreconditionFailed)
			return
		}
		filename := fi.Name()
		size := fi.Size()
		ext := filepath.Ext(filename)
		if ext == ".json" {
			w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
		} else {
			w.Header().Set("Content-Type", "application/vnd.docker.image.rootfs.diff.tar.gzip")
		}
		w.Header().Set("Docker-Content-Digest", dgst.String())
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
		w.Header().Set("X-Content-Type-Options", "nosniff")
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
		tag := router.ParamFromContext(ctx, "tag")
		path := registry.PathJoinWithBase(name, tag)
		os.MkdirAll(path, 0700)
		path = filepath.Join(path, "manifest.json")
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
