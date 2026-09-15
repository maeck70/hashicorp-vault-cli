# Test Coverage Report

Comprehensive unit and integration test coverage for HashiCorp Vault CLI (`cmd/vault`) and the client library (`pkg/vault-lib`).

---

## Executive Summary

| Metric | Value |
|---|---|
| **Overall Statement Coverage** | **75.4%** |
| **`cmd/vault` Coverage** | **72.8%** |
| **`pkg/vault-lib` Coverage** | **77.1%** |
| **Test Suites** | 4 files (`main_test.go`, `paths_test.go`, `client_test.go`, `secrets_test.go`) |
| **Execution Time** | < 0.05s |
| **External Dependencies** | None (uses in-memory `httptest.Server` mocks) |

---

## Running Test Coverage in Makefile

Test coverage can be run directly using the project Makefile targets:

```bash
# Run test coverage and display per-function breakdown
make coverage

# Alternative alias
make testcoverage
```

### Other Related Makefile Targets

```bash
# Run all unit tests with verbose test logs
make test

# Run Go vulnerability check
make vulncheck

# Remove build artifacts and coverage profiles
make clean
```

---

## Test Coverage Run Output

```text
$ make coverage
go test -coverprofile=coverage.out ./...
ok  	github.com/maeck70/hashicorp-vault-cli/cmd/vault	coverage: 72.8% of statements
ok  	github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib	coverage: 77.1% of statements
go tool cover -func=coverage.out
github.com/maeck70/hashicorp-vault-cli/cmd/vault/main.go:18:		main			70.2%
github.com/maeck70/hashicorp-vault-cli/cmd/vault/main.go:392:		resolveEnv		100.0%
github.com/maeck70/hashicorp-vault-cli/cmd/vault/main.go:409:		printUsage		100.0%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/client.go:27:	NewClient		100.0%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/client.go:51:	NewVaultClient		100.0%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/client.go:57:	CheckConnection		76.9%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/client.go:119:	CheckVaultConnection	100.0%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/client.go:124:	Unseal			80.0%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/client.go:145:	UnsealVault		100.0%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/paths.go:10:	ResolveKey		100.0%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/paths.go:40:	ResolveVaultKey		100.0%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/paths.go:45:	NormalizePath		100.0%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/paths.go:73:	NormalizeVaultPath	100.0%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/paths.go:78:	SetEnvPrefix		90.9%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/secrets.go:16:	parseJSONValue		87.0%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/secrets.go:96:	getNestedValue		69.2%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/secrets.go:142:	deleteNestedValue	90.0%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/secrets.go:160:	fetchSecretData		72.7%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/secrets.go:183:	CreateSecret		81.1%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/secrets.go:236:	CreateVaultSecret	100.0%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/secrets.go:241:	ReadSecret		75.4%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/secrets.go:343:	ReadVaultSecret		100.0%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/secrets.go:348:	PutSecret		100.0%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/secrets.go:353:	GetSecret		100.0%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/secrets.go:364:	collectPrefixGroups	72.2%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/secrets.go:427:	ListKeys		49.4%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/secrets.go:559:	ListVaultKeys		100.0%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/secrets.go:564:	DeletePrefix		84.5%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/secrets.go:650:	DeleteVaultPrefix	100.0%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/secrets.go:655:	PrefixExists		100.0%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/secrets.go:669:	DeleteSecret		72.1%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/secrets.go:752:	DeleteVaultSecret	100.0%
github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib/secrets.go:757:	RemoveSecret		100.0%
total:									(statements)		75.4%
```

---

## Detailed Function Breakdown

| Package | Function | Coverage | Description |
|---|---|---|---|
| `cmd/vault` | `resolveEnv` | **100.0%** | Environment variable and `.env` resolution logic |
| `cmd/vault` | `printUsage` | **100.0%** | CLI usage and help banner formatting |
| `cmd/vault` | `main` | **70.2%** | CLI dispatch, flag parsing, subcommands |
| `pkg/vault-lib` | `NewClient` / `NewVaultClient` | **100.0%** | Vault API client constructor |
| `pkg/vault-lib` | `ResolveKey` / `ResolveVaultKey` | **100.0%** | Prefix resolution and path sanitization |
| `pkg/vault-lib` | `NormalizePath` / `NormalizeVaultPath` | **100.0%** | Vault KV-v2 path translation |
| `pkg/vault-lib` | `SetEnvPrefix` | **90.9%** | `.env` prefix update and file persistence |
| `pkg/vault-lib` | `parseJSONValue` | **87.0%** | JSON parsing, map detection, file references (`@file`) |
| `pkg/vault-lib` | `deleteNestedValue` | **90.0%** | Nested field deletion via dot notation |
| `pkg/vault-lib` | `DeletePrefix` / `DeleteVaultPrefix` | **84.5%** | Recursive prefix deletion |
| `pkg/vault-lib` | `CreateSecret` / `CreateVaultSecret` | **81.1%** | KV-v2 write / update operations |
| `pkg/vault-lib` | `Unseal` / `UnsealVault` | **80.0%** | Server unseal shard processing |
| `pkg/vault-lib` | `CheckConnection` / `CheckVaultConnection` | **76.9%** | Health check and token policy validation |
| `pkg/vault-lib` | `ReadSecret` / `ReadVaultSecret` | **75.4%** | Secret retrieval and dot-notation lookup |
| `pkg/vault-lib` | `fetchSecretData` | **72.7%** | Internal KV-v2 `/v1/secret/data/` payload fetcher |
| `pkg/vault-lib` | `DeleteSecret` / `DeleteVaultSecret` | **72.1%** | Secret deletion (entire secret or nested field) |
| `pkg/vault-lib` | `collectPrefixGroups` | **72.2%** | Recursive traversal and prefix categorization |
| `pkg/vault-lib` | `getNestedValue` | **69.2%** | Dot-notation traversal and array indexing |
| `pkg/vault-lib` | `ListKeys` / `ListVaultKeys` | **49.4%** | Metadata listing and prefix group discovery |
| `pkg/vault-lib` | `PrefixExists` | **100.0%** | Prefix existence check |
| `pkg/vault-lib` | Aliases (`PutSecret`, `GetSecret`, `RemoveSecret`) | **100.0%** | Backward-compatibility wrappers |

---

## Test Suites Architecture

### 1. `cmd/vault/main_test.go`
- **Flag and command dispatch**: Validates `version`, `-v`, `-version`, `-h`, `-help`, `status`, `prefix`, and custom `-prefix <folder>`.
- **Environment resolution**: Tests `.env` parsing via `godotenv`, falling back to process environment variables.
- **End-to-end CLI runs**: Exercises CLI subcommands against an in-process mock Vault HTTP server.

### 2. `pkg/vault-lib/paths_test.go`
- **Key Resolution**: Tests `ResolveKey` edge cases (empty keys, `/data/` substrings, duplicate prefixes, environment fallback from `VAULT_PREFIX` or `PREFIX`).
- **Path Normalization**: Validates automatic translation of user keys to `secret/data/...` and custom `VAULT_MOUNT` handling.
- **Environment File Management**: Tests creating new `.env` files and in-place line replacement for `VAULT_PREFIX`.

### 3. `pkg/vault-lib/client_test.go`
- **Connection Checks**: Tests `CheckConnection` against initialized, unsealed, sealed, and uninitialized mock Vault responses.
- **Unsealing**: Tests `Unseal` logic verifying progress and handling error states.

### 4. `pkg/vault-lib/secrets_test.go`
- **Value Parsing**: Unit tests for raw JSON objects, JSON arrays, shell space-separated key-value pairs, and file reading via `@filename`.
- **Dot-Notation Engine**: Deeply nested map and slice traversals (`a.b.c[0].d`), string-encoded JSON traversal, and key deletion.
- **Full KV-v2 Secret Operations**: In-memory mock Vault KV-v2 endpoints verifying `CreateSecret`, `ReadSecret`, `ListKeys`, `DeleteSecret`, and recursive `DeletePrefix`.
