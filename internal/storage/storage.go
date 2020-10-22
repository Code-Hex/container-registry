package storage

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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

const baseTagDir = "tags"

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
	hash := sha256.New()
	reader := io.TeeReader(body, hash)
	var m registry.Manifest
	if err := json.NewDecoder(reader).Decode(&m); err != nil {
		return nil, errors.Wrap(err,
			errors.WithCodeManifestInvalid(),
		)
	}
	sha256sum := fmt.Sprintf("sha256:%x", hash.Sum(nil))

	// create directory
	path := registry.PathJoinWithBase(name, baseTagDir)
	os.MkdirAll(path, 0700)

	// create tag file
	tagPath := filepath.Join(path, tag)
	tagFile, err := os.Create(tagPath)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithCodeTagInvalid(),
		)
	}
	tagFile.Write([]byte(sha256sum))
	tagFile.Close()

	manifestPath := registry.PathJoinWithBase(name, sha256sum)
	os.MkdirAll(manifestPath, 0700)

	// create manifest file onto it
	manifestPath = filepath.Join(manifestPath, "manifest.json")
	manifestF, err := os.Create(manifestPath)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithCodeTagInvalid(),
		)
	}
	defer manifestF.Close()
	if err := json.NewEncoder(manifestF).Encode(&m); err != nil {
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
	tagFilePath := registry.PathJoinWithBase(name, baseTagDir, ref)
	if _, err := os.Stat(tagFilePath); err == nil {
		digest, err := ioutil.ReadFile(tagFilePath)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		ref = string(digest)
	}

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
	tagDir := registry.PathJoinWithBase(name, baseTagDir, tag)
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

// ListTags lists tags by image name.
func (l *Local) ListTags(name string) ([]string, error) {
	path := registry.PathJoinWithBase(name, baseTagDir)
	fis, err := ioutil.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.Wrap(err,
				errors.WithStatusCode(http.StatusNotFound),
			)
		}
		return nil, err
	}
	tags := make([]string, len(fis))
	for i, tag := range fis {
		tags[i] = tag.Name()
	}
	return tags, nil
}
