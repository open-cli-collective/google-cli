VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-s -w \
	-X github.com/open-cli-collective/google-cli/internal/version.Version=$(VERSION) \
	-X github.com/open-cli-collective/google-cli/internal/version.Commit=$(COMMIT) \
	-X github.com/open-cli-collective/google-cli/internal/version.Date=$(DATE)"

export GOFLAGS := -tags=keyring_no1password,keyring_nopassage

.PHONY: build test test-cover test-cover-check lint fmt tidy check install

build:
	go build $(LDFLAGS) -o bin/gro ./cmd/gro
	go build $(LDFLAGS) -o bin/grw ./cmd/grw

test:
	go test -race ./...

test-cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-cover-check:
	@go test -race -coverprofile=coverage.out ./... > /dev/null 2>&1
	@total=$$(go tool cover -func=coverage.out | grep '^total:' | awk '{print $$3}' | tr -d '%'); \
	threshold=60; \
	echo "Total coverage: $${total}% (threshold: $${threshold}%)"; \
	if [ $$(echo "$$total < $$threshold" | bc) -eq 1 ]; then \
		echo "FAIL: coverage below threshold"; exit 1; \
	fi

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...
	goimports -local github.com/open-cli-collective/google-cli -w .

tidy:
	go mod tidy
	git diff --exit-code go.mod go.sum

check: tidy lint test build

install: build
	install -m 755 bin/gro bin/grw /usr/local/bin/
