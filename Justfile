#!/usr/bin/env just --justfile

RELEASE      := "1"
VERSION      := `date +%Y.%m.%d | cut -c 3-`
COMMIT       := `git rev-parse --short HEAD`
BUILD_DATE   := `date +%FT%T%z`
GOARCH       := "amd64"
GOOS         := `go env GOOS`
LD_FLAGS     := "-X 'version.VERSION={{VERSION}}' -X 'version.Release={{RELEASE}}' -X 'version.Commit={{COMMIT}}' -X 'version.BuildDate={{BUILD_DATE}}'"
BINARY_NAME  := "ontap-mcp"
DOCKER_TAG   := "ontap-mcp:latest"

# Automatically loads .env if it exists; no error if missing
set dotenv-load := true
# Specify a custom path (like your GO_ENV)
set dotenv-path := ".go.env"

license-check:
    @go run github.com/frapposelli/wwhrd@latest check -q -t

@_lint-impl target:
    cd {{target}} && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.9.0 run ./...
    cd {{target}} && go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest $(go list ./... | grep -v /third_party/)
    cd {{target}} && go run golang.org/x/vuln/cmd/govulncheck@latest ./...

@lint:
    just _lint-impl .
    just _lint-impl integration

@test:
    go test ./...

docs:
    mkdocs serve

checks: license-check lint test

build: lint ## Build the ONTAP MCP server binary with development checks
	@echo "Building ONTAP MCP server..."
	@GOOS={{GOOS}} GOARCH={{GOARCH}} CGO_ENABLED=0 go build -trimpath -ldflags="{{LD_FLAGS}}" -o {{BINARY_NAME}} .
	@echo "✅ Build complete: {{BINARY_NAME}}"

docker-build: ## Build Docker image (use DOCKER_TAG to customize tag, e.g., just docker-build DOCKER_TAG=ontap-mcp:dev)
	@echo "Building Docker image..."
	@docker build -f Dockerfile --build-arg GO_VERSION=${GO_VERSION} -t {{DOCKER_TAG}} .
	@echo "✅ Docker image built: {{DOCKER_TAG}}"

ci: ## Run integration tests
    #!/usr/bin/env bash
    set -euo pipefail
    TEST_SUFFIX="$(date +%s)"
    VERSION="${VERSION:-$(date +%Y.%m.%d | cut -c 3-)}"
    LD_FLAGS="-X 'github.com/netapp/ontap-mcp/version.VERSION=${VERSION}'"
    echo "Integration CI: TEST_SUFFIX=$TEST_SUFFIX  VERSION=$VERSION"
    export TEST_SUFFIX
    export CHECK_TOOLS=1
    cd integration/test
    go mod tidy
    go test -v -timeout 1h -ldflags="$LD_FLAGS"