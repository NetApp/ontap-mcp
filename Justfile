#!/usr/bin/env just --justfile

license-check:
    @go run github.com/frapposelli/wwhrd@latest check -q -t

lint:
	@go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run ./...
	@go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest ./...
	@go run golang.org/x/vuln/cmd/govulncheck@latest ./...

test:
    @go test ./...

ci: license-check lint test
