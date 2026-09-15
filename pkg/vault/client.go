package vault

import (
	"context"
	"fmt"
	"log"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
)

// Client is a type alias for the official HashiCorp Vault API client.
type Client = vaultapi.Client

// Config encapsulates the connection parameters for HashiCorp Vault.
type Config struct {
	Address   string
	Token     string
	Namespace string
	Timeout   time.Duration
}

// VaultConfig is an alias for Config.
type VaultConfig = Config

// NewClient initializes and returns an authenticated HashiCorp Vault API client.
func NewClient(cfg Config) (*Client, error) {
	config := vaultapi.DefaultConfig()
	config.Address = cfg.Address
	if cfg.Timeout > 0 {
		config.Timeout = cfg.Timeout
	}

	client, err := vaultapi.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Vault client: %w", err)
	}

	if cfg.Token != "" {
		client.SetToken(cfg.Token)
	}

	if cfg.Namespace != "" {
		client.SetNamespace(cfg.Namespace)
	}

	return client, nil
}

// NewVaultClient is an alias for NewClient.
func NewVaultClient(cfg Config) (*Client, error) {
	return NewClient(cfg)
}

// CheckConnection verifies connectivity, reports server health and seal status,
// optionally unseals if a key is provided, and validates the configured authentication token.
func CheckConnection(client *Client, timeout time.Duration, unsealKey string, verbose bool) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	health, err := client.Sys().HealthWithContext(ctx)
	if err != nil {
		log.Printf("Warning: Failed to query Vault health: %v", err)
		log.Println("Please verify that the Vault server is running and reachable.")
	} else if verbose {
		fmt.Println("\n--- Vault Server Status ---")
		fmt.Printf("Initialized : %t\n", health.Initialized)
		fmt.Printf("Sealed      : %t\n", health.Sealed)
		fmt.Printf("Version     : %s\n", health.Version)
		if health.ClusterName != "" {
			fmt.Printf("Cluster Name: %s\n", health.ClusterName)
		}
	}

	// If Vault is sealed, attempt unsealing if key is available
	if health != nil && health.Sealed {
		if unsealKey != "" {
			if verbose {
				fmt.Println("\nVault is sealed. Attempting to unseal with provided key...")
			}
			status, err := Unseal(client, unsealKey)
			if err != nil {
				log.Printf("Unseal attempt failed: %v", err)
				return
			}
			if status.Sealed {
				return
			}
		} else {
			fmt.Println("Note: Vault is currently sealed. Set VAULT_UNSEAL_KEY in .env or run with 'vault unseal' to unseal.")
			return
		}
	}

	// Verify token authentication details
	tokenSecret, err := client.Auth().Token().LookupSelfWithContext(ctx)
	if err != nil {
		if verbose {
			log.Printf("Warning: Failed to verify token: %v", err)
		}
	} else if verbose && tokenSecret != nil && tokenSecret.Data != nil {
		fmt.Println("\n--- Token Details ---")
		if displayName, ok := tokenSecret.Data["display_name"].(string); ok {
			fmt.Printf("Display Name: %s\n", displayName)
		}
		if policies, ok := tokenSecret.Data["policies"]; ok {
			fmt.Printf("Policies    : %v\n", policies)
		}
		if ttl, ok := tokenSecret.Data["ttl"]; ok {
			fmt.Printf("TTL         : %v\n", ttl)
		}
	}
}

// CheckVaultConnection is an alias for CheckConnection.
func CheckVaultConnection(client *Client, timeout time.Duration, unsealKey string, verbose bool) {
	CheckConnection(client, timeout, unsealKey, verbose)
}

// Unseal attempts to unseal the Vault server using the provided unseal shard key.
func Unseal(client *Client, key string) (*vaultapi.SealStatusResponse, error) {
	if key == "" {
		return nil, fmt.Errorf("unseal key cannot be empty")
	}

	fmt.Println("Sending unseal shard to Vault...")
	status, err := client.Sys().Unseal(key)
	if err != nil {
		return nil, fmt.Errorf("failed to unseal Vault: %w", err)
	}

	if status.Sealed {
		fmt.Printf("Unseal progress: %d/%d threshold reached. Vault is still sealed.\n", status.Progress, status.T)
	} else {
		fmt.Println("✓ Vault successfully UNSEALED!")
	}

	return status, nil
}

// UnsealVault is an alias for Unseal.
func UnsealVault(client *Client, key string) (*vaultapi.SealStatusResponse, error) {
	return Unseal(client, key)
}
