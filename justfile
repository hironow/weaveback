# weaveback - Weave Feedback Go Client

# Build all packages
build:
    go build ./...

# Run all tests
test:
    go test ./...

# Run linter (requires golangci-lint)
lint:
    go vet ./...

# Generate Go code from OpenAPI spec via oapi-codegen
generate:
    go generate ./pkg/weave/gen/

# Run integration tests (requires WANDB_API_KEY)
test-integration:
    go test -v -count=1 ./tests/integration/...

# Fetch OpenAPI spec from Weave API (see MY-393)
fetch-spec:
    @echo "fetch-spec: not yet configured (see MY-393)" >&2
    @exit 1
