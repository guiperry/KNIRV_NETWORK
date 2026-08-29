# KNIRVENGINE Alpine container deployer

This is the KNIRVENGINE clone of the server container-deployer convention. It
builds `knirvengine-alpine:latest` from the Alpine OS definition in
`../os_builder/alpine/Dockerfile`, can run the opt-in eBPF/cgroup profile, and
can tag/push the same image to Docker Hub.

## Build and run

From `packages/KNIRVENGINE`:

```bash
go run ./cmd/os_builder -action 1
go run ./cmd/container_deployer -run
```

The runtime compose file is deliberately privileged: eBPF tracing, joining a
sandbox cgroup namespace, and writable cgroup control cannot be provided by a
Dockerfile alone. It gives the container host-level kernel visibility (`pid:
host`, host cgroup namespace, `seccomp=unconfined`, and `privileged: true`).
Only use it on a dedicated trusted host; do not expose this container to
untrusted users or workloads.

## Docker Hub

1. Create the `knirvengine` repository under the Docker Hub namespace you
   intend to use (for example `knirvcorp/knirvengine`).
2. Authenticate locally: `docker login` (prefer a Docker Hub personal access
   token rather than an account password).
3. Build, tag, and push the public Alpine image:

```bash
go run ./cmd/container_deployer -push
```

This pushes `knirvcorp/knirvengine:alpine-latest`. Docker Hub credentials are
never read from or baked into the image.

For CI, store a repository-scoped Docker Hub access token in the CI secret
store and run `docker login --username "$DOCKERHUB_USERNAME" --password-stdin`.
