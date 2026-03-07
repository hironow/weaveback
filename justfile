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

# Generate code from OpenAPI spec (requires oapi-codegen, see MY-394)
generate:
    @echo "generate: not yet configured (see MY-394)" >&2
    @exit 1

# Fetch OpenAPI spec from Weave API (see MY-393)
fetch-spec:
    @echo "fetch-spec: not yet configured (see MY-393)" >&2
    @exit 1
