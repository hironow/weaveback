# weaveback

Go client library and CLI for the [Weave](https://wandb.ai/site/weave/) Feedback API.

## Overview

weaveback provides:

- **Library** (`pkg/weave/`): Go client wrapper for the Weave Feedback API with authentication, retry, and helper functions
- **CLI** (`cmd/weaveback/`): Command-line tool for feedback operations (create, query, replace, purge) with stdin pipe support

## Setup

```bash
# Install dependencies
go mod download

# Build
just build

# Run tests
just test

# Lint
just lint
```

## Project Structure

```
cmd/weaveback/    # CLI entrypoint
pkg/weave/        # Client library
pkg/weave/gen/    # Generated code from OpenAPI spec
api/              # OpenAPI spec files
internal/         # Internal packages
docs/             # Documentation
docs/adr/         # Architecture Decision Records
tests/unit/       # Unit tests
tests/integration/# Integration tests
tests/e2e/        # End-to-end tests
```

## License

Apache 2.0
