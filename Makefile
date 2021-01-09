.ONESHELL:
.PHONY: all deps build install lint test ci

# Build vars
git_commit  = $(shell git rev-parse HEAD)
build_date  = $(shell date +%FT%T%Z)
VERSION     ?= $(shell git symbolic-ref -q --short HEAD || git describe --tags --exact-match)
VERSION_PKG = github.com/ankitpokhrel/jira-cli/internal/version
LDFLAGS     := "-X $(VERSION_PKG).Version=$(VERSION) \
				-X $(VERSION_PKG).GitCommit=$(git_commit) \
				-X $(VERSION_PKG).BuildDate=$(build_date)"

all: deps lint test install

deps:
	go mod vendor -v

build: deps
	CGO_ENABLED=0 go build -ldflags $(LDFLAGS) ./...

install:
	CGO_ENABLED=0 go install -ldflags $(LDFLAGS) ./...

lint:
	@scripts/lint.sh

test:
	@go clean -testcache ./...
	go test -race $(shell go list ./...)

ci: lint test
