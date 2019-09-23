
version ?= $(subst v,,$(shell git describe --tags))
commit ?= $(shell git rev-parse --short HEAD)

.PHONY: test
test:
	golint ./...
	go vet ./...
	go test ./...

.PHONY: build
build:
	go build ./...

.PHONY: install
install:
	go install -ldflags "-X main.Version=$(version) -X main.Commit=$(commit)" ./...


