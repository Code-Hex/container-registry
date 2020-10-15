package main

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// https://docs.docker.com/registry/spec/manifest-v2-2/
const helloworldManifest = `{
	"schemaVersion": 2,
	"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
	"config": {
	   "mediaType": "application/vnd.docker.container.image.v1+json",
	   "size": 1510,
	   "digest": "sha256:bf756fb1ae65adf866bd8c456593cd24beb6a0a061dedf42b26a993176745f6b"
	},
	"layers": [
	   {
		  "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
		  "size": 2529,
		  "digest": "sha256:0e03bdcc26d7a9a57ef3b6f1bf1a210cff6239bff7c8cac72435984032851689"
	   }
	]
 }`

type Manifest struct {
	SchemaVersion int                  `json:"schemaVersion"`
	MediaType     string               `json:"mediaType"`
	Config        ocispec.Descriptor   `json:"config"`
	Layers        []ocispec.Descriptor `json:"layers"`
}
