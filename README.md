## Pull

### try on docker registry (based on https://containers.gitbook.io/build-containers-the-hard-way/)

```sh
$ TOKEN=$(curl "https://auth.docker.io/token?service=registry.docker.io&scope=repository:library/hello-world:pull" | jq .access_token -r)
$ curl https://registry.hub.docker.com/v2/library/hello-world/manifests/latest -H "Accept: application/vnd.docker.distribution.manifest.v2+json" -H "Authorization: Bearer $TOKEN"
{
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
}

$ curl "https://registry.hub.docker.com/v2/library/hello-world/blobs/sha256:bf756fb1ae65adf866bd8c456593cd24beb6a0a061dedf42b26a993176745f6b" -H "Authorization: Bearer $TOKEN" -L
{"architecture":"amd64","config":{"Hostname":"","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/hello"],"ArgsEscaped":true,"Image":"sha256:eb850c6a1aedb3d5c62c3a484ff01b6b4aade130b950e3bf3e9c016f17f70c34","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null},"container":"71237a2659e6419aee44fc0b51ffbd12859d1a50ba202e02c2586ed999def583","container_config":{"Hostname":"71237a2659e6","Domainname":"","User":"","AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh","-c","#(nop) ","CMD [\"/hello\"]"],"ArgsEscaped":true,"Image":"sha256:eb850c6a1aedb3d5c62c3a484ff01b6b4aade130b950e3bf3e9c016f17f70c34","Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":{}},"created":"2020-01-03T01:21:37.263809283Z","docker_version":"18.06.1-ce","history":[{"created":"2020-01-03T01:21:37.132606296Z","created_by":"/bin/sh -c #(nop) COPY file:7bf12aab75c3867a023fe3b8bd6d113d43a4fcc415f3cc27cbcf0fff37b65a02 in / "},{"created":"2020-01-03T01:21:37.263809283Z","created_by":"/bin/sh -c #(nop)  CMD [\"/hello\"]","empty_layer":true}],"os":"linux","rootfs":{"type":"layers","diff_ids":["sha256:9c27e219663c25e0f28493790cc0b88bc973ba3b1686355f221c38a36978ac63"]}}

$ curl "https://registry.hub.docker.com/v2/library/hello-world/blobs/sha256:0e03bdcc26d7a9a57ef3b6f1bf1a210cff6239bff7c8cac72435984032851689" -H "Authorization: Bearer $TOKEN" -L -o layer.tar.gz
```

### try docker pull on localhost

Need to fix `/etc/hosts` like below.

```
...

# Added by manually
127.0.0.1 codehex-local
# End of section
```

Then you can try to pull `$ docker pull codehex-local:5080/hello-world:latest`
