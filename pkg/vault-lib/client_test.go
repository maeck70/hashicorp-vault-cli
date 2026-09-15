package vault

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	t.Run("successful client initialization", func(t *testing.T) {
		cfg := Config{
			Address:   "http://127.0.0.1:8200",
			Token:     "test-token",
			Namespace: "test-namespace",
			Timeout:   5 * time.Second,
		}

		client, err := NewClient(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client == nil {
			t.Fatal("expected non-nil client")
		}
		if client.Token() != "test-token" {
			t.Errorf("expected token 'test-token', got %q", client.Token())
		}
	})

	t.Run("NewVaultClient alias", func(t *testing.T) {
		cfg := VaultConfig{
			Address: "http://127.0.0.1:8200",
			Token:   "alias-token",
		}
		client, err := NewVaultClient(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client == nil {
			t.Fatal("expected non-nil client")
		}
	})

	t.Run("invalid address returns error", func(t *testing.T) {
		cfg := Config{
			Address: "::invalid-url::",
		}
		client, err := NewClient(cfg)
		if err == nil {
			t.Errorf("expected error for invalid address, got client: %v", client)
		}
	})
}

func TestUnseal(t *testing.T) {
	t.Run("empty key returns error", func(t *testing.T) {
		cfg := Config{Address: "http://127.0.0.1:8200"}
		client, _ := NewClient(cfg)
		_, err := Unseal(client, "")
		if err == nil {
			t.Errorf("expected error when unseal key is empty, got nil")
		}
		_, err = UnsealVault(client, "")
		if err == nil {
			t.Errorf("expected error for alias when unseal key is empty, got nil")
		}
	})

	t.Run("unseal against mock server", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/v1/sys/unseal":
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"sealed":   false,
					"t":        1,
					"n":        1,
					"progress": 0,
					"version":  "1.18.4",
				})
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		client, err := NewClient(Config{Address: server.URL, Token: "root"})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		resp, err := Unseal(client, "test-shard-key")
		if err != nil {
			t.Fatalf("unexpected unseal error: %v", err)
		}
		if resp == nil || resp.Sealed {
			t.Errorf("expected unsealed status, got %v", resp)
		}
	})
}

func TestCheckConnection(t *testing.T) {
	t.Run("unsealed server with valid token", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/v1/sys/health":
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"initialized":   true,
					"sealed":        false,
					"version":       "1.18.4",
					"cluster_name":  "test-cluster",
					"server_time_utc": time.Now().Unix(),
				})
			case "/v1/auth/token/lookup-self":
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{
						"display_name": "token-test",
						"policies":     []string{"root"},
						"ttl":          3600,
					},
				})
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		client, err := NewClient(Config{Address: server.URL, Token: "root"})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		// CheckConnection should run without panic
		CheckConnection(client, 2*time.Second, "", true)
		CheckVaultConnection(client, 2*time.Second, "", false)
	})

	t.Run("sealed server auto-unseals when key provided", func(t *testing.T) {
		unsealCalled := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/v1/sys/health":
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"initialized": true,
					"sealed":      true,
					"version":     "1.18.4",
				})
			case "/v1/sys/unseal":
				unsealCalled = true
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"sealed":   false,
					"progress": 0,
					"t":        1,
				})
			case "/v1/auth/token/lookup-self":
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{
						"display_name": "token-test",
					},
				})
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		client, err := NewClient(Config{Address: server.URL, Token: "root"})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		CheckConnection(client, 2*time.Second, "my-shard-key", true)
		if !unsealCalled {
			t.Errorf("expected auto-unseal to be triggered when unsealKey is provided")
		}
	})
}
