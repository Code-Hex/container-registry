package storage

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Code-Hex/container-registry/internal/errors"
	"github.com/Code-Hex/container-registry/internal/registry"
	"github.com/google/uuid"
)

// Repository represents the storage behavior.
type Repository interface {
	// Push
	IssueSession() string
	PutBlobByReference(ref string, imgName string, body io.Reader) (int64, error)
	EnsurePutBlobBySession(sessionID string, imgName string, digest string) error
	CheckBlobByDigest(imgName string, digest string) (os.FileInfo, error)
	CreateManifest(body io.Reader, name string, tag string) (*registry.Manifest, error)

	// Pull
	FindBlobByImage(name, digest string) (*os.File, error)
	FindManifestByImage(name, ref string) (*registry.Manifest, error)

	// Delete
	DeleteManifestByImage(name, tag string) error
	DeleteBlobByImage(name, digest string) error
}

var _ Repository = (*Local)(nil)

// Local implemented Repository using local storage.
type Local struct{}

// IssueSession issues session ID.
func (l *Local) IssueSession() string {
	return uuid.New().String()
}

// PutBlobByReference tries to put uploaded file on the reference directory.
//
// first, this method creates directory like "testdata/<image-name>/<reference>"
// then, put the layer file onto it.
func (l *Local) PutBlobByReference(ref string, imgName string, body io.Reader) (int64, error) {
	path := registry.PathJoinWithBase(imgName, ref)
	os.MkdirAll(path, 0700)
	return registry.CreateLayer(body, path)
}

// EnsurePutBlobBySession ensures the temporary path created by PutBlobBySession.
//
// this method moves from the temporary directory to "testdata/<image-name>/<digest>" directory
func (l *Local) EnsurePutBlobBySession(sessionID string, imgName string, digest string) error {
	newDir := registry.PathJoinWithBase(imgName, digest)
	os.MkdirAll(newDir, 0700)

	oldDir := registry.PathJoinWithBase(imgName, sessionID)
	fi, err := registry.PickupFileinfo(oldDir)
	if err != nil {
		return err
	}
	filename := fi.Name()
	oldpath := filepath.Join(oldDir, filename)
	newpath := filepath.Join(newDir, filename)
	if err := os.Rename(oldpath, newpath); err != nil {
		return err
	}
	os.Remove(oldDir)
	return nil
}

// CheckBlobByDigest checks for the existence of a blob with a digest.
func (l *Local) CheckBlobByDigest(imgName string, digest string) (os.FileInfo, error) {
	dir := registry.PathJoinWithBase(imgName, digest)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, errors.Wrap(err,
			errors.WithStatusCode(http.StatusNotFound),
		)
	}
	return registry.PickupFileinfo(dir)
}

// CreateManifest creates manifest json file by name and tag.
//
// this method creates to "<image-name>/<tag>/manifest.json"
func (l *Local) CreateManifest(body io.Reader, name string, tag string) (*registry.Manifest, error) {
	var m registry.Manifest
	if err := json.NewDecoder(body).Decode(&m); err != nil {
		return nil, errors.Wrap(err,
			errors.WithCodeManifestInvalid(),
		)
	}
	// create directory
	path := registry.PathJoinWithBase(name, tag)
	os.MkdirAll(path, 0700)

	// create manifest file onto it
	path = filepath.Join(path, "manifest.json")
	f, err := os.Create(path)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithCodeTagInvalid(),
		)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

// FindBlobByImage finds blob by docker image name and that's digest.
//
// digest format is like <digest-alg>:<digest>. see grammar.Digest
func (l *Local) FindBlobByImage(name, digest string) (*os.File, error) {
	dir := registry.PathJoinWithBase(name, digest)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, errors.Wrap(err,
			errors.WithCodeBlobUnknown(),
		)
	}
	fi, err := registry.PickupFileinfo(dir)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, fi.Name())
	return os.Open(path)
}

// FindManifestByImage finds manifest json file by image name and that's tag.
func (l *Local) FindManifestByImage(name, ref string) (*registry.Manifest, error) {
	manifest := registry.PathJoinWithBase(name, ref, "manifest.json")
	if _, err := os.Stat(manifest); os.IsNotExist(err) {
		return nil, errors.Wrap(err,
			errors.WithCodeManifestUnknown(),
		)
	}
	f, err := os.Open(manifest)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var m registry.Manifest
	if err := json.NewDecoder(f).Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

// DeleteManifestByImage deletes manifest json file by image name and that's tag.
func (l *Local) DeleteManifestByImage(name, tag string) error {
	tagDir := registry.PathJoinWithBase(name, tag)
	manifest := filepath.Join(tagDir, "manifest.json")
	if _, err := os.Stat(manifest); os.IsNotExist(err) {
		return errors.Wrap(err,
			errors.WithStatusCode(http.StatusBadRequest),
		)
	}
	return os.RemoveAll(tagDir)
}

// DeleteBlobByImage deletes blob by docker image name and that's digest.
//
// digest format is like <digest-alg>:<digest>. see grammar.Digest
func (l *Local) DeleteBlobByImage(name, digest string) error {
	dir := registry.PathJoinWithBase(name, digest)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return errors.Wrap(err,
			errors.WithCodeBlobUnknown(),
		)
	}
	return os.RemoveAll(dir)
}
