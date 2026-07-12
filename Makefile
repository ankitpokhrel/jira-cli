.PHONY: all deps build install ldflags lint golangci-lint-install golangci-lint-version test ci jira.server clean distclean

################
# Tool version #
################

GOLANGCI_LINT_VERSION ?= v2.12.2
# Installed into the repository's own bin/ (gitignored), not GOPATH/bin:
# enforcing the pin there would silently downgrade a golangci-lint the
# developer keeps for other projects.
GOLANGCI_LINT_BIN ?= $(CURDIR)/bin
GOLANGCI_LINT ?= $(GOLANGCI_LINT_BIN)/golangci-lint

##############
# Build vars #
##############

# https://git-scm.com/docs/git-stash#Documentation/git-stash.txt-create
#
# If uncommitted changes exist, then 'git stash create' will create a "stash
# entry" and print its object name; otherwise 'git stash create' will do
# nothing and print the empty string. In either case, 'git stash create'
# returns success.
#
# 'git rev-parse HEAD` (on success) prints the sha1sum of the current HEAD.
#
# Invoke both commands and take the first 40-xdigit string.
GIT_COMMIT ?= $(shell { git stash create; git rev-parse HEAD; } | grep -Exm1 '[[:xdigit:]]{40}')

# https://reproducible-builds.org/docs/source-date-epoch/
export SOURCE_DATE_EPOCH ?= $(shell git show -s --format="%ct" $(GIT_COMMIT))

VERSION ?= $(shell git symbolic-ref -q --short HEAD || git describe --tags --exact-match)
VERSION_PKG = github.com/rethab/jira-cli/internal/version
export LDFLAGS += -X $(VERSION_PKG).GitCommit=$(GIT_COMMIT)
export LDFLAGS += -X $(VERSION_PKG).SourceDateEpoch=$(SOURCE_DATE_EPOCH)
export LDFLAGS += -X $(VERSION_PKG).Version=$(VERSION)
export LDFLAGS += -s
export LDFLAGS += -w

export CGO_ENABLED ?= 0
export GOCACHE ?= $(CURDIR)/.gocache

all: build

deps:
	go mod vendor -v

build: deps
	go build -ldflags='$(LDFLAGS)' ./...

install:
	go install -ldflags='$(LDFLAGS)' ./...

# For callers that cannot use the targets above and must invoke the Go
# toolchain themselves, such as the cross-compiling Dockerfile.
ldflags:
	@echo '$(LDFLAGS)'

lint: golangci-lint-install
	$(GOLANGCI_LINT) run ./...

# Built from source rather than installed as a release binary: the published
# binaries are compiled with an older Go than the one in go.mod, and golangci-lint
# refuses to typecheck a language version newer than the one it was built with.
# `go install` compiles it with our toolchain, keeping the two in lockstep.
#
# The version has to be enforced, not merely looked for: whichever golangci-lint
# a developer happens to have would otherwise lint the tree, and a newer one
# reports findings that never gate the pull request while an older one misses
# findings that do.
golangci-lint-install:
	@if [ "$$($(GOLANGCI_LINT) version --short 2>/dev/null)" != "$(GOLANGCI_LINT_VERSION:v%=%)" ]; then \
		GOBIN="$(GOLANGCI_LINT_BIN)" go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
	fi

# So CI can key its linter cache on the pinned version without restating it.
golangci-lint-version:
	@echo '$(GOLANGCI_LINT_VERSION)'

test:
	@go clean -testcache
	CGO_ENABLED=1 go test -race $(GOTESTFLAGS) ./...

ci: lint test

jira.server:
	docker compose up -d

clean:
	go clean -x ./...

distclean:
	go clean -x -cache -testcache -modcache ./...
