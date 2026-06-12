VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0")
BINARY  = dist/emilyos
MODULE  = emilyos

.PHONY: build build-static test verify deb clean

build:
	GOWORK=off go build -trimpath -ldflags="-X main.Version=$(VERSION)" -o $(BINARY) ./cmd/emilyos

build-static:
	GOWORK=off CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath \
		-ldflags="-s -w -X main.Version=$(VERSION)" \
		-o $(BINARY) ./cmd/emilyos

test:
	GOWORK=off go test ./...

verify: test
	@echo "SHA256: $$(sha256sum $(BINARY) 2>/dev/null || echo '(not built)')"

deb: build-static
	VERSION=$(VERSION) ./packaging/scripts/build-deb.sh

clean:
	rm -rf dist/
