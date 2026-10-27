GO ?= go

.PHONY: all install format check test pre-commit generate clean

all: test

install: test
	$(GO) install github.com/kode4food/argex/cmd/argex

format: generate
	$(GO) tool goimports -w .
	$(GO) fix ./...

check:
	$(GO) vet ./...
	$(GO) tool staticcheck ./...

test: generate check
	$(GO) clean -testcache
	$(GO) test ./...

pre-commit: format test

generate:
	$(GO) generate ./...

clean:
	$(GO) clean -testcache
