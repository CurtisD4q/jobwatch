.PHONY: build test dist

build:
	CGO_ENABLED=0 go build -trimpath -o jobwatch ./cmd/jobwatch

test:
	test -z "$$(gofmt -l .)" && go vet ./... && go test -race ./...

# Archives for Linux amd64 and arm64 in dist/, with SHA256SUMS.
# TARGETS="linux/amd64 darwin/arm64" make dist for others.
dist:
	scripts/dist.sh
