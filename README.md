# HashiCorp Vault CLI (`vault`)

A fast, lightweight, and intuitive command-line interface written in Go for managing secrets on HashiCorp Vault (KV v2 secrets engine).

---

## Features

- **Intuitive Subcommands**: First-class command support (`read`, `write`, `delete`, `list`, `prefix`, `unseal`, `status`).
- **Flexible Path Prefixes**: Set a default prefix in `.env` or override per command (`-prefix <folder>`).
- **In-Terminal Prefix Management**: Inspect or change your active prefix dynamically with `vault prefix <name>`.
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

Build the `bin/vault` binary:

```bash
make build
```

Binary will be output to: `bin/vault`.

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

# Or load from a JSON file:
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

### 4. List Keys (`list` / `ls`)
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

### 5. Delete Secrets & Full Prefixes (`delete` / `del` / `rm` / `remove` / `delete-prefix`)
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

### 6. Unseal Vault (`unseal`)
Submit an unseal shard key:

```bash
# Uses VAULT_UNSEAL_KEY from .env
./bin/vault unseal

# Or provide key explicitly
./bin/vault unseal 3c12834f66ad4f77aad5c8bdef4ddbb5f986c4c260f0484ddd3baba986a9262e
```

### 7. Status & Health Check (`status`)
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

## Automated Verification

Run the included end-to-end verification script:

```bash
./test_vault.sh
```

Tests status, prefix switching, writing, reading, listing, flag overrides, and secret deletion.

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
