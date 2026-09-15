package vault

import (
	"fmt"
	"os"
	"strings"
)

// ResolveKey cleanly joins prefix and key. If prefix is empty, it checks VAULT_PREFIX or PREFIX env vars.
func ResolveKey(prefix, key string) string {
	cleanKey := strings.Trim(key, "/")

	if prefix == "" {
		prefix = os.Getenv("VAULT_PREFIX")
		if prefix == "" {
			prefix = os.Getenv("PREFIX")
		}
	}
	cleanPrefix := strings.Trim(prefix, "/")

	if cleanKey == "" {
		return cleanPrefix
	}
	if cleanPrefix == "" {
		return cleanKey
	}

	// If key is an absolute path with /data/ or already prefixed, don't duplicate
	if strings.Contains(cleanKey, "/data/") {
		return cleanKey
	}
	if cleanKey == cleanPrefix || strings.HasPrefix(cleanKey, cleanPrefix+"/") {
		return cleanKey
	}

	return fmt.Sprintf("%s/%s", cleanPrefix, cleanKey)
}

// ResolveVaultKey is an alias for ResolveKey for backwards compatibility.
func ResolveVaultKey(prefix, key string) string {
	return ResolveKey(prefix, key)
}

// NormalizePath ensures the path targets the KV v2 endpoint (<mount>/data/<key>).
func NormalizePath(path string) string {
	clean := strings.Trim(path, "/")
	if clean == "" {
		return ""
	}

	// If already in KV v2 format with /data/ (e.g. secret/data/captain)
	if strings.Contains(clean, "/data/") {
		return clean
	}

	mount := os.Getenv("VAULT_MOUNT")
	if mount == "" {
		mount = "secret"
	}
	mount = strings.Trim(mount, "/")

	// If path starts with <mount>/, e.g. secret/captain -> secret/data/captain
	if strings.HasPrefix(clean, mount+"/") {
		subPath := strings.TrimPrefix(clean, mount+"/")
		return fmt.Sprintf("%s/data/%s", mount, subPath)
	}

	// Otherwise captain -> secret/data/captain
	return fmt.Sprintf("%s/data/%s", mount, clean)
}

// NormalizeVaultPath is an alias for NormalizePath for backwards compatibility.
func NormalizeVaultPath(path string) string {
	return NormalizePath(path)
}

// SetEnvPrefix updates or appends VAULT_PREFIX in the target .env file.
func SetEnvPrefix(envPath, newPrefix string) error {
	if envPath == "" {
		envPath = ".env"
	}
	cleanPrefix := strings.Trim(newPrefix, "/")

	data, err := os.ReadFile(envPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read %s: %w", envPath, err)
	}

	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "VAULT_PREFIX=") {
			lines[i] = fmt.Sprintf("VAULT_PREFIX=%s", cleanPrefix)
			found = true
			break
		}
	}
	if !found {
		lines = append(lines, fmt.Sprintf("VAULT_PREFIX=%s", cleanPrefix))
	}

	if err := os.WriteFile(envPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", envPath, err)
	}

	return nil
}
