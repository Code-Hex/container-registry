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
	"strconv"
	"syscall"

	"github.com/Code-Hex/container-registry/internal/errors"
	"github.com/Code-Hex/container-registry/internal/grammar"
	"github.com/Code-Hex/container-registry/internal/registry"
	"github.com/Code-Hex/container-registry/internal/storage"
	"github.com/Code-Hex/go-router-simple"
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
		Handler: ServerApply(rs, AccessLogServerAdapter(), SetHeaderServerAdapter()),
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
	return Handler(func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte{'{', '}'})
		return nil
	})
}

// PullingBlobs to pull a blob.
func PullingBlobs() http.Handler {
	s := new(storage.Local)
	return Handler(func(w http.ResponseWriter, r *http.Request) error {
		ctx := r.Context()
		dq := router.ParamFromContext(ctx, "digest")
		dgst, err := digest.Parse(dq)
		if err != nil {
			return errors.Wrap(err,
				errors.WithCodeDigestInvalid(),
			)
		}
		name := router.ParamFromContext(ctx, "name")
		f, err := s.FindBlobByImage(name, dgst.String())
		if err != nil {
			return err
		}
		defer f.Close()
		w.Header().Set("Content-Type", registry.PredictDockerContentType(f.Name()))
		_, err = io.Copy(w, f)
		return err
	})
}

// PullingManifests to pull a manifest.
func PullingManifests() http.Handler {
	s := new(storage.Local)
	return Handler(func(w http.ResponseWriter, r *http.Request) error {
		ctx := r.Context()
		name := router.ParamFromContext(ctx, "name")
		tag := router.ParamFromContext(ctx, "tag")
		m, err := s.FindManifestByImage(name, tag)
		if err != nil {
			return err
		}
		w.Header().Set("Content-Type", registry.PredictDockerContentType("manifest.json"))
		return json.NewEncoder(w).Encode(m)
	})
}

func PushBlob() http.Handler {
	s := new(storage.Local)
	return Handler(func(w http.ResponseWriter, r *http.Request) error {
		sessionID := s.IssueSession()
		name := router.ParamFromContext(r.Context(), "name")
		location := "/v2/" + name + "/blobs/uploads/" + sessionID
		w.Header().Set("Location", location)
		w.WriteHeader(http.StatusAccepted)
		return nil
	})
}

func PushBlobPatch() http.Handler {
	s := new(storage.Local)
	return Handler(func(w http.ResponseWriter, r *http.Request) error {
		ctx := r.Context()
		name := router.ParamFromContext(ctx, "name")
		sessionID := router.ParamFromContext(ctx, "reference")
		size, err := s.PutBlobBySession(sessionID, name, r.Body)
		if err != nil {
			return err
		}
		location := "/v2/" + name + "/blobs/uploads/" + sessionID
		w.Header().Set("Location", location)
		w.Header().Set("Docker-Upload-UUID", sessionID)
		w.Header().Set("Range", fmt.Sprintf("0-%d", size))
		w.WriteHeader(http.StatusAccepted)
		return nil
	})
}

func PushBlobPut() http.Handler {
	s := new(storage.Local)
	return Handler(func(w http.ResponseWriter, r *http.Request) error {
		dgst, err := digest.Parse(r.URL.Query().Get("digest"))
		if err != nil {
			return errors.Wrap(err,
				errors.WithCodeDigestInvalid(),
			)
		}
		ctx := r.Context()
		name := router.ParamFromContext(ctx, "name")
		sessionID := router.ParamFromContext(ctx, "reference")
		if err := s.EnsurePutBlobBySession(sessionID, name, dgst.String()); err != nil {
			return err
		}
		w.WriteHeader(http.StatusCreated)
		return nil
	})
}

func PushBlobHead() http.Handler {
	s := new(storage.Local)
	return Handler(func(w http.ResponseWriter, r *http.Request) error {
		ctx := r.Context()
		dq := router.ParamFromContext(ctx, "digest")
		dgst, err := digest.Parse(dq)
		if err != nil {
			return errors.Wrap(err,
				errors.WithCodeDigestInvalid(),
			)
		}
		name := router.ParamFromContext(ctx, "name")
		fi, err := s.CheckBlobByDigest(name, dgst.String())
		if err != nil {
			return err
		}
		w.Header().Set("Content-Type", registry.PredictDockerContentType(fi.Name()))
		w.Header().Set("Docker-Content-Digest", dgst.String())
		w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
		w.WriteHeader(http.StatusAccepted)
		return nil
	})
}

func PushManifestPut() http.Handler {
	s := new(storage.Local)
	return Handler(func(w http.ResponseWriter, r *http.Request) error {
		ctx := r.Context()
		name := router.ParamFromContext(ctx, "name")
		tag := router.ParamFromContext(ctx, "tag")
		m, err := s.CreateManifest(r.Body, name, tag)
		if err != nil {
			return err
		}
		w.Header().Set("Docker-Content-Digest", m.Config.Digest.String())
		w.WriteHeader(http.StatusCreated)
		return nil
	})
}
