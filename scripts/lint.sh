#!/usr/bin/env bash

if ! command -v golangci-lint &>/dev/null; then
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$(go env GOPATH)/bin" v1.31.0
fi

golangci-lint run ./...
