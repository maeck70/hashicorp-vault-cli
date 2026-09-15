#!/usr/bin/env bash
# ==============================================================================
# test_vault.sh - Comprehensive CLI Verification Test Script for HashiCorp Vault
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

VAULT_BIN="./bin/vault"

# Colors for terminal output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m' # No Color

pass_count=0
fail_count=0

pass() {
    echo -e "  ${GREEN}✓ PASS:${NC} $1"
    pass_count=$((pass_count + 1))
}

fail() {
    echo -e "  ${RED}✗ FAIL:${NC} $1"
    fail_count=$((fail_count + 1))
}

header() {
    echo -e "\n${BOLD}${BLUE}=== $1 ===${NC}"
}

echo -e "${BOLD}HashiCorp Vault CLI Functional Test Suite${NC}"
echo "=========================================="

# 1. Build Binary
header "1. Ensuring CLI Binary is Built"
if [ ! -f "$VAULT_BIN" ]; then
    echo "Binary not found, building with 'make build'..."
    make build
fi
if [ -x "$VAULT_BIN" ]; then
    pass "Binary $VAULT_BIN exists and is executable"
else
    fail "Binary $VAULT_BIN could not be built"
    exit 1
fi

# 2. Check Status & Connectivity
header "2. Testing Status & Connectivity"
status_output=$("$VAULT_BIN" status)
if echo "$status_output" | grep -q "Vault client successfully initialized"; then
    pass "Connected to Vault server and verified token"
else
    fail "Could not connect to Vault server"
    echo "$status_output"
    exit 1
fi

# 3. Test Prefix Management
header "3. Testing Prefix Configuration"
# Save original prefix to restore at the end
original_prefix=$("$VAULT_BIN" prefix | grep -o 'Current default prefix: .*' | awk '{print $NF}' || true)

echo "Setting prefix to 'smoketest'..."
set_prefix_output=$("$VAULT_BIN" prefix smoketest)
if echo "$set_prefix_output" | grep -q 'Default prefix updated to "smoketest"'; then
    pass "Updated prefix to 'smoketest' via 'vault prefix smoketest'"
else
    fail "Failed to update prefix"
    echo "$set_prefix_output"
fi

verify_prefix_output=$("$VAULT_BIN" prefix)
if echo "$verify_prefix_output" | grep -q 'Current default prefix: smoketest'; then
    pass "Confirmed prefix 'smoketest' persisted in .env"
else
    fail "Prefix was not persisted properly"
    echo "$verify_prefix_output"
fi

# 4. Test Writing Secret
header "4. Testing Secret Write"
test_key="api_key"
test_val="tok_live_987654321_secret"

write_output=$("$VAULT_BIN" write "$test_key" "$test_val")
if echo "$write_output" | grep -q "Secret successfully written"; then
    pass "Wrote secret '$test_key' under default prefix 'smoketest'"
else
    fail "Failed to write secret"
    echo "$write_output"
fi

# 5. Test Reading Secret
header "5. Testing Secret Read"
read_output=$("$VAULT_BIN" read "$test_key")
if echo "$read_output" | grep -q "$test_val"; then
    pass "Successfully read back secret value: $test_val"
else
    fail "Secret value did not match written value"
    echo "$read_output"
fi

# 6. Test Listing Keys
header "6. Testing Key Listing"
list_output=$("$VAULT_BIN" list)
if echo "$list_output" | grep -q "$test_key"; then
    pass "Key '$test_key' found in list output under 'smoketest'"
else
    fail "Key '$test_key' was not found in list output"
    echo "$list_output"
fi

# 7. Test Prefix Override Flag
header "7. Testing Flag Override (-prefix)"
override_key="custom_setting"
override_val="enabled_true"

override_output=$("$VAULT_BIN" -prefix manual_override write "$override_key" "$override_val")
if echo "$override_output" | grep -q "manual_override/$override_key"; then
    pass "Flag '-prefix manual_override' successfully overrode .env prefix"
else
    fail "Prefix override flag did not work"
    echo "$override_output"
fi

# Clean up override secret
"$VAULT_BIN" -prefix manual_override delete "$override_key" > /dev/null 2>&1 || true

# 8. Test Deleting Secret
header "8. Testing Secret Deletion"
delete_output=$("$VAULT_BIN" delete "$test_key")
if echo "$delete_output" | grep -q "Secret successfully deleted"; then
    pass "Deleted secret '$test_key'"
else
    fail "Failed to delete secret"
    echo "$delete_output"
fi

# Verify it's deleted upon read
read_after_delete=$("$VAULT_BIN" read "$test_key" 2>&1)
if echo "$read_after_delete" | grep -qiE "deleted|no secret found"; then
    pass "Verified secret reports deleted status upon read"
else
    fail "Secret still readable or deletion message missing"
    echo "$read_after_delete"
fi

# 9. Test Aliases (create, get, put, rm, ls)
header "9. Testing Command Aliases"
alias_key="alias_test_item"
alias_val="val_alpha"
alias_val2="val_beta"

# create
create_out=$("$VAULT_BIN" create "$alias_key" "$alias_val")
if echo "$create_out" | grep -q "Secret successfully written"; then
    pass "Alias 'create' successfully wrote secret"
else
    fail "Alias 'create' failed"
fi

# get
get_out=$("$VAULT_BIN" get "$alias_key")
if echo "$get_out" | grep -q "$alias_val"; then
    pass "Alias 'get' successfully read secret"
else
    fail "Alias 'get' failed"
fi

# put
put_out=$("$VAULT_BIN" put "$alias_key" "$alias_val2")
if echo "$put_out" | grep -q "Secret successfully written"; then
    pass "Alias 'put' successfully updated secret"
else
    fail "Alias 'put' failed"
fi

# ls
ls_out=$("$VAULT_BIN" ls)
if echo "$ls_out" | grep -q "$alias_key"; then
    pass "Alias 'ls' successfully listed secret"
else
    fail "Alias 'ls' failed"
fi

# rm
rm_out=$("$VAULT_BIN" rm "$alias_key")
if echo "$rm_out" | grep -q "Secret successfully deleted"; then
    pass "Alias 'rm' successfully deleted secret"
else
    fail "Alias 'rm' failed"
fi

# 10. Test Grouping Keys By Prefix (No Prefix)
header "10. Testing Listing with No Prefix (Grouped by Prefix)"
grouped_out=$("$VAULT_BIN" -prefix "" list)
if echo "$grouped_out" | grep -q "Prefix: (root)" && echo "$grouped_out" | grep -q "Prefix: startrek"; then
    pass "Correctly grouped keys by prefix when no prefix was provided"
else
    fail "Grouping by prefix failed"
    echo "$grouped_out"
fi

# 11. Test JSON & Multi-Level Component Retrieval
header "11. Testing JSON & Multi-Level Component Access"
json_test_key="mysql_test"
json_payload='{"username":"dbadmin","password":"secretpassword","host":"server192.168.10.10rpa","port":3306,"cluster":{"primary":{"zone":"us-east-1"}}}'

# Write JSON
write_json_out=$("$VAULT_BIN" write "$json_test_key" "$json_payload")
if echo "$write_json_out" | grep -q "Secret successfully written"; then
    pass "Successfully wrote JSON secret"
else
    fail "Failed writing JSON secret"
fi

# Read whole JSON
read_whole_out=$("$VAULT_BIN" read "$json_test_key")
if echo "$read_whole_out" | grep -q "dbadmin" && echo "$read_whole_out" | grep -q "us-east-1"; then
    pass "Successfully read whole JSON block"
else
    fail "Whole JSON read failed"
fi

# Read individual level-1 components
read_user_out=$("$VAULT_BIN" read "$json_test_key.username")
if echo "$read_user_out" | grep -q "dbadmin"; then
    pass "Read component '$json_test_key.username' -> dbadmin"
else
    fail "Component read username failed"
fi

read_port_out=$("$VAULT_BIN" read "$json_test_key.port")
if echo "$read_port_out" | grep -q "3306"; then
    pass "Read component '$json_test_key.port' -> 3306"
else
    fail "Component read port failed"
fi

# Read multi-level nested component
read_nested_out=$("$VAULT_BIN" read "$json_test_key.cluster.primary.zone")
if echo "$read_nested_out" | grep -q "us-east-1"; then
    pass "Read multi-level nested component '$json_test_key.cluster.primary.zone' -> us-east-1"
else
    fail "Multi-level component read failed"
fi

# Delete single component
del_comp_out=$("$VAULT_BIN" delete "$json_test_key.port")
if echo "$del_comp_out" | grep -q "successfully deleted"; then
    pass "Deleted component '$json_test_key.port'"
else
    fail "Deleting component failed"
fi

# Verify port is gone from JSON
read_after_comp_del=$("$VAULT_BIN" read "$json_test_key")
if echo "$read_after_comp_del" | grep -qv "3306"; then
    pass "Verified component 'port' is no longer in JSON block"
else
    fail "Component 'port' still present after deletion"
fi

# Clean up json test secret
"$VAULT_BIN" delete "$json_test_key" > /dev/null 2>&1 || true

# 12. Test Full Prefix Deletion
header "12. Testing Full Prefix Deletion"
prefix_demo="pfx_demo_folder"
"$VAULT_BIN" -prefix "$prefix_demo" write "service1" "alpha" > /dev/null
"$VAULT_BIN" -prefix "$prefix_demo" write "service2" "beta" > /dev/null
"$VAULT_BIN" -prefix "$prefix_demo" write "sub/nested_svc" "gamma" > /dev/null

# Verify secrets exist
pfx_list_before=$("$VAULT_BIN" list "$prefix_demo")
if echo "$pfx_list_before" | grep -q "service1" && echo "$pfx_list_before" | grep -q "service2"; then
    pass "Created multiple secrets under prefix '$prefix_demo'"
else
    fail "Failed setting up secrets under prefix '$prefix_demo'"
fi

# Delete full prefix
pfx_del_out=$("$VAULT_BIN" delete -r "$prefix_demo")
if echo "$pfx_del_out" | grep -q "Successfully deleted"; then
    pass "Command 'vault delete -r $prefix_demo' deleted all secrets under prefix"
else
    fail "Full prefix delete command failed"
    echo "$pfx_del_out"
fi

# Verify prefix is empty
pfx_list_after=$("$VAULT_BIN" list "$prefix_demo")
if echo "$pfx_list_after" | grep -q "No keys found"; then
    pass "Verified prefix '$prefix_demo' has no remaining keys"
else
    fail "Prefix still contains keys after deletion"
    echo "$pfx_list_after"
fi

# 13. Test -v (version), -h (help), and -V (verbose) Flags
header "13. Testing -v, -h, and Verbosity Flags (-V / --verbose)"
ver_out=$("$VAULT_BIN" -v)
if [ "$ver_out" = "HashiCorp Vault v0.5.0" ]; then
    pass "vault -v outputs exactly 'HashiCorp Vault v0.5.0'"
else
    fail "vault -v output unexpected: '$ver_out'"
fi

help_out=$("$VAULT_BIN" -h)
if echo "$help_out" | grep -q "HashiCorp Vault CLI v0.5.0" && echo "$help_out" | grep -q "A command-line tool"; then
    pass "vault -h reports program name and its function"
else
    fail "vault -h output missing program description"
fi

# Verbosity quiet vs verbose check
"$VAULT_BIN" write "v_sample_key" "sample_value" > /dev/null

quiet_out=$("$VAULT_BIN" read v_sample_key)
if echo "$quiet_out" | grep -q "Vault Server Status" || echo "$quiet_out" | grep -q "Connecting to HashiCorp"; then
    fail "vault read in default quiet mode emitted verbose connection banners"
else
    pass "vault read is quiet by default (no connection banners)"
fi

verbose_out=$("$VAULT_BIN" -V read v_sample_key)
if echo "$verbose_out" | grep -q "Vault Server Status" && echo "$verbose_out" | grep -q "Reading Secret from"; then
    pass "vault -V read emits consistent server status and operation banners"
else
    fail "vault -V read missing expected verbose diagnostic banners"
fi

verbose_long_out=$("$VAULT_BIN" --verbose read v_sample_key)
if echo "$verbose_long_out" | grep -q "Vault Server Status" && echo "$verbose_long_out" | grep -q "Reading Secret from"; then
    pass "vault --verbose produces identical verbose banners to -V"
else
    fail "vault --verbose failed to enable verbose output"
fi

"$VAULT_BIN" delete "v_sample_key" > /dev/null

# 14. Cleanup / Restore Prefix
header "14. Restoring Original Environment"
if [ -n "$original_prefix" ]; then
    "$VAULT_BIN" prefix "$original_prefix" > /dev/null
    pass "Restored original prefix '$original_prefix'"
else
    # Clear out smoketest if there was no prefix
    "$VAULT_BIN" prefix "" > /dev/null
    pass "Cleared test prefix"
fi

# 10. Summary
echo ""
echo "=========================================="
echo -e "${BOLD}Test Results:${NC} ${GREEN}$pass_count passed${NC}, ${RED}$fail_count failed${NC}"
echo "=========================================="

if [ "$fail_count" -eq 0 ]; then
    echo -e "${GREEN}${BOLD}ALL TESTS PASSED SUCCESSFULLY!${NC}\n"
    exit 0
else
    echo -e "${RED}${BOLD}SOME TESTS FAILED!${NC}\n"
    exit 1
fi
