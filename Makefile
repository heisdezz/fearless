SHELL := /bin/sh

.PHONY: fmt vet lint staticcheck vulncheck build test dep-check ci

fmt:
	gofmt -w .

vet:
	go vet ./...

staticcheck:
	staticcheck ./...

vulncheck:
	govulncheck ./...

lint: fmt vet staticcheck

dep-check:
	./scripts/check-deps.sh

build:
	CGO_ENABLED=0 go build ./...

test:
	CGO_ENABLED=0 go test ./...

ci: dep-check
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed:"; gofmt -l .; exit 1)
	go vet ./...
	staticcheck ./...
	govulncheck ./...
	CGO_ENABLED=0 go build ./...
	CGO_ENABLED=0 go test ./...
