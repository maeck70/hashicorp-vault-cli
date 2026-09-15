package vault

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseJSONValue(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test JSON file
	jsonFile := filepath.Join(tempDir, "config.json")
	fileContent := `{"host":"127.0.0.1","port":5432}`
	if err := os.WriteFile(jsonFile, []byte(fileContent), 0644); err != nil {
		t.Fatalf("failed to create temp json file: %v", err)
	}

	tests := []struct {
		name      string
		input     string
		isJSON    bool
		checkType string
	}{
		{
			name:      "json object",
			input:     `{"name":"test","active":true}`,
			isJSON:    true,
			checkType: "map",
		},
		{
			name:      "json array",
			input:     `["item1", "item2"]`,
			isJSON:    true,
			checkType: "slice",
		},
		{
			name:      "wrapped braces key-value pairs",
			input:     `"user":"admin", "port":8080`,
			isJSON:    true,
			checkType: "map",
		},
		{
			name:      "space separated shell key:value pairs",
			input:     `username:guest password:mypassword`,
			isJSON:    true,
			checkType: "map",
		},
		{
			name:      "file reference with @",
			input:     "@" + jsonFile,
			isJSON:    true,
			checkType: "map",
		},
		{
			name:      "file reference without @",
			input:     jsonFile,
			isJSON:    true,
			checkType: "map",
		},
		{
			name:      "plain text string is not json",
			input:     "my_secret_token_value",
			isJSON:    false,
			checkType: "nil",
		},
		{
			name:      "empty string",
			input:     "",
			isJSON:    false,
			checkType: "nil",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			val, ok := parseJSONValue(tc.input)
			if ok != tc.isJSON {
				t.Fatalf("parseJSONValue(%q) ok = %v; want %v", tc.input, ok, tc.isJSON)
			}
			switch tc.checkType {
			case "map":
				if _, isMap := val.(map[string]any); !isMap {
					t.Errorf("expected map[string]any, got %T", val)
				}
			case "slice":
				if _, isSlice := val.([]any); !isSlice {
					t.Errorf("expected []any, got %T", val)
				}
			case "nil":
				if val != nil {
					t.Errorf("expected nil, got %v", val)
				}
			}
		})
	}
}

func TestGetNestedValue(t *testing.T) {
	data := map[string]any{
		"simple": "hello",
		"nested": map[string]any{
			"deep": map[string]any{
				"target": "found_deep",
			},
		},
		"list": []any{
			map[string]any{"id": "first"},
			map[string]any{"id": "second"},
		},
		"json_str": `{"embedded_key":"embedded_val","arr":[10, 20]}`,
	}

	tests := []struct {
		name     string
		path     []string
		expected any
		found    bool
	}{
		{
			name:     "top-level lookup",
			path:     []string{"simple"},
			expected: "hello",
			found:    true,
		},
		{
			name:     "multi-level nested lookup",
			path:     []string{"nested", "deep", "target"},
			expected: "found_deep",
			found:    true,
		},
		{
			name:     "list index lookup",
			path:     []string{"list", "1", "id"},
			expected: "second",
			found:    true,
		},
		{
			name:     "embedded json string lookup",
			path:     []string{"json_str", "embedded_key"},
			expected: "embedded_val",
			found:    true,
		},
		{
			name:     "embedded json string array lookup",
			path:     []string{"json_str", "arr", "0"},
			expected: float64(10),
			found:    true,
		},
		{
			name:     "non-existent key",
			path:     []string{"nested", "missing"},
			expected: nil,
			found:    false,
		},
		{
			name:     "out of bounds list index",
			path:     []string{"list", "99"},
			expected: nil,
			found:    false,
		},
		{
			name:     "invalid numeric index",
			path:     []string{"list", "not_a_number"},
			expected: nil,
			found:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			val, found := getNestedValue(data, tc.path)
			if found != tc.found {
				t.Fatalf("getNestedValue path %v found = %v; want %v", tc.path, found, tc.found)
			}
			if tc.found && !reflect.DeepEqual(val, tc.expected) {
				t.Errorf("getNestedValue path %v = %v (%T); want %v (%T)", tc.path, val, val, tc.expected, tc.expected)
			}
		})
	}
}

func TestDeleteNestedValue(t *testing.T) {
	t.Run("delete top-level key", func(t *testing.T) {
		m := map[string]any{"a": 1, "b": 2}
		deleted := deleteNestedValue(m, []string{"a"})
		if !deleted {
			t.Errorf("expected delete to return true")
		}
		if _, exists := m["a"]; exists {
			t.Errorf("key 'a' was not deleted")
		}
	})

	t.Run("delete nested key", func(t *testing.T) {
		m := map[string]any{
			"sub": map[string]any{
				"target": "val",
				"keep":   "ok",
			},
		}
		deleted := deleteNestedValue(m, []string{"sub", "target"})
		if !deleted {
			t.Errorf("expected delete to return true")
		}
		subMap := m["sub"].(map[string]any)
		if _, exists := subMap["target"]; exists {
			t.Errorf("nested key 'target' was not deleted")
		}
		if subMap["keep"] != "ok" {
			t.Errorf("sibling key 'keep' was modified")
		}
	})

	t.Run("delete non-existent key returns false", func(t *testing.T) {
		m := map[string]any{"a": 1}
		deleted := deleteNestedValue(m, []string{"missing"})
		if deleted {
			t.Errorf("expected delete to return false for missing key")
		}
	})

	t.Run("delete with empty path returns false", func(t *testing.T) {
		m := map[string]any{"a": 1}
		deleted := deleteNestedValue(m, []string{})
		if deleted {
			t.Errorf("expected delete to return false for empty path")
		}
	})
}

func setupMockVaultKV2Server() (*httptest.Server, map[string]map[string]any) {
	store := make(map[string]map[string]any)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		switch {
		case path == "/v1/sys/health":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"initialized": true, "sealed": false})

		case strings.HasPrefix(path, "/v1/secret/data/"):
			subPath := strings.TrimPrefix(path, "/v1/secret/data/")
			switch r.Method {
			case http.MethodGet:
				data, exists := store[subPath]
				if !exists {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{
						"data":     data,
						"metadata": map[string]any{"version": 1},
					},
				})

			case http.MethodPost, http.MethodPut:
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				if payload, ok := body["data"].(map[string]any); ok {
					store[subPath] = payload
				}
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{"version": 1},
				})

			case http.MethodDelete:
				delete(store, subPath)
				w.WriteHeader(http.StatusNoContent)

			default:
				http.NotFound(w, r)
			}

		case strings.HasPrefix(path, "/v1/secret/metadata/"):
			subPath := strings.TrimPrefix(path, "/v1/secret/metadata/")
			cleanPrefix := strings.Trim(subPath, "/")

			switch r.Method {
			case http.MethodDelete:
				// Delete all keys matching this metadata prefix
				for k := range store {
					if cleanPrefix == "" || k == cleanPrefix || strings.HasPrefix(k, cleanPrefix+"/") {
						delete(store, k)
					}
				}
				w.WriteHeader(http.StatusNoContent)

			default: // List
				var keys []string
				seen := make(map[string]bool)
				for k := range store {
					trimmed := k
					if cleanPrefix != "" {
						if !strings.HasPrefix(k, cleanPrefix+"/") && k != cleanPrefix {
							continue
						}
						trimmed = strings.TrimPrefix(k, cleanPrefix+"/")
					}
					parts := strings.Split(trimmed, "/")
					candidate := parts[0]
					if len(parts) > 1 {
						candidate += "/"
					}
					if !seen[candidate] {
						seen[candidate] = true
						keys = append(keys, candidate)
					}
				}
				if len(keys) == 0 {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{"keys": keys},
				})
			}

		default:
			http.NotFound(w, r)
		}
	}))

	return server, store
}

func TestSecretOperationsAgainstMockVault(t *testing.T) {
	server, _ := setupMockVaultKV2Server()
	defer server.Close()

	client, err := NewClient(Config{
		Address: server.URL,
		Token:   "test-token",
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	timeout := 2 * time.Second

	t.Run("CreateSecret and ReadSecret with plain value", func(t *testing.T) {
		CreateSecret(client, "testapp/api_key", "secret123", timeout, false)

		var val string
		ReadSecret(client, "testapp/api_key", timeout, &val, false)
		if val != "secret123" {
			t.Errorf("ReadSecret() = %q; want %q", val, "secret123")
		}
	})

	t.Run("CreateSecret and ReadSecret with JSON and dot notation", func(t *testing.T) {
		PutSecret(client, "testapp/database", `{"host":"192.168.1.5","port":3306,"user":"root"}`, timeout, false)

		var whole string
		GetSecret(client, "testapp/database", timeout, &whole, false)
		if !strings.Contains(whole, "192.168.1.5") {
			t.Errorf("Read whole secret expected host, got: %s", whole)
		}

		var host string
		ReadSecret(client, "testapp/database.host", timeout, &host, false)
		if host != "192.168.1.5" {
			t.Errorf("ReadSecret dot notation host = %q; want %q", host, "192.168.1.5")
		}
	})

	t.Run("DeleteSecret property via dot notation", func(t *testing.T) {
		err := DeleteSecret(client, "testapp/database.port", timeout, false)
		if err != nil {
			t.Fatalf("DeleteSecret property error: %v", err)
		}

		var port string
		ReadSecret(client, "testapp/database.port", timeout, &port, false)
		if port != "" {
			t.Errorf("expected deleted port to be empty, got: %s", port)
		}

		var host string
		ReadSecret(client, "testapp/database.host", timeout, &host, false)
		if host != "192.168.1.5" {
			t.Errorf("expected sibling field host to remain, got: %s", host)
		}
	})

	t.Run("ListKeys and PrefixExists", func(t *testing.T) {
		CreateSecret(client, "service/web", "val1", timeout, false)
		CreateSecret(client, "service/db", "val2", timeout, false)

		if !PrefixExists(client, "service") {
			t.Errorf("expected PrefixExists('service') to be true")
		}
		if PrefixExists(client, "nonexistent_prefix") {
			t.Errorf("expected PrefixExists('nonexistent_prefix') to be false")
		}

		keys, err := ListKeys(client, "service", timeout, false)
		if err != nil {
			t.Fatalf("ListKeys error: %v", err)
		}
		if len(keys) != 2 {
			t.Errorf("ListKeys('service') expected 2 keys, got %d (%v)", len(keys), keys)
		}
	})

	t.Run("DeletePrefix", func(t *testing.T) {
		CreateSecret(client, "purge_folder/s1", "a", timeout, false)
		CreateSecret(client, "purge_folder/s2", "b", timeout, false)

		count, err := DeletePrefix(client, "purge_folder", timeout, false)
		if err != nil {
			t.Fatalf("DeletePrefix error: %v", err)
		}
		if count != 2 {
			t.Errorf("DeletePrefix expected count 2, got %d", count)
		}

		if PrefixExists(client, "purge_folder") {
			t.Errorf("expected purge_folder to no longer exist")
		}
	})

	t.Run("DeleteSecret key", func(t *testing.T) {
		CreateSecret(client, "to_delete/key1", "val", timeout, false)
		err := RemoveSecret(client, "to_delete/key1", timeout, false)
		if err != nil {
			t.Fatalf("RemoveSecret error: %v", err)
		}

		var out string
		ReadSecret(client, "to_delete/key1", timeout, &out, false)
		if out != "" {
			t.Errorf("expected deleted key output to be empty, got: %s", out)
		}
	})

	t.Run("Compatibility Aliases", func(t *testing.T) {
		CreateVaultSecret(client, "alias_folder/key", "alias_val", timeout, false)

		var out string
		ReadVaultSecret(client, "alias_folder/key", timeout, &out, false)
		if out != "alias_val" {
			t.Errorf("ReadVaultSecret() = %q; want %q", out, "alias_val")
		}

		keys, err := ListVaultKeys(client, "alias_folder", timeout, false)
		if err != nil || len(keys) == 0 {
			t.Errorf("ListVaultKeys failed: %v", err)
		}

		if err := DeleteVaultSecret(client, "alias_folder/key", timeout, false); err != nil {
			t.Errorf("DeleteVaultSecret failed: %v", err)
		}

		CreateVaultSecret(client, "alias_folder/key2", "val2", timeout, false)
		if _, err := DeleteVaultPrefix(client, "alias_folder", timeout, false); err != nil {
			t.Errorf("DeleteVaultPrefix failed: %v", err)
		}
	})
}
