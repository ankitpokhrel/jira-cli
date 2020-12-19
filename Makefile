.PHONY: deps build install lint test

# Build vars
git_commit  = $(shell git rev-parse HEAD)
build_date  = $(shell date +%FT%T%Z)
VERSION     ?= $(shell git symbolic-ref -q --short HEAD || git describe --tags --exact-match)
VERSION_PKG = github.com/ankitpokhrel/jira-cli/internal/version
LDFLAGS     := "-X $(VERSION_PKG).Version=$(VERSION) \
				-X $(VERSION_PKG).GitCommit=$(git_commit) \
				-X $(VERSION_PKG).BuildDate=$(build_date)"

deps:
	@echo "Installing dependencies..."
	go mod vendor -v

build: deps
	@echo "Building application..."
	CGO_ENABLED=0 go build -ldflags $(LDFLAGS) ./...

install: deps
	@echo "Installing application..."
	CGO_ENABLED=0 go install -ldflags $(LDFLAGS) ./...

lint:
	@scripts/lint.sh

test:
	@go clean -testcache ./...
	@go test -race $(shell go list ./...)
