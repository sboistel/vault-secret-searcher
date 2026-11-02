# vault-secret-searcher

A Go CLI tool that recursively searches HashiCorp Vault KV secrets for matching key names or values. Shells out to the `vault` CLI rather than using the Go SDK.

## Project Structure

- **`/cmd/vault-secret-searcher/`** — Entry point (`main.go`, `cli.go`). Kong CLI setup with debug flag.
- **`/pkg/cmd/search/`** — Search command. Recursive path walking, key/value matching, output formatting.
- **`/pkg/client/vault/`** — Vault client interface + CLI implementation. `Login()`, `ListPath()`, `GetSecret()`.

## Commands

One command: `search` (default, no need to type it).

```bash
vault-secret-searcher -k redis                # search keys in secret/
vault-secret-searcher -v "secret_value"       # search values
vault-secret-searcher -p secret/infra/ -k pg  # custom path
vault-secret-searcher -l -k redis             # OIDC login first
vault-secret-searcher -d -k redis             # debug logging
vault-secret-searcher -n my-namespace -k redis # specify namespace
```

## Development

### Prerequisites

- Go 1.26+
- `vault` CLI installed

### Build and test

```bash
go install ./cmd/vault-secret-searcher   # install to $GOPATH/bin
go test ./...                             # run tests
go tool golangci-lint run ./...           # lint
```

### Before committing

Always run in this order:

```bash
go tool gosimports -local "$(go list -m)" -w .
go mod tidy
go test ./...
go tool golangci-lint run ./...
```

CI enforces formatting and lint — code will be rejected if these fail.

## Patterns

### CLI framework

Uses [Kong](https://github.com/alecthomas/kong). Commands are structs with a `Run(ctx context.Context) error` method. Flags are struct field tags.

### Vault client

The `vault.Client` interface abstracts vault operations. `CLIClient` shells out to the `vault` binary — this avoids token management issues with the Go SDK. Tests use a mock implementation.

### Error handling

- Wrap errors with context: `fmt.Errorf("context: %w", err)`
- Permission denied and connectivity errors return real errors, not nil
- Missing paths return `(nil, nil)` — the caller treats this as "no data"

### Output

All output goes through `c.printf()` which writes to an `io.Writer` (defaults to stdout). Tests inject a `bytes.Buffer` instead. Matches are sorted alphabetically.

### KV v1 vs v2

`extractData()` in `pkg/client/vault/vault.go` handles both formats. KV v2 nests data at `.data.data`, KV v1 at `.data`.

### Logging

Uses `charmbracelet/log`. Debug output is enabled with `-d` flag. Do not use `fmt.Println` for debug output.

## Adding new commands

1. Create `pkg/cmd/<name>/<name>.go` with a `Cmd` struct and `Run()` method
2. Add the command to the `CLI` struct in `cmd/vault-secret-searcher/cli.go`
3. Add tests in `pkg/cmd/<name>/<name>_test.go`
