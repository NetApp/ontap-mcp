#!/usr/bin/env just --justfile

# lint
lint:
	@go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run ./...
	@go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest ./...

test:
    @go test ./...

ci:
    echo 'Running CI tasks...'
