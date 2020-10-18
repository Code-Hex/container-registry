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

type Repository interface {
	IssueSession() string
	PutBlobBySession(sessionID string, imgName string, body io.Reader) (int64, error)
	EnsurePutBlobBySession(sessionID string, imgName string, digest string) error
	CheckBlobByDigest(digest string) (string, error)
	CreateManifest(body io.Reader, name string, tag string) (*registry.Manifest, error)
}

// Local implemented Repository using local storage.
type Local struct{}

// IssueSession issues session ID.
func (l *Local) IssueSession() string {
	return uuid.New().String()
}

// PutBlobBySession tries to put uploaded file on the sessionID directory.
//
// first, this method creates directory like "testdata/<image-name>/<session-id>"
// then, put the layer file onto it.
func (l *Local) PutBlobBySession(sessionID string, imgName string, body io.Reader) (int64, error) {
	path := registry.PathJoinWithBase(imgName, sessionID)
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
