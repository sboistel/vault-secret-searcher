# vault-secret-searcher

A CLI tool that recursively searches HashiCorp Vault KV secrets for matching key names or values.

## Install

```bash
go install ./cmd/vault-secret-searcher
```

Make sure `$GOPATH/bin` is on your `$PATH`. Add this to your `~/.zshrc` if it's not:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

## Usage

```bash
# Search key names in secret/
vault-secret-searcher -k redis

# Search values
vault-secret-searcher -v "secret_value"

# Search a different path
vault-secret-searcher -p secret/infra/ -v redis

# Login to Vault first, then search
vault-secret-searcher -l -k redis

# Search within a Vault namespace (prompts OIDC login if the token is invalid there)
vault-secret-searcher -n my-namespace -k redis

# Enable debug logging
vault-secret-searcher -d -k redis
```

## Flags

| Flag | Description |
|------|-------------|
| `-k TERM` | Search key names (case-insensitive) |
| `-v TERM` | Search values (case-insensitive) |
| `-p` | Vault path (default: `secret/`) |
| `-l` | Run `vault login -method oidc` before searching |
| `-n` | Vault namespace (default: `$VAULT_NAMESPACE`) |
| `-d` | Enable debug logging |

## Requirements

- `vault` CLI installed and configured
- Authenticated Vault session (or use `-l` to login)
