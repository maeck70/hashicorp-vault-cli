# HashiCorp Vault CLI (`vault`)

A fast, lightweight, and intuitive command-line interface written in Go for managing secrets on HashiCorp Vault (KV v2 secrets engine).

---

## Features

- **Intuitive Subcommands**: First-class command support (`read`, `write`, `delete`, `list`, `prefix`, `unseal`, `status`).
- **Flexible Path Prefixes**: Set a default prefix in `.env` or override per command (`-prefix <folder>`).
- **In-Terminal Prefix Management**: Inspect or change your active prefix dynamically with `vault prefix <name>`.
- **Rich JSON & Dot Notation**:
  - Read and write complex nested JSON objects or arrays.
  - Query nested fields directly via dot notation (e.g., `mysql.primary.host` or array elements `items[0]`).
  - Delete individual JSON keys in-place without destroying the parent secret.
  - Automatic parsing of JSON files (`@path/to/file.json` or `file.json`) and raw strings.
- **Automatic Unsealing**: Automatically unseals the Vault server on launch if `VAULT_UNSEAL_KEY` is present in `.env`.
- **KV v2 Normalization**: Automatically translates simple keys (e.g. `captain`) into canonical Vault KV v2 paths (`secret/data/captain`).
- **Full CLI Aliases**: Supports both natural subcommands and classic flags (`read` / `-read`, `write` / `-write`, `list` / `-list`, `delete` / `-delete`).
- **Standard Build Target**: Builds directly into `bin/vault`.

---

## Configuration (`.env`)

Configure your connection details in `.env` (or copy from `.env.example`):

```env
# Vault Server Address
VAULT_ADDR=http://127.0.0.1:8200

# Vault Authentication Token
VAULT_TOKEN=your-vault-token-here

# Vault Unseal Key Shard (optional, enables automatic unsealing)
VAULT_UNSEAL_KEY=your-unseal-key-here

# Default Secret Path Prefix / Folder
VAULT_PREFIX=myserver

# Vault Namespace (optional)
VAULT_NAMESPACE=your-namespace-here
```

---

## Installation & Build

### 1. Direct Installation via Go (Straight from GitHub)

Install the `vault` executable directly into your `$GOPATH/bin`:

```bash
go install github.com/maeck70/hashicorp-vault-cli/cmd/vault@latest
```

This compiles and places the binary named `vault` directly into your `$GOPATH/bin` (or `$(go env GOPATH)/bin`). Ensure this directory is in your `$PATH`, then run:

```bash
vault status
```

### 2. Build from Source (Git Clone)

Clone the repository and build the binary into `./bin/vault`:

```bash
git clone https://github.com/maeck70/hashicorp-vault-cli.git
cd hashicorp-vault-cli
make build
```

The executable is compiled to `./bin/vault`.

---

## Command Reference

```bash
./bin/vault <command> [arguments] [flags]
```

### 1. View or Set Prefix (`prefix`)
Manage the default secret folder / prefix stored in `.env`:

```bash
# View active default prefix
./bin/vault prefix

# Set new default prefix in .env
./bin/vault prefix myserver
```

### 2. Write a Secret (`write` / `create` / `put` / `set` / `add`)
Write a secret key and value. Automatically uses the active prefix:

```bash
./bin/vault write database_url "postgres://user:pass@localhost:5432/db"
./bin/vault create database_url "postgres://user:pass@localhost:5432/db"
./bin/vault put database_url "postgres://user:pass@localhost:5432/db"

# Or override the prefix on the fly:
./bin/vault -prefix production write database_url "postgres://prod-db:5432/db"

# Store JSON documents directly:
./bin/vault write mysql '{"username":"guest","password":"muysecreto","host":"192.168.10.10","port":3306}'

# Or load directly from a JSON file:
./bin/vault write mysql @config.json
./bin/vault write mysql config.json
```

### 3. Read a Secret (`read` / `get`)
Retrieve a secret by key, either as a whole block or by individual multi-level components:

```bash
# Read simple secret
./bin/vault read database_url
./bin/vault get database_url

# Read whole JSON block
./bin/vault read mysql

# Read individual components using dot notation:
./bin/vault read mysql.username
./bin/vault read mysql.password
./bin/vault read mysql.host
./bin/vault read mysql.port

# Supports arbitrary multi-level nested JSON paths:
./bin/vault read mysql.cluster.primary.host
```

### 4. Working with JSON Files & Nested Secrets
You can store complex structured configuration directly from local JSON files and extract or mutate individual components without downloading or rewriting the entire payload.

#### Writing from a JSON File
Given a file `db-config.json`:
```json
{
  "host": "postgres.internal",
  "port": 5432,
  "credentials": {
    "username": "app_user",
    "password": "supersecretpassword"
  },
  "replicas": [
    {"host": "replica1.internal", "port": 5432},
    {"host": "replica2.internal", "port": 5432}
  ]
}
```

Upload the file directly into Vault:
```bash
# Using @ prefix (curl-style):
./bin/vault write db @db-config.json

# Or passing the filename directly:
./bin/vault write db db-config.json
```

#### Retrieving Individual Components (Dot Notation)
Retrieve either the complete JSON document or drill down to specific nested components:

```bash
# Retrieve full JSON payload:
./bin/vault read db

# Retrieve a top-level component:
./bin/vault read db.host
# => postgres.internal

# Retrieve nested object properties:
./bin/vault read db.credentials.username
# => app_user

./bin/vault read db.credentials.password
# => supersecretpassword

# Retrieve array elements and nested fields:
./bin/vault read db.replicas[0].host
# => replica1.internal

./bin/vault read db.replicas[1].port
# => 5432
```

#### Deleting an Individual Component
You can also remove an individual property within a JSON secret without affecting other fields:
```bash
./bin/vault delete db.credentials.password
```

### 5. List Keys (`list` / `ls`)
List secret keys under the current prefix, a specific subpath, or group across all prefixes:

```bash
# List all keys under active default prefix (reports 'Prefix: <prefix>')
./bin/vault list
./bin/vault ls

# List keys under a specific subpath/prefix
./bin/vault list other_folder

# List keys when no prefix is set or with root /:
# Automatically discovers and groups keys by prefix!
./bin/vault list /
./bin/vault -prefix "" list
```

### 6. Delete Secrets & Full Prefixes (`delete` / `del` / `rm` / `remove` / `delete-prefix`)
Delete a single secret, a JSON component, or recursively delete an entire prefix:

```bash
# Delete a single secret
./bin/vault delete database_url
./bin/vault rm database_url

# Delete an individual component from a JSON secret:
./bin/vault delete mysql.username

# Delete an entire prefix and all its secrets recursively:
./bin/vault delete -r rabbitmq
./bin/vault delete-prefix rabbitmq
./bin/vault delete rabbitmq/

# Delete all secrets under the active default prefix:
./bin/vault delete -r /
```

### 7. Unseal Vault (`unseal`)
Submit an unseal shard key:

```bash
# Uses VAULT_UNSEAL_KEY from .env
./bin/vault unseal

# Or provide key explicitly
./bin/vault unseal 3c12834f66ad4f77aad5c8bdef4ddbb5f986c4c260f0484ddd3baba986a9262e
```

### 8. Status & Health Check (`status`)
Inspect server status, seal state, version, and active token policies:

```bash
./bin/vault status
```

---

## Global Flags

All flags can be placed before or after subcommands:

| Flag | Description | Default |
|------|-------------|---------|
| `-v`, `-version` | Output version string (`HashiCorp Vault v0.5.0`) | — |
| `-h`, `-help` | Display program info, function, usage, and command list | — |
| `-V`, `--V`, `-verbose`, `--verbose` | Enable verbose output (connection banners, health & token details, debug info) | `false` (quiet mode) |
| `-r`, `-recursive`, `-all` | Recursively delete all secrets under target prefix | `false` |
| `-prefix, -p <path>` | Override secret path prefix for this command | `.env` `VAULT_PREFIX` |
| `-addr <url>` | Override Vault server address | `.env` `VAULT_ADDR` |
| `-token <token>` | Override authentication token | `.env` `VAULT_TOKEN` |
| `-namespace <ns>` | Override Vault namespace | `.env` `VAULT_NAMESPACE` |
| `-timeout <duration>` | Request timeout | `10s` |
| `-env <path>` | Path to `.env` configuration file | `.env` |

---

## Using `vault-lib` as a Go Library (Without the CLI)

You can import and use the high-level Vault client library (`pkg/vault-lib`) directly in other Go programs without the CLI:

### 1. Add the Dependency

```bash
go get github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib
```

### 2. Quick Integration Example

```go
package main

import (
	"fmt"
	"time"

	"github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib"
)

func main() {
	// 1. Initialize client
	client, err := vault.NewClient(vault.Config{
		Address: "http://127.0.0.1:8200",
		Token:   "your-vault-token",
		Timeout: 10 * time.Second,
	})
	if err != nil {
		panic(err)
	}

	// 2. Write a secret (plain value, JSON object, or file path)
	vault.CreateSecret(client, "myserver/database", `{"host":"10.0.0.1","port":3306}`, 10*time.Second, false)

	// 3. Read back a secret or nested property with dot notation
	var host string
	vault.ReadSecret(client, "myserver/database.host", 10*time.Second, &host, false)
	fmt.Printf("Database host: %s\n", host)

	// 4. List keys under a prefix folder
	keys, _ := vault.ListKeys(client, "myserver", 10*time.Second, false)
	fmt.Printf("Keys under prefix: %v\n", keys)
}
```

For full API reference, unsealing helpers, and nested JSON traversal details, see the [pkg/vault-lib Documentation](pkg/vault-lib/README.md).

---

## Testing & Verification

### 1. Unit & Integration Tests (No External Vault Required)

Run unit tests and generate coverage reports directly from the Makefile:

```bash
# Run unit tests
make test

# Run test coverage and display per-function breakdown (75.4% coverage)
make coverage
```

Detailed coverage metrics, breakdown by function, and test suite details are available in [TESTCOVERAGE.md](TESTCOVERAGE.md).

### 2. End-to-End Verification (Against Running Vault)

Run the end-to-end bash verification script against a real Vault instance:

```bash
./test_vault.sh
```

Tests status, prefix switching, writing, reading, listing, flag overrides, and secret deletion.


---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
