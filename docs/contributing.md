# Contributing

## Prerequisites

- Go 1.24+
- Docker (for integration tests)
- `golangci-lint` (install via `brew install golangci-lint` or see golangci-lint.run)

## Running the Project Locally

```bash
# Clone and enter the repository
git clone https://github.com/marcusPrado02/go-commons
cd go-commons

# Start all infrastructure (Redis, RabbitMQ, LocalStack, Elasticsearch, OpenSearch)
docker compose up -d

# Run all tests
make test

# Run linter
make lint

# Check formatting
make fmt

# Check for vulnerabilities
make vulncheck
```

## Commit Conventions

This project follows [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <short description>

[optional body]
```

| Type | When to use |
|---|---|
| `feat` | New feature or new adapter |
| `fix` | Bug fix |
| `test` | Adding or updating tests |
| `docs` | Documentation only |
| `refactor` | Code change that isn't a fix or feature |
| `chore` | Tooling, CI, dependencies |

**Scope** is the package or adapter name: `feat(adapters/queue/rabbitmq)`, `fix(app/outbox)`, `docs(architecture)`.

Examples:
```
feat(adapters/queue/rabbitmq): add QueuePort implementation via amqp091-go
fix(app/resilience): call ValidatePolicies in Run() for fail-fast validation
test(adapters/files/s3): add LocalStack integration tests for Upload/Download/Delete
```

## Adding a New Adapter

1. **Define or reuse a port** in `ports/<capability>/port.go`. If a port already exists, skip this step.

2. **Create the adapter module:**
   ```bash
   mkdir -p adapters/<capability>/<provider>
   cd adapters/<capability>/<provider>
   ```

3. **Write `go.mod`:**
   ```
   module github.com/marcusPrado02/go-commons/adapters/<capability>/<provider>

   go 1.24

   replace github.com/marcusPrado02/go-commons => ../../..

   require (
       github.com/marcusPrado02/go-commons v0.0.0-00010101000000-000000000000
       <provider SDK>
   )
   ```

4. **Implement the port interface** in `<provider>.go` or `client.go`. Always add a compile-time check:
   ```go
   var _ port.MyPort = (*Client)(nil)
   ```

5. **Add the module to `go.work`:**
   ```
   use (
       ...
       ./adapters/<capability>/<provider>
   )
   ```

6. **Write tests:**
   - Use `httptest.Server` to stub HTTP-based providers without a real server.
   - For binary-protocol providers (AMQP, Redis), write integration tests with `t.Skip` when the service is unavailable.
   - Run `go mod tidy` after adding test dependencies.

7. **Run linter and tests:**
   ```bash
   go vet ./...
   go test ./...
   ```

8. **Update `TASKS.md`** if the adapter was listed there.

## Test Standards

- Use the standard library `testing` package — no test frameworks.
- Use table-driven tests (`tests := []struct{...}`) for multiple input variants.
- Integration tests must call `t.Skip` when external infrastructure is unavailable.
- Use `t.Cleanup` for resource teardown, not `defer` in test helpers.
- Each test should be independent — never share mutable state between test functions.
- Compile-time interface checks (`var _ port.X = (*Impl)(nil)`) must appear in every adapter test file.

## Contract Tests (testkit/contracts)

The `testkit/contracts` package provides reusable contract test suites for ports. If you're adding a new adapter for an existing port that has a contract suite, run it against your implementation:

```go
func TestMyAdapter_Contract(t *testing.T) {
    client := setupMyAdapter(t)
    contracts.RunCacheContract(t, client) // or EmailContract, QueueContract, etc.
}
```

Contract tests guarantee that your adapter behaves consistently with all other adapters for the same port.

## Code Style

- Follow standard Go formatting (`gofmt`). Run `make fmt` before committing.
- Do not add doc comments to unexported functions.
- All exported types, functions, and methods must have a doc comment.
- Error strings should be lowercase and not end with punctuation (Go convention).
- Use `fmt.Errorf("package: operation: %w", err)` for error wrapping.
- Avoid `panic` outside of `NewXxx` constructors with nil-validation. Document panics in the doc comment.

## Security

See [docs/security.md](security.md) for guidelines on:
- Not logging sensitive `Problem.Details`
- Using `LogSanitizer` before logging structured data
- Rotating secrets and never hardcoding credentials
