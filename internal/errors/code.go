package errors

import "net/http"

// WithCodeUnknown is a generic error that can be used as a last
// resort if there is no situation-specific error message that can be used
func WithCodeUnknown() WrapOption {
	return func(e *Error) {
		e.Code = "UNKNOWN"
		e.Message = "unknown error"
		e.StatusCode = http.StatusInternalServerError
	}
}

// WithCodeUnsupported is returned when an operation is not supported.
func WithCodeUnsupported() WrapOption {
	return func(e *Error) {
		e.Code = "UNSUPPORTED"
		e.Message = "The operation is unsupported."
		e.StatusCode = http.StatusMethodNotAllowed
	}
}

// ----- Error Code spec
//
// see: https://github.com/opencontainers/distribution-spec/blob/master/spec.md#error-codes
// -----

// WithCodeDigestInvalid is returned when uploading a blob if the
// provided digest does not match the blob contents.
func WithCodeDigestInvalid() WrapOption {
	return func(e *Error) {
		e.Code = "DIGEST_INVALID"
		e.Message = "provided digest did not match uploaded content"
		e.StatusCode = http.StatusBadRequest
	}
}

// WithCodeSizeInvalid is returned when uploading a blob if the provided
func WithCodeSizeInvalid() WrapOption {
	return func(e *Error) {
		e.Code = "SIZE_INVALID"
		e.Message = "provided length did not match content length"
		e.StatusCode = http.StatusBadRequest
	}
}

// WithCodeNameInvalid is returned when the name in the manifest does not
// match the provided name.
func WithCodeNameInvalid() WrapOption {
	return func(e *Error) {
		e.Code = "NAME_INVALID"
		e.Message = "invalid repository name"
		e.StatusCode = http.StatusBadRequest
	}
}

// WithCodeTagInvalid is returned when the tag in the manifest does not
// match the provided tag.
func WithCodeTagInvalid() WrapOption {
	return func(e *Error) {
		e.Code = "TAG_INVALID"
		e.Message = "manifest tag did not match URI"
		e.StatusCode = http.StatusBadRequest
	}
}

// WithCodeNameUnknown when the repository name is not known.
func WithCodeNameUnknown() WrapOption {
	return func(e *Error) {
		e.Code = "NAME_UNKNOWN"
		e.Message = "repository name not known to registry"
		e.StatusCode = http.StatusNotFound
	}
}

// WithCodeManifestUnknown returned when image manifest is unknown.
func WithCodeManifestUnknown() WrapOption {
	return func(e *Error) {
		e.Code = "MANIFEST_UNKNOWN"
		e.Message = "manifest unknown"
		e.StatusCode = http.StatusNotFound
	}
}

// WithCodeManifestInvalid returned when an image manifest is invalid,
// typically during a PUT operation. This error encompasses all errors
// encountered during manifest validation that aren't signature errors.
func WithCodeManifestInvalid() WrapOption {
	return func(e *Error) {
		e.Code = "MANIFEST_INVALID"
		e.Message = "manifest invalid"
		e.StatusCode = http.StatusBadRequest
	}
}

// WithCodeManifestUnverified is returned when the manifest fails
// signature verification.
func WithCodeManifestUnverified() WrapOption {
	return func(e *Error) {
		e.Code = "MANIFEST_UNVERIFIED"
		e.Message = "manifest failed signature verification"
		e.StatusCode = http.StatusBadRequest
	}
}

// WithCodeManifestBlobUnknown is returned when a manifest blob is
// unknown to the registry.
func WithCodeManifestBlobUnknown() WrapOption {
	return func(e *Error) {
		e.Code = "MANIFEST_BLOB_UNKNOWN"
		e.Message = "blob unknown to registry"
		e.StatusCode = http.StatusBadRequest
	}
}

// WithCodeBlobUnknown is returned when a blob is unknown to the
// registry. This can happen when the manifest references a nonexistent
// layer or the result is not found by a blob fetch.
func WithCodeBlobUnknown() WrapOption {
	return func(e *Error) {
		e.Code = "BLOB_UNKNOWN"
		e.Message = "blob unknown to registry"
		e.StatusCode = http.StatusNotFound
	}
}

// WithCodeBlobUploadUnknown is returned when an upload is unknown.
func WithCodeBlobUploadUnknown() WrapOption {
	return func(e *Error) {
		e.Code = "BLOB_UPLOAD_UNKNOWN"
		e.Message = "blob upload unknown to registry"
		e.StatusCode = http.StatusRequestedRangeNotSatisfiable
	}
}

// WithCodeBlobUploadInvalid is returned when an upload is invalid.
func WithCodeBlobUploadInvalid() WrapOption {
	return func(e *Error) {
		e.Code = "BLOB_UPLOAD_INVALID"
		e.Message = "blob upload invalid"
		e.StatusCode = http.StatusNotFound
	}
}
