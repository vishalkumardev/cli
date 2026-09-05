VERSION ?= 0.1.0
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w \
	-X 'github.com/buildshare/cli/cmd/buildshare.Version=$(VERSION)' \
	-X 'github.com/buildshare/cli/cmd/buildshare.Commit=$(COMMIT)' \
	-X 'github.com/buildshare/cli/cmd/buildshare.BuildDate=$(DATE)'

.PHONY: build test lint clean release

## Build for current platform
build:
	go build -ldflags "$(LDFLAGS)" -o dist/buildshare ./main.go

## Run tests
test:
	go test ./... -v

## Lint
lint:
	golangci-lint run ./...

## Clean build artifacts
clean:
	rm -rf dist/

## Cross-compile for all platforms
release: clean
	@mkdir -p dist
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/buildshare-darwin-arm64     ./main.go
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/buildshare-darwin-amd64     ./main.go
	GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/buildshare-linux-arm64      ./main.go
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/buildshare-linux-amd64      ./main.go
	GOOS=windows GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/buildshare-windows-arm64.exe ./main.go
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/buildshare-windows-amd64.exe ./main.go
	@echo "✓ All binaries built in dist/"
	@ls -lh dist/

## Generate checksums
checksums:
	cd dist && shasum -a 256 buildshare-* > checksums.txt
	@cat dist/checksums.txt
