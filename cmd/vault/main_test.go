package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveEnv(t *testing.T) {
	tests := []struct {
		name     string
		flagVal  string
		envMap   map[string]string
		keys     []string
		envSetup map[string]string
		expected string
	}{
		{
			name:     "flag value takes highest priority",
			flagVal:  "from-flag",
			envMap:   map[string]string{"ADDR": "from-map"},
			keys:     []string{"ADDR"},
			envSetup: map[string]string{"ADDR": "from-os"},
			expected: "from-flag",
		},
		{
			name:     "envMap takes precedence over os.Getenv",
			flagVal:  "",
			envMap:   map[string]string{"VAULT_ADDR": "http://map-addr:8200"},
			keys:     []string{"VAULT_ADDR"},
			envSetup: map[string]string{"VAULT_ADDR": "http://os-addr:8200"},
			expected: "http://map-addr:8200",
		},
		{
			name:     "fallback to os.Getenv when key not in envMap",
			flagVal:  "",
			envMap:   map[string]string{},
			keys:     []string{"VAULT_TOKEN"},
			envSetup: map[string]string{"VAULT_TOKEN": "os-token-123"},
			expected: "os-token-123",
		},
		{
			name:     "fallback through multiple keys",
			flagVal:  "",
			envMap:   map[string]string{"NAMESPACE": "backup-ns"},
			keys:     []string{"VAULT_NAMESPACE", "NAMESPACE"},
			envSetup: map[string]string{},
			expected: "backup-ns",
		},
		{
			name:     "fallback through multiple keys to os.Getenv",
			flagVal:  "",
			envMap:   map[string]string{},
			keys:     []string{"VAULT_PREFIX", "PREFIX"},
			envSetup: map[string]string{"PREFIX": "os-fallback-prefix"},
			expected: "os-fallback-prefix",
		},
		{
			name:     "nil envMap safely falls back to os.Getenv",
			flagVal:  "",
			envMap:   nil,
			keys:     []string{"VAULT_ADDR"},
			envSetup: map[string]string{"VAULT_ADDR": "http://safe-nil:8200"},
			expected: "http://safe-nil:8200",
		},
		{
			name:     "returns empty string when nothing is set",
			flagVal:  "",
			envMap:   map[string]string{},
			keys:     []string{"UNSET_KEY"},
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envSetup {
				t.Setenv(k, v)
			}

			result := resolveEnv(tc.flagVal, tc.envMap, tc.keys...)
			if result != tc.expected {
				t.Errorf("resolveEnv(%q, %v, %v) = %q; want %q", tc.flagVal, tc.envMap, tc.keys, result, tc.expected)
			}
		})
	}
}

func TestPrintUsage(t *testing.T) {
	t.Run("with prefix", func(t *testing.T) {
		printUsage("myserver")
	})

	t.Run("without prefix", func(t *testing.T) {
		printUsage("")
	})
}

func TestMainExecutionFlows(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	t.Run("version flag", func(t *testing.T) {
		os.Args = []string{"vault", "-v"}
		main()
	})

	t.Run("version command", func(t *testing.T) {
		os.Args = []string{"vault", "version"}
		main()
	})

	t.Run("help flag", func(t *testing.T) {
		os.Args = []string{"vault", "-help"}
		main()
	})

	t.Run("prefix view and update in temp env", func(t *testing.T) {
		tempDir := t.TempDir()
		envPath := filepath.Join(tempDir, ".env")
		_ = os.WriteFile(envPath, []byte("VAULT_PREFIX=initial_pfx\n"), 0644)

		os.Args = []string{"vault", "-env", envPath, "prefix"}
		main()

		os.Args = []string{"vault", "-env", envPath, "prefix", "new_pfx"}
		main()

		data, err := os.ReadFile(envPath)
		if err != nil || string(data) == "" {
			t.Fatalf("expected updated env file")
		}
	})
}
