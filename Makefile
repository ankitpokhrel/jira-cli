.PHONY: deps build install lint

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
	@bin/lint.sh
