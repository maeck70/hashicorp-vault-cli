package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveKey(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		key      string
		envKey   string
		envVal   string
		expected string
	}{
		{
			name:     "both prefix and key provided",
			prefix:   "myserver",
			key:      "rabbitmq/user",
			expected: "myserver/rabbitmq/user",
		},
		{
			name:     "clean leading and trailing slashes",
			prefix:   "/myserver/",
			key:      "/rabbitmq/user/",
			expected: "myserver/rabbitmq/user",
		},
		{
			name:     "empty prefix with non-empty key",
			prefix:   "",
			key:      "rabbitmq/user",
			expected: "rabbitmq/user",
		},
		{
			name:     "empty key with non-empty prefix",
			prefix:   "myserver",
			key:      "",
			expected: "myserver",
		},
		{
			name:     "key already has /data/ path",
			prefix:   "myserver",
			key:      "secret/data/database/password",
			expected: "secret/data/database/password",
		},
		{
			name:     "key is identical to prefix",
			prefix:   "myserver",
			key:      "myserver",
			expected: "myserver",
		},
		{
			name:     "key already has prefix as subpath",
			prefix:   "myserver",
			key:      "myserver/database/password",
			expected: "myserver/database/password",
		},
		{
			name:     "prefix from VAULT_PREFIX environment variable",
			prefix:   "",
			key:      "database/port",
			envKey:   "VAULT_PREFIX",
			envVal:   "env_prefix",
			expected: "env_prefix/database/port",
		},
		{
			name:     "prefix from PREFIX environment variable",
			prefix:   "",
			key:      "database/port",
			envKey:   "PREFIX",
			envVal:   "legacy_prefix",
			expected: "legacy_prefix/database/port",
		},
		{
			name:     "both prefix and key empty",
			prefix:   "",
			key:      "",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envKey != "" {
				t.Setenv(tc.envKey, tc.envVal)
			} else {
				t.Setenv("VAULT_PREFIX", "")
				t.Setenv("PREFIX", "")
			}

			result := ResolveKey(tc.prefix, tc.key)
			if result != tc.expected {
				t.Errorf("ResolveKey(%q, %q) = %q; want %q", tc.prefix, tc.key, result, tc.expected)
			}

			// Also verify backwards compatibility alias
			aliasResult := ResolveVaultKey(tc.prefix, tc.key)
			if aliasResult != tc.expected {
				t.Errorf("ResolveVaultKey(%q, %q) = %q; want %q", tc.prefix, tc.key, aliasResult, tc.expected)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		vaultMount string
		expected   string
	}{
		{
			name:     "empty path returns empty",
			path:     "",
			expected: "",
		},
		{
			name:     "simple key uses default mount 'secret'",
			path:     "captain",
			expected: "secret/data/captain",
		},
		{
			name:     "hierarchical key uses default mount",
			path:     "myserver/database/password",
			expected: "secret/data/myserver/database/password",
		},
		{
			name:     "slashes are cleaned",
			path:     "/myserver/database/password/",
			expected: "secret/data/myserver/database/password",
		},
		{
			name:     "path already starting with mount name",
			path:     "secret/captain",
			expected: "secret/data/captain",
		},
		{
			name:     "path already in KV v2 format with /data/",
			path:     "secret/data/captain",
			expected: "secret/data/captain",
		},
		{
			name:       "custom VAULT_MOUNT",
			path:       "app/db_pass",
			vaultMount: "kv-engine",
			expected:   "kv-engine/data/app/db_pass",
		},
		{
			name:       "custom VAULT_MOUNT when path starts with mount name",
			path:       "kv-engine/app/db_pass",
			vaultMount: "kv-engine",
			expected:   "kv-engine/data/app/db_pass",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.vaultMount != "" {
				t.Setenv("VAULT_MOUNT", tc.vaultMount)
			} else {
				t.Setenv("VAULT_MOUNT", "")
			}

			result := NormalizePath(tc.path)
			if result != tc.expected {
				t.Errorf("NormalizePath(%q) = %q; want %q", tc.path, result, tc.expected)
			}

			aliasResult := NormalizeVaultPath(tc.path)
			if aliasResult != tc.expected {
				t.Errorf("NormalizeVaultPath(%q) = %q; want %q", tc.path, aliasResult, tc.expected)
			}
		})
	}
}

func TestSetEnvPrefix(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("creates new file if not exists", func(t *testing.T) {
		envPath := filepath.Join(tempDir, "new.env")
		err := SetEnvPrefix(envPath, "my_prefix")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := os.ReadFile(envPath)
		if err != nil {
			t.Fatalf("failed to read created env file: %v", err)
		}

		if !strings.Contains(string(content), "VAULT_PREFIX=my_prefix") {
			t.Errorf("expected file to contain VAULT_PREFIX=my_prefix, got: %s", string(content))
		}
	})

	t.Run("updates existing VAULT_PREFIX in place", func(t *testing.T) {
		envPath := filepath.Join(tempDir, "existing.env")
		initial := "# Vault Configuration\nVAULT_ADDR=http://127.0.0.1:8200\nVAULT_PREFIX=old_prefix\nVAULT_TOKEN=tok123\n"
		if err := os.WriteFile(envPath, []byte(initial), 0644); err != nil {
			t.Fatalf("failed to write initial file: %v", err)
		}

		err := SetEnvPrefix(envPath, "updated_prefix")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := os.ReadFile(envPath)
		if err != nil {
			t.Fatalf("failed to read env file: %v", err)
		}

		lines := strings.Split(string(content), "\n")
		found := false
		for _, line := range lines {
			switch line {
			case "VAULT_PREFIX=updated_prefix":
				found = true
			case "VAULT_PREFIX=old_prefix":
				t.Errorf("old prefix was not replaced")
			}
		}
		if !found {
			t.Errorf("VAULT_PREFIX=updated_prefix not found in file: %s", string(content))
		}
	})

	t.Run("appends when VAULT_PREFIX not present in existing file", func(t *testing.T) {
		envPath := filepath.Join(tempDir, "no_prefix.env")
		initial := "VAULT_ADDR=http://127.0.0.1:8200\nVAULT_TOKEN=tok123\n"
		if err := os.WriteFile(envPath, []byte(initial), 0644); err != nil {
			t.Fatalf("failed to write initial file: %v", err)
		}

		err := SetEnvPrefix(envPath, "appended_prefix")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := os.ReadFile(envPath)
		if err != nil {
			t.Fatalf("failed to read env file: %v", err)
		}

		if !strings.Contains(string(content), "VAULT_PREFIX=appended_prefix") {
			t.Errorf("expected appended prefix, got: %s", string(content))
		}
	})

	t.Run("error on invalid path like a directory", func(t *testing.T) {
		err := SetEnvPrefix(tempDir, "some_prefix")
		if err == nil {
			t.Errorf("expected error writing to directory path, got nil")
		}
	})
}
