# Usage:
#   $ docker build -t jira-cli:latest .
#   $ docker run --rm -it -v ~/.netrc:/root/.netrc -v ~/.config/.jira:/root/.config/.jira jira-cli

# Pinned to the build platform so Go cross-compiles for the target instead of
# the whole toolchain running emulated under QEMU. The image is built for four
# platforms on every merge to main, and emulation makes that take hours.
#
# Pinned to the same patch as the `toolchain` directive in go.mod: whenever a
# floating tag lags it, GOTOOLCHAIN=auto downloads the pinned toolchain during
# every one of the four platform builds, and fails outright without a proxy.
FROM --platform=$BUILDPLATFORM golang:1.26.5-alpine3.24 AS builder

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

# CI passes the branch or tag being published; a tag's v prefix is stripped
# below because GoReleaser stamps release binaries without it. Left empty,
# the Makefile falls back to deriving it from git, which is what a local
# `docker build` wants.
ARG VERSION=

ENV CGO_ENABLED=0

WORKDIR /app

COPY . .

RUN apk add -U --no-cache make git

# `go install` refuses to cross-compile into GOBIN, hence the explicit
# `go build`. The linker flags still come from the Makefile so the version
# stamped here matches the one in a release binary.
RUN set -eux; \
    export GOOS="$TARGETOS" GOARCH="$TARGETARCH"; \
    if [ -n "$TARGETVARIANT" ]; then export GOARM="${TARGETVARIANT#v}"; fi; \
    if [ -z "$VERSION" ]; then unset VERSION; else export VERSION="${VERSION#v}"; fi; \
    make deps; \
    go build -trimpath -ldflags="$(make -s ldflags)" -o /out/jira ./cmd/jira

# Kept free of RUN instructions: they would execute on the target platform
# and drag QEMU back into CI. The TLS certificates that apk used to install
# here ship with the base image (ca-certificates-bundle) since Alpine 3.9.
FROM alpine:3.24

WORKDIR /root/

COPY --from=builder /out/jira /bin/jira

ENTRYPOINT ["/bin/sh"]
