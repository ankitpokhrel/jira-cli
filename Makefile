.PHONY: deps build install lint test

deps:
	@echo "Installing dependencies..."
	go mod vendor -v

build: deps
	@echo "Building application..."
	CGO_ENABLED=0 go build ./...

install: deps
	@echo "Installing application..."
	CGO_ENABLED=0 go install ./...

lint:
	@scripts/lint.sh

test:
	@go clean -testcache ./...
	@go test -race $(shell go list ./...)
