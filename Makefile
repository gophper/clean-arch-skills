.PHONY: scaffold run test help

## scaffold: (Re)generate examples/minimal/ demo — idempotent, safe to run multiple times.
scaffold:
	go run ./cmd/scaffold

## run: Generate demo then start the HTTP server on :8080.
run: scaffold
	cd examples/minimal && go run ./cmd/demo

## test: Generate demo then run all unit tests (no DB / MQ required).
test: scaffold
	cd examples/minimal && go test ./... -v

## help: Show this help.
help:
	@grep -E '^## ' Makefile | sed 's/^## //'
