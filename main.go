package main

import (
	"context"
	"encoding/json"
	"errors"
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

	"github.com/Code-Hex/go-router-simple"
	"github.com/google/uuid"
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

// Grammar: https://github.com/docker/distribution/blob/2800ab02245e2eafc10e338939511dd1aeb5e135/reference/reference.go#L4-L24
const (
	reference = name + (`(?::` + tag + `)?`) + (`(?:@` + digest + `)?`)

	name            = (`(?:` + domain + `\/)?`) + pathComponent + (`(?:\/` + pathComponent + `)*`)
	domain          = domainComponent + (`(?:\.` + domainComponent + `)*`) + ("(?::" + portNumber + ")?")
	domainComponent = `(?:[a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9])`
	portNumber      = `[0-9]+`
	tag             = `[\w][\w.-]{0,127}`
	pathComponent   = alphaNumeric + (`(?:` + separator + alphaNumeric + `)*`)
	alphaNumeric    = `[a-z0-9]+`
	separator       = `[_.]|__|[-]*`

	digest                   = digestAlgorithm + ":" + digestHex
	digestAlgorithm          = digestAlgorithmComponent + (`(?:` + digestAlgorithmSeparator + digestAlgorithmComponent + `)*`)
	digestAlgorithmSeparator = `[+.-_]`
	digestAlgorithmComponent = `[A-Za-z][A-Za-z0-9]*`
	digestHex                = `[0-9a-fA-F]{32,}`
)

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
			name, digestHex,
		),
		PullingBlobs(),
	)

	// /v2/:name/manifests/:reference
	rs.GET(
		fmt.Sprintf(
			`/v2/{name:%s}/manifests/{reference:%s}`,
			name, reference,
		),
		PullingManifests(),
	)

	// /?digest=<digest>
	rs.POST(
		fmt.Sprintf(
			`/v2/{name:%s}/blobs/uploads/`,
			name,
		),
		PushBlob(),
	)

	rs.PATCH(
		fmt.Sprintf(
			`/v2/{name:%s}/blobs/uploads/{reference:%s}`,
			name, reference,
		),
		PushBlobPatch(),
	)

	// /?digest=<digest>
	rs.PUT(
		fmt.Sprintf(
			`/v2/{name:%s}/blobs/uploads/{reference:%s}`,
			name, reference,
		),
		PushBlobPut(),
	)

	rs.HEAD(
		fmt.Sprintf(
			`/v2/{name:%s}/blobs/{digest:%s}`,
			name, digest,
		),
		PushBlobHead(),
	)

	rs.PUT(
		fmt.Sprintf(
			`/v2/{name:%s}/manifests/{tag:%s}`,
			name, tag,
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
		digest := router.ParamFromContext(ctx, "digest")
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

		path := filepath.Join("testdata", reference)
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
		digest := r.URL.Query().Get("digest")
		sep := strings.SplitN(digest, ":", 2)
		if len(sep) != 2 {
			log.Println(sep)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		digestHex := sep[1]

		os.MkdirAll("testdata/"+digestHex, 0700)

		ctx := r.Context()
		reference := router.ParamFromContext(ctx, "reference")
		uuid := reference
		oldpath := filepath.Join("testdata", uuid, "layer.tar.gz")
		newpath := filepath.Join("testdata", digestHex, "layer.tar.gz")
		if err := os.Rename(oldpath, newpath); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		os.Remove("testdata/" + uuid)

		w.WriteHeader(http.StatusCreated)
	})
}

func PushBlobHead() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		digest := router.ParamFromContext(r.Context(), "digest")
		w.Header().Set("Docker-Content-Digest", digest)
		w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
		w.Header().Set("Content-Length", strconv.Itoa(len(helloworldManifest)))
		w.WriteHeader(http.StatusAccepted)
	})
}

func PushManifestPut() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m Manifest
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		//ctx := r.Context()
		w.Header().Set("Docker-Content-Digest", m.Config.Digest.String())
		w.WriteHeader(http.StatusCreated)
	})
}
