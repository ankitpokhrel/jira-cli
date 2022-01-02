.ONESHELL:
.PHONY: all deps build install lint test ci jira.server clean distclean

# Build vars
git_commit  = $(shell git rev-parse HEAD)
build_date  = $(shell date +%FT%T%Z)
VERSION     ?= $(shell git symbolic-ref -q --short HEAD || git describe --tags --exact-match)
VERSION_PKG = github.com/ankitpokhrel/jira-cli/internal/version
LDFLAGS     := "-X $(VERSION_PKG).Version=$(VERSION) \
				-X $(VERSION_PKG).GitCommit=$(git_commit) \
				-X $(VERSION_PKG).BuildDate=$(build_date)"

export GOCACHE ?= $(CURDIR)/.gocache

all: build

deps:
	go mod vendor -v

build: deps
	CGO_ENABLED=0 go build -ldflags $(LDFLAGS) ./...

install:
	CGO_ENABLED=0 go install -ldflags $(LDFLAGS) ./...

lint:
	@if ! command -v golangci-lint > /dev/null 2>&1; then
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
		sh -s -- -b "$$(go env GOPATH)/bin" v1.43.0
	fi
	golangci-lint run ./...

test:
	@go clean -testcache ./...
	go test -race $(shell go list ./...)

ci: lint test

jira.server:
	docker compose up -d

clean:
	go clean -x ./...

distclean:
	go clean -x -cache -testcache -modcache ./...
