# Package `vault`

Package `vault` (`pkg/vault`) provides a high-level, idiomatic Go client and helper library for interacting with HashiCorp Vault. It wraps the official [HashiCorp Vault Go API](https://github.com/hashicorp/vault/api) with enhanced support for KV version 2 (KV-v2) secret engines, automatic path normalization, prefix scoping, nested JSON querying via dot notation, and server lifecycle operations (health checks and unsealing).

---

## Features

- **Simplified Client Management**: Initialize authenticated clients with standard configuration, timeout handling, and optional namespaces.
- **Server Health & Auto-Unseal**: Health checks, connection verification, token introspection, and unsealing with master key shards.
- **KV v2 Native Support**: Automatically routes paths through `<mount>/data/<key>` and `<mount>/metadata/<key>` endpoints without manual URL manipulation.
- **Prefix Scoping**: Automatically resolves hierarchical prefix paths (e.g. `myserver/database`), supporting defaults from environment variables (`VAULT_PREFIX`).
- **Rich JSON & Dot Notation**:
  - Read and write complex nested JSON objects or arrays.
  - Query nested fields directly via dot notation (e.g., `mysql.primary.host`).
  - Delete individual JSON keys in-place without destroying the parent secret.
  - Automatic parsing of JSON files (`@path/to/file.json`) and raw strings.
- **Recursive Directory Operations**: Discover, list, and recursively purge keys within a prefix.

---

## File Structure

| File | Purpose |
|------|---------|
| [client.go](client.go) | Client initialization (`NewClient`), server health checks (`CheckConnection`), token validation, and unsealing (`Unseal`). |
| [paths.go](paths.go) | Path normalization for KV-v2 (`NormalizePath`), prefix resolution (`ResolveKey`), and `.env` prefix persistence (`SetEnvPrefix`). |
| [secrets.go](secrets.go) | CRUD operations: `CreateSecret`, `ReadSecret`, `ListKeys`, `DeleteSecret`, `DeletePrefix`, nested JSON traversal, and dot notation resolution. |

---

## Installation & Import

Import the package into your Go project:

```go
import "github.com/maeck70/hashicorp-vault-cli/pkg/vault"
```

---

## API Reference & Usage

### 1. Client Initialization & Health Check

```go
package main

import (
    "time"
    "github.com/maeck70/hashicorp-vault-cli/pkg/vault"
)

func main() {
    cfg := vault.Config{
        Address:   "http://127.0.0.1:8200",
        Token:     "your-vault-token",
        Namespace: "", // optional
        Timeout:   10 * time.Second,
    }

    client, err := vault.NewClient(cfg)
    if err != nil {
        panic(err)
    }

    // Verify connectivity, inspect token policies, and optionally unseal:
    vault.CheckConnection(client, 5*time.Second, "unseal-key-shard-if-needed", true)
}
```

### 2. Path & Prefix Resolution

```go
// Combines prefix with key, avoiding duplicated /data/ or prefixes:
resolved := vault.ResolveKey("myserver", "database/password")
// => "myserver/database/password"

// Normalizes KV v2 secret paths according to VAULT_MOUNT (default: "secret"):
normalized := vault.NormalizePath("myserver/database/password")
// => "secret/data/myserver/database/password"
```

### 3. Writing Secrets (`CreateSecret`)

`CreateSecret` writes secrets to Vault KV-v2. If the value is a valid JSON string or references a JSON file (`@filename.json`), it is stored as structured JSON. If the value is omitted, a random Star Trek character name is generated automatically:

```go
timeout := 10 * time.Second
verbose := false

// Plain string value:
vault.CreateSecret(client, "myserver/api_key", "secret123", timeout, verbose)

// Structured JSON:
vault.CreateSecret(client, "myserver/mysql", `{"host":"10.0.0.1","port":3306,"user":"admin"}`, timeout, verbose)

// From a JSON file:
vault.CreateSecret(client, "myserver/config", "@config.json", timeout, verbose)
```

### 4. Reading Secrets with Dot Notation (`ReadSecret`)

`ReadSecret` reads secrets, pretty-prints JSON structures, and supports dot notation to extract nested values:

```go
var out string

// Read the entire secret object:
vault.ReadSecret(client, "myserver/mysql", timeout, &out, verbose)
fmt.Println(out)

// Read a specific nested key using dot notation:
vault.ReadSecret(client, "myserver/mysql.host", timeout, &out, verbose)
// 'out' will contain "10.0.0.1"
```

### 5. Listing Keys (`ListKeys`)

`ListKeys` recursively scans the KV-v2 metadata engine under a given prefix and returns all discovered secret paths:

```go
keys, err := vault.ListKeys(client, "myserver", timeout, verbose)
if err != nil {
    panic(err)
}
for _, key := range keys {
    fmt.Println("-", key)
}
```

### 6. Deleting Secrets & Prefixes

```go
// Delete an individual JSON field from a secret:
err := vault.DeleteSecret(client, "myserver/mysql.host", timeout, verbose)

// Delete an entire secret key:
err = vault.DeleteSecret(client, "myserver/api_key", timeout, verbose)

// Recursively delete all secrets under a prefix:
deletedCount, err := vault.DeletePrefix(client, "myserver", timeout, verbose)
fmt.Printf("Deleted %d secret(s)\n", deletedCount)
```

### 7. Unsealing Vault

```go
sealStatus, err := vault.Unseal(client, "3c12834f66ad4f77aad5c8bdef4ddbb5f986c4c260f0484ddd3baba986a9262e")
if err != nil {
    log.Fatalf("Unseal failed: %v", err)
}
fmt.Printf("Sealed status: %t\n", sealStatus.Sealed)
```

---

## Environment Variables

The package interacts with standard environment variables when specific options are omitted:

| Variable | Description | Default |
|----------|-------------|---------|
| `VAULT_ADDR` | Vault server endpoint URL | `http://127.0.0.1:8200` |
| `VAULT_TOKEN` | Vault authentication token | — |
| `VAULT_PREFIX` / `PREFIX` | Default prefix folder for secrets | `""` |
| `VAULT_MOUNT` | KV engine mount name | `secret` |
| `VAULT_NAMESPACE` | Vault namespace (Enterprise) | `""` |
| `VAULT_UNSEAL_KEY` | Key shard used for unsealing | `""` |

---

## License

This package is part of the `hashicorp-vault-cli` project and is licensed under the [MIT License](../../LICENSE).
