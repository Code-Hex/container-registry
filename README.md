# Container Registry

![test](https://github.com/Code-Hex/container-registry/workflows/test/badge.svg) ![e2e test](https://github.com/Code-Hex/container-registry/workflows/e2e%20test/badge.svg) ![OCI Conformance Tests](https://github.com/Code-Hex/container-registry/workflows/OCI%20Conformance%20Tests/badge.svg)

The Container Registry is implemented using the file system.

- ✅ Implemented of the [OCI Distribution Spec](https://github.com/opencontainers/distribution-spec/blob/master/spec.md)
  - [x] Pull
  - [x] Push
  - [x] Content Discovery
  - [x] Content Management
- ✅ Supported Docker Client

## Why developed?

I have developed this container registry for learning purposes. And the reason why I implemented this using the file system because I thought it would be easier to understand how images are stored and what kind of files are stored.

⚠ BTW, I do not recommend that you run this application in production.

## How to try this on your localhost

Need to fix `/etc/hosts` like below.

```
...

# Added by manually
127.0.0.1 container-registry
# End of section
```


- Build binary - `make build`
- Run Container Registry - `./bin/registry`

If you want to clean up in `testdata` directry, let's use `make clean`.

### Push

1. Pull any images to push - `docker pull registry:2`
2. Tag to push - `docker tag $(docker images --format '{{.ID}}' --filter=reference='registry') container-registry:5080/registry:latest`
3. Try push - `docker push container-registry:5080/registry:latest`

## Pull

Try pull if you have completed push steps

```sh
$ docker pull container-registry:5080/registry:latest
```

## debug

### docker daemon

**MacOS**

```
$ tail -f ~/Library/Containers/com.docker.docker/Data/log/vm/dockerd.log
```

## References

- [Container image registry - Build Containers the Hard Way (WIP)](https://containers.gitbook.io/build-containers-the-hard-way/#container-image-registry)
- [Open Container Initiative Distribution Specification](https://github.com/opencontainers/distribution-spec/blob/master/spec.md)