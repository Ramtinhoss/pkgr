.PHONY: build test lint fmt vet tidy run clean

BINARY := pkgr
PKG    := github.com/ramtinhoss/pkgr
GO     ?= go

build:
	$(GO) build -trimpath -ldflags "-s -w -X main.version=dev" -o $(BINARY) ./cmd/pkgr

test:
	$(GO) test -race -coverprofile=coverage.txt ./...

lint:
	golangci-lint run

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

tidy:
	$(GO) mod tidy

run:
	$(GO) run ./cmd/pkgr

clean:
	rm -f $(BINARY) coverage.txt
	rm -rf dist
