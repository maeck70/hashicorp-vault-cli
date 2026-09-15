package vault

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// parseJSONValue checks if the provided string is a JSON file reference or a valid JSON literal.
func parseJSONValue(value string) (interface{}, bool) {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "@") {
		filePath := strings.TrimPrefix(trimmed, "@")
		if data, err := os.ReadFile(filePath); err == nil {
			trimmed = strings.TrimSpace(string(data))
		}
	} else if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		if fi, err := os.Stat(trimmed); err == nil && !fi.IsDir() {
			if data, err := os.ReadFile(trimmed); err == nil {
				trimmed = strings.TrimSpace(string(data))
			}
		}
	}

	// 1. Direct JSON object
	if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(trimmed), &obj); err == nil {
			return obj, true
		}
	}

	// 2. Direct JSON array
	if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
		var arr []interface{}
		if err := json.Unmarshal([]byte(trimmed), &arr); err == nil {
			return arr, true
		}
	}

	// 3. Wrapped in braces if braces were stripped
	if strings.Contains(trimmed, ":") {
		wrapped := "{" + trimmed + "}"
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(wrapped), &obj); err == nil {
			return obj, true
		}
	}

	// 4. Space or comma-separated key:value pairs from shell expansion (e.g. username:guest password:muysecreto)
	if strings.Contains(trimmed, ":") {
		fields := strings.Fields(trimmed)
		if len(fields) > 0 {
			obj := make(map[string]interface{})
			allPairs := true
			for _, field := range fields {
				field = strings.Trim(field, "{}, ")
				if field == "" {
					continue
				}
				idx := strings.Index(field, ":")
				if idx <= 0 {
					allPairs = false
					break
				}
				k := strings.Trim(field[:idx], `"' `)
				v := strings.Trim(field[idx+1:], `"' `)
				if k == "" {
					allPairs = false
					break
				}
				var parsedV interface{}
				if err := json.Unmarshal([]byte(v), &parsedV); err == nil {
					obj[k] = parsedV
				} else {
					obj[k] = v
				}
			}
			if allPairs && len(obj) > 0 {
				return obj, true
			}
		}
	}

	return nil, false
}

// getNestedValue extracts a nested property using dot-separated path segments.
func getNestedValue(data interface{}, path []string) (interface{}, bool) {
	current := data
	for _, segment := range path {
		switch node := current.(type) {
		case map[string]interface{}:
			val, exists := node[segment]
			if !exists {
				return nil, false
			}
			current = val
		case []interface{}:
			idx, err := strconv.Atoi(segment)
			if err != nil || idx < 0 || idx >= len(node) {
				return nil, false
			}
			current = node[idx]
		case string:
			var parsed interface{}
			if err := json.Unmarshal([]byte(node), &parsed); err == nil {
				switch parsedNode := parsed.(type) {
				case map[string]interface{}:
					val, exists := parsedNode[segment]
					if !exists {
						return nil, false
					}
					current = val
				case []interface{}:
					idx, err := strconv.Atoi(segment)
					if err != nil || idx < 0 || idx >= len(parsedNode) {
						return nil, false
					}
					current = parsedNode[idx]
				default:
					return nil, false
				}
			} else {
				return nil, false
			}
		default:
			return nil, false
		}
	}
	return current, true
}

// deleteNestedValue removes a nested property from a map structure.
func deleteNestedValue(data map[string]interface{}, path []string) bool {
	if len(path) == 0 {
		return false
	}
	if len(path) == 1 {
		if _, ok := data[path[0]]; ok {
			delete(data, path[0])
			return true
		}
		return false
	}
	if nextNode, ok := data[path[0]].(map[string]interface{}); ok {
		return deleteNestedValue(nextNode, path[1:])
	}
	return false
}

// fetchSecretData reads a secret from Vault and returns the unwrapped data map.
func fetchSecretData(ctx context.Context, client *Client, path string) (map[string]interface{}, bool, bool, error) {
	normPath := NormalizePath(path)
	secret, err := client.Logical().ReadWithContext(ctx, normPath)
	if err != nil {
		return nil, false, false, err
	}
	if secret == nil || secret.Data == nil {
		return nil, false, false, nil
	}

	if kv2Data, ok := secret.Data["data"].(map[string]interface{}); ok {
		return kv2Data, false, true, nil
	} else if secret.Data["data"] == nil && secret.Data["metadata"] != nil {
		return nil, true, true, nil
	}
	return secret.Data, false, true, nil
}

// CreateSecret creates or updates a secret in Vault KV v2.
// If value is a JSON literal or JSON file, components are stored for field-level access.
func CreateSecret(client *Client, secretPath string, value string, timeout time.Duration, verbose bool) {
	normPath := NormalizePath(secretPath)
	if verbose {
		fmt.Printf("\n--- Writing Vault Secret to %s ---\n", normPath)
		fmt.Printf("Value: %s\n", value)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	health, err := client.Sys().HealthWithContext(ctx)
	if err == nil && health != nil && health.Sealed {
		fmt.Println("Warning: Vault is sealed. Cannot write secret until Vault is unsealed.")
		return
	}

	var payloadData map[string]interface{}
	jsonVal, isJSON := parseJSONValue(value)
	displayVal := value

	if isJSON {
		if objMap, ok := jsonVal.(map[string]interface{}); ok {
			payloadData = objMap
			jsonBytes, _ := json.Marshal(objMap)
			displayVal = string(jsonBytes)
		} else {
			payloadData = map[string]interface{}{
				"value":      jsonVal,
				"updated_at": time.Now().UTC().Format(time.RFC3339),
			}
		}
	} else {
		payloadData = map[string]interface{}{
			"value":      value,
			"updated_at": time.Now().UTC().Format(time.RFC3339),
		}
	}

	payload := map[string]interface{}{
		"data": payloadData,
	}

	_, err = client.Logical().WriteWithContext(ctx, normPath, payload)
	if err != nil {
		log.Printf("Error writing secret to %s: %v", normPath, err)
		return
	}

	fmt.Printf("✓ Secret successfully written to %s (value=%s)\n", normPath, displayVal)
}

// CreateVaultSecret is an alias for CreateSecret.
func CreateVaultSecret(client *Client, secretPath string, value string, timeout time.Duration, verbose bool) {
	CreateSecret(client, secretPath, value, timeout, verbose)
}

// ReadSecret retrieves a secret or an individual JSON component from Vault KV v2.
func ReadSecret(client *Client, secretPath string, timeout time.Duration, out *string, verbose bool) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	health, err := client.Sys().HealthWithContext(ctx)
	if err == nil && health != nil && health.Sealed {
		fmt.Println("Warning: Vault is sealed. Cannot read secret until Vault is unsealed.")
		return
	}

	// 1. Try reading secretPath directly
	dataMap, isDeleted, exists, err := fetchSecretData(ctx, client, secretPath)
	var fieldSelector []string
	actualSecretPath := secretPath

	if isDeleted {
		fmt.Printf("Notice: Secret at %s has been deleted or has no active data.\n", NormalizePath(secretPath))
		return
	}

	// 2. If not found directly and secretPath contains '.', try splitting to find the secret and JSON selector
	if !exists && strings.Contains(secretPath, ".") {
		parts := strings.Split(secretPath, ".")
		for i := len(parts) - 1; i >= 1; i-- {
			candidatePath := strings.Join(parts[:i], ".")
			cData, cDeleted, cExists, cErr := fetchSecretData(ctx, client, candidatePath)
			if cErr == nil && cDeleted {
				fmt.Printf("Notice: Secret at %s has been deleted or has no active data.\n", NormalizePath(candidatePath))
				return
			}
			if cErr == nil && cExists {
				dataMap = cData
				exists = true
				actualSecretPath = candidatePath
				fieldSelector = parts[i:]
				break
			}
		}
	}

	if !exists || dataMap == nil {
		log.Printf("No secret found at %s", NormalizePath(secretPath))
		return
	}

	if verbose {
		fmt.Printf("\n--- Reading Secret from %s ---\n", NormalizePath(actualSecretPath))
		if len(fieldSelector) > 0 {
			fmt.Printf("Selector Path: %s\n", strings.Join(fieldSelector, "."))
		}
	}

	var extracted string
	if len(fieldSelector) > 0 {
		// Extract specific component
		targetVal, found := getNestedValue(dataMap, fieldSelector)
		if !found {
			// Check if dataMap has a "value" field containing a JSON string
			if v, ok := dataMap["value"]; ok {
				targetVal, found = getNestedValue(v, fieldSelector)
			}
		}
		if !found {
			log.Printf("Field %q not found in secret %s", strings.Join(fieldSelector, "."), NormalizePath(actualSecretPath))
			return
		}

		switch v := targetVal.(type) {
		case string:
			extracted = v
		case map[string]interface{}, []interface{}:
			jsonBytes, err := json.Marshal(v)
			if err == nil {
				extracted = string(jsonBytes)
			} else {
				extracted = fmt.Sprintf("%v", v)
			}
		default:
			extracted = fmt.Sprintf("%v", v)
		}
	} else {
		// Whole secret retrieval
		// If stored as {"value": "..."} and value is not JSON, return value
		if val, ok := dataMap["value"]; ok && len(dataMap) <= 2 {
			extracted = fmt.Sprintf("%v", val)
		} else {
			// Return as JSON block
			jsonBytes, err := json.Marshal(dataMap)
			if err == nil {
				extracted = string(jsonBytes)
			} else {
				extracted = fmt.Sprintf("%v", dataMap)
			}
		}
	}

	if out != nil {
		*out = extracted
	}
}

// ReadVaultSecret is an alias for ReadSecret.
func ReadVaultSecret(client *Client, secretPath string, timeout time.Duration, out *string, verbose bool) {
	ReadSecret(client, secretPath, timeout, out, verbose)
}

// PutSecret is an alias for CreateSecret.
func PutSecret(client *Client, secretPath string, value string, timeout time.Duration, verbose bool) {
	CreateSecret(client, secretPath, value, timeout, verbose)
}

// GetSecret is an alias for ReadSecret.
func GetSecret(client *Client, secretPath string, timeout time.Duration, out *string, verbose bool) {
	ReadSecret(client, secretPath, timeout, out, verbose)
}

// PrefixGroup represents a collection of keys grouped under a prefix.
type PrefixGroup struct {
	Prefix string
	Keys   []string
}

// collectPrefixGroups recursively traverses Vault metadata starting from relPath and groups keys by prefix.
func collectPrefixGroups(ctx context.Context, client *Client, mount, relPath string) ([]PrefixGroup, error) {
	vaultPath := fmt.Sprintf("%s/metadata", mount)
	if relPath != "" {
		vaultPath = fmt.Sprintf("%s/metadata/%s", mount, relPath)
	}

	secret, err := client.Logical().ListWithContext(ctx, vaultPath)
	if err != nil {
		return nil, err
	}
	if secret == nil || secret.Data == nil {
		return nil, nil
	}

	rawKeys, ok := secret.Data["keys"]
	if !ok {
		return nil, nil
	}

	var directKeys []string
	var subFolders []string

	if keyList, ok := rawKeys.([]interface{}); ok {
		for _, k := range keyList {
			if s, ok := k.(string); ok {
				if strings.HasSuffix(s, "/") {
					subFolders = append(subFolders, strings.TrimSuffix(s, "/"))
				} else {
					directKeys = append(directKeys, s)
				}
			}
		}
	}

	var groups []PrefixGroup
	if len(directKeys) > 0 {
		displayPrefix := relPath
		if displayPrefix == "" {
			displayPrefix = "(root)"
		}
		groups = append(groups, PrefixGroup{
			Prefix: displayPrefix,
			Keys:   directKeys,
		})
	}

	for _, sub := range subFolders {
		childPath := sub
		if relPath != "" {
			childPath = fmt.Sprintf("%s/%s", relPath, sub)
		}
		childGroups, err := collectPrefixGroups(ctx, client, mount, childPath)
		if err == nil && len(childGroups) > 0 {
			groups = append(groups, childGroups...)
		}
	}

	return groups, nil
}

// ListKeys retrieves and displays the secret keys.
// If prefix is provided, it reports the prefix and lists keys under it.
// If no prefix is provided (empty, "/", or "all"), it recursively groups keys by prefix and reports each prefix.
func ListKeys(client *Client, prefix string, timeout time.Duration, verbose bool) ([]string, error) {
	mount := os.Getenv("VAULT_MOUNT")
	if mount == "" {
		mount = "secret"
	}
	mount = strings.Trim(mount, "/")

	if prefix == "/" || prefix == "*" || prefix == "all" {
		prefix = ""
	}

	cleanPrefix := strings.Trim(prefix, "/")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Check if server is sealed before listing
	health, err := client.Sys().HealthWithContext(ctx)
	if err == nil && health != nil && health.Sealed {
		fmt.Println("Warning: Vault is sealed. Cannot list keys until Vault is unsealed.")
		return nil, fmt.Errorf("vault is sealed")
	}

	// Case 1: No prefix provided -> group keys by prefix and report each prefix
	if cleanPrefix == "" {
		if verbose {
			fmt.Printf("\n--- Listing All Keys Grouped by Prefix (Mount: %s) ---\n", mount)
		}
		groups, err := collectPrefixGroups(ctx, client, mount, "")
		if err != nil {
			log.Printf("Error listing keys at %s/metadata: %v", mount, err)
			return nil, err
		}

		if len(groups) == 0 {
			fmt.Println("No keys found.")
			return nil, nil
		}

		if verbose {
			total := 0
			for _, grp := range groups {
				total += len(grp.Keys)
			}
			fmt.Printf("Found %d key(s) across %d prefix group(s):\n\n", total, len(groups))
		}

		var allKeys []string
		for i, grp := range groups {
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("Prefix: %s\n", grp.Prefix)
			for _, k := range grp.Keys {
				fmt.Printf("  • %s\n", k)
				if grp.Prefix == "(root)" {
					allKeys = append(allKeys, k)
				} else {
					allKeys = append(allKeys, fmt.Sprintf("%s/%s", grp.Prefix, k))
				}
			}
		}
		return allKeys, nil
	}

	// Case 2: Specific prefix provided -> report prefix and list keys under it
	var vaultPath string
	if strings.Contains(cleanPrefix, "/data/") {
		vaultPath = strings.Replace(cleanPrefix, "/data/", "/metadata/", 1)
	} else if strings.HasPrefix(cleanPrefix, mount+"/metadata") {
		vaultPath = cleanPrefix
	} else if strings.HasPrefix(cleanPrefix, mount+"/") {
		sub := strings.TrimPrefix(cleanPrefix, mount+"/")
		vaultPath = fmt.Sprintf("%s/metadata/%s", mount, sub)
	} else {
		vaultPath = fmt.Sprintf("%s/metadata/%s", mount, cleanPrefix)
	}

	if verbose {
		fmt.Printf("\n--- Listing Keys from %s ---\n", vaultPath)
	}

	fmt.Printf("Prefix: %s\n", cleanPrefix)
	secret, err := client.Logical().ListWithContext(ctx, vaultPath)
	if err != nil {
		log.Printf("Error listing keys at %s: %v", vaultPath, err)
		return nil, err
	}

	var keys []string
	if secret != nil && secret.Data != nil {
		if rawKeys, ok := secret.Data["keys"]; ok {
			if keyList, ok := rawKeys.([]interface{}); ok {
				for _, k := range keyList {
					if s, ok := k.(string); ok {
						keys = append(keys, s)
					}
				}
			}
		}
	}

	if len(keys) == 0 {
		// Check if cleanPrefix is an existing secret containing JSON components
		sData, isDel, exists, _ := fetchSecretData(ctx, client, cleanPrefix)
		if exists && !isDel && sData != nil {
			for k := range sData {
				if k != "updated_at" {
					keys = append(keys, k)
				}
			}
			sort.Strings(keys)
		}
	}

	if len(keys) == 0 {
		fmt.Println("No keys found.")
		return nil, nil
	}

	if verbose {
		fmt.Printf("Found %d key(s):\n", len(keys))
	}
	for _, k := range keys {
		fmt.Printf("  • %s\n", k)
	}

	return keys, nil
}

// ListVaultKeys is an alias for ListKeys.
func ListVaultKeys(client *Client, prefix string, timeout time.Duration, verbose bool) ([]string, error) {
	return ListKeys(client, prefix, timeout, verbose)
}

// DeletePrefix recursively deletes all secrets and metadata under the given prefix.
func DeletePrefix(client *Client, prefix string, timeout time.Duration, verbose bool) (int, error) {
	mount := os.Getenv("VAULT_MOUNT")
	if mount == "" {
		mount = "secret"
	}
	mount = strings.Trim(mount, "/")

	cleanPrefix := strings.Trim(prefix, "/")
	displayPrefix := cleanPrefix
	if displayPrefix == "" {
		displayPrefix = "(root)"
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	health, err := client.Sys().HealthWithContext(ctx)
	if err == nil && health != nil && health.Sealed {
		fmt.Println("Warning: Vault is sealed. Cannot delete prefix until Vault is unsealed.")
		return 0, fmt.Errorf("vault is sealed")
	}

	if verbose {
		fmt.Printf("\n--- Deleting Full Prefix %q (Mount: %s) ---\n", displayPrefix, mount)
	}

	groups, err := collectPrefixGroups(ctx, client, mount, cleanPrefix)
	if err != nil {
		return 0, fmt.Errorf("failed to scan prefix %q: %w", displayPrefix, err)
	}

	var allSecretPaths []string
	for _, grp := range groups {
		for _, k := range grp.Keys {
			var fullKey string
			if grp.Prefix == "(root)" || grp.Prefix == "" {
				fullKey = k
			} else {
				fullKey = fmt.Sprintf("%s/%s", grp.Prefix, k)
			}
			allSecretPaths = append(allSecretPaths, fullKey)
		}
	}

	// Also check if cleanPrefix itself is an individual secret (e.g. leaf)
	if len(allSecretPaths) == 0 && cleanPrefix != "" {
		sData, _, exists, _ := fetchSecretData(ctx, client, cleanPrefix)
		if exists && sData != nil {
			allSecretPaths = append(allSecretPaths, cleanPrefix)
		}
	}

	if len(allSecretPaths) == 0 {
		fmt.Printf("No secrets found under prefix %q to delete.\n", displayPrefix)
		return 0, nil
	}

	deletedCount := 0
	for _, secPath := range allSecretPaths {
		dataPath := fmt.Sprintf("%s/data/%s", mount, strings.Trim(secPath, "/"))
		_, _ = client.Logical().DeleteWithContext(ctx, dataPath)

		metaPath := fmt.Sprintf("%s/metadata/%s", mount, strings.Trim(secPath, "/"))
		_, _ = client.Logical().DeleteWithContext(ctx, metaPath)

		fmt.Printf("  • Deleted %s\n", secPath)
		deletedCount++
	}

	// Also delete intermediate and top folder metadata endpoints
	for _, grp := range groups {
		if grp.Prefix != "(root)" && grp.Prefix != "" {
			folderMetaPath := fmt.Sprintf("%s/metadata/%s", mount, strings.Trim(grp.Prefix, "/"))
			_, _ = client.Logical().DeleteWithContext(ctx, folderMetaPath)
		}
	}
	if cleanPrefix != "" {
		folderMetaPath := fmt.Sprintf("%s/metadata/%s", mount, cleanPrefix)
		_, _ = client.Logical().DeleteWithContext(ctx, folderMetaPath)
	}

	fmt.Printf("✓ Successfully deleted %d secret(s) under prefix %q\n", deletedCount, displayPrefix)
	return deletedCount, nil
}

// DeleteVaultPrefix is an alias for DeletePrefix.
func DeleteVaultPrefix(client *Client, prefix string, timeout time.Duration, verbose bool) (int, error) {
	return DeletePrefix(client, prefix, timeout, verbose)
}

// PrefixExists checks whether any secrets exist under the specified prefix.
func PrefixExists(client *Client, prefix string) bool {
	mount := os.Getenv("VAULT_MOUNT")
	if mount == "" {
		mount = "secret"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	clean := strings.Trim(prefix, "/")
	groups, err := collectPrefixGroups(ctx, client, strings.Trim(mount, "/"), clean)
	return err == nil && len(groups) > 0
}

// DeleteSecret deletes a secret, an individual JSON component, or a prefix from Vault KV v2.
func DeleteSecret(client *Client, secretPath string, timeout time.Duration, verbose bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	health, err := client.Sys().HealthWithContext(ctx)
	if err == nil && health != nil && health.Sealed {
		fmt.Println("Warning: Vault is sealed. Cannot delete secret until Vault is unsealed.")
		return fmt.Errorf("vault is sealed")
	}

	// 1. Explicit folder delete if path ends with /
	if strings.HasSuffix(secretPath, "/") {
		_, err := DeletePrefix(client, strings.Trim(secretPath, "/"), timeout, verbose)
		return err
	}

	// 2. Check if secretPath specifies a sub-field of an existing secret (e.g. mysql.username)
	if strings.Contains(secretPath, ".") {
		normPath := NormalizePath(secretPath)
		_, _, exists, _ := fetchSecretData(ctx, client, normPath)
		if !exists {
			parts := strings.Split(secretPath, ".")
			for i := len(parts) - 1; i >= 1; i-- {
				candidatePath := strings.Join(parts[:i], ".")
				cData, _, cExists, cErr := fetchSecretData(ctx, client, candidatePath)
				if cErr == nil && cExists && cData != nil {
					fieldSelector := parts[i:]
					if deleteNestedValue(cData, fieldSelector) {
						normCandidate := NormalizePath(candidatePath)
						if verbose {
							fmt.Printf("\n--- Deleting Property %s from %s ---\n", strings.Join(fieldSelector, "."), normCandidate)
						}
						payload := map[string]interface{}{
							"data": cData,
						}
						_, err := client.Logical().WriteWithContext(ctx, normCandidate, payload)
						if err != nil {
							return err
						}
						fmt.Printf("✓ Property %s successfully deleted from %s\n", strings.Join(fieldSelector, "."), normCandidate)
						return nil
					}
				}
			}
		}
	}

	// 3. Check if secretPath is actually a prefix containing child secrets
	normPath := NormalizePath(secretPath)
	_, _, leafExists, _ := fetchSecretData(ctx, client, normPath)
	if !leafExists {
		mount := os.Getenv("VAULT_MOUNT")
		if mount == "" {
			mount = "secret"
		}
		clean := strings.Trim(secretPath, "/")
		groups, gErr := collectPrefixGroups(ctx, client, strings.Trim(mount, "/"), clean)
		if gErr == nil && len(groups) > 0 {
			_, err := DeletePrefix(client, clean, timeout, verbose)
			return err
		}
	}

	if verbose {
		fmt.Printf("\n--- Deleting Secret from %s ---\n", normPath)
	}

	// Delete data endpoint
	_, err = client.Logical().DeleteWithContext(ctx, normPath)
	if err != nil {
		log.Printf("Error deleting secret at %s: %v", normPath, err)
		return err
	}

	// Delete metadata endpoint to purge completely
	metaPath := strings.Replace(normPath, "/data/", "/metadata/", 1)
	_, _ = client.Logical().DeleteWithContext(ctx, metaPath)

	fmt.Printf("✓ Secret successfully deleted from %s\n", normPath)
	return nil
}

// DeleteVaultSecret is an alias for DeleteSecret.
func DeleteVaultSecret(client *Client, secretPath string, timeout time.Duration, verbose bool) error {
	return DeleteSecret(client, secretPath, timeout, verbose)
}

// RemoveSecret is an alias for DeleteSecret.
func RemoveSecret(client *Client, secretPath string, timeout time.Duration, verbose bool) error {
	return DeleteSecret(client, secretPath, timeout, verbose)
}
