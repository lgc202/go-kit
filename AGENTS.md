# Repository Guidelines

## Project Structure & Module Organization

- `config/`: Generic config loader built on Viper (file + env), with change watching. Examples: `config/examples/json`, `config/examples/yaml`.
- `version/`: Build/version helpers and CLI-friendly formatting. Examples: `version/example`, `version/example/cobra`.

## Build, Test, and Development Commands

- `go test ./...`: Run the full unit test suite.
- `go test ./... -race`: Race detector (slower, but preferred before merging concurrency-related changes).
- `go vet ./...`: Static checks for common issues.
- `go fmt ./...`: Format all packages (must be clean before review).
- Examples:
  - `go run ./config/examples/yaml`
  - `go run ./version/example`

If you run tests in a restricted/sandboxed environment where Go caches are not writable, set a local cache dir:
`GOCACHE=$(pwd)/.gocache GOMODCACHE=$(pwd)/.gomodcache go test ./...`.

## Coding Style & Naming Conventions

- Follow standard Go style: tabs for indentation and `gofmt`-formatted code.
- Keep public APIs stable; prefer additive changes (new options/types) over breaking renames.

## Testing Guidelines

- Use the standard `testing` package and table-driven tests (`t.Run`) as the default pattern.
- Avoid real network calls; stub HTTP via custom `http.RoundTripper`/`http.Client` as seen in provider tests.

## Commit & Pull Request Guidelines

- PRs should include: what changed, why, any public API impact, and the commands you ran (at least `go test ./...`). Update `README.md` files when adding or changing user-facing behavior.
