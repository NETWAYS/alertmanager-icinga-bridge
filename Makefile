.PHONY: test coverage lint vet

CONTAINER_RUNTIME?=podman

COMMIT := $(shell git rev-parse HEAD)
DATE := $(shell date --iso-8601)
VERSION?=latest

build-image:
	$(CONTAINER_RUNTIME) build --pull \
        --build-arg BRIDGE_VERSION=$(VERSION) \
        --build-arg BRIDGE_COMMIT=$(COMMIT) \
        --build-arg BRIDGE_DATE=$(DATE) \
        -t ghcr.io/netways/alertmanager-icinga-bridge:latest .
build:
	mkdir -p dist; \
	CGO_ENABLED=0 go build -o dist/alertmanager-icinga-bridge
release-snapshot:
	goreleaser release --snapshot --clean
release:
	mkdir -p dist;\
	CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)" -o dist/alertmanager-icinga-bridge
lint:
	go fmt $(go list ./... | grep -v /vendor/)
vet:
	go vet $(go list ./... | grep -v /vendor/)
test:
	go test -v -cover ./...
coverage:
	go test -v -cover -coverprofile=coverage.out ./... &&\
	go tool cover -html=coverage.out -o coverage.html
clean:
	rm -f dist/*
