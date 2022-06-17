# Usage:
#   $ docker build -t jira-cli:latest .
#   $ docker run --rm -it -v ~/.netrc:/root/.netrc -v ~/.config/.jira:/root/.config/.jira jira-cli

FROM golang:1.17.11-alpine3.16 as builder

ENV CGO_ENABLED=0
ENV GOOS=linux

WORKDIR /app

COPY . .

RUN set -eux; \
    env ; \
    ls -la ; \
    apk add -U --no-cache make git ; \
    make deps install

FROM alpine:3.16

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /go/bin/jira /bin/jira

ENTRYPOINT ["/bin/sh"]
