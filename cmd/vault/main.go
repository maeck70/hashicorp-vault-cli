package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/maeck70/hashicorp-vault-cli/pkg/vault-lib"
)

const VERSION = "0.5.0"

func main() {

	var flagArgs []string
	var cmdArgs []string
	var isVerbose bool

	boolFlags := map[string]bool{
		"-V": true, "--V": true, "-verbose": true, "--verbose": true,
		"-r": true, "-recursive": true, "--recursive": true,
		"-all": true, "--all": true,
		"-list": true, "--list": true, "-ls": true, "--ls": true,
		"-h": true, "-help": true, "--help": true,
		"-v": true, "-version": true, "--version": true,
	}

	valueFlags := map[string]bool{
		"-addr": true, "--addr": true,
		"-token": true, "--token": true,
		"-namespace": true, "--namespace": true,
		"-timeout": true, "--timeout": true,
		"-unseal-key": true, "--unseal-key": true,
		"-prefix": true, "--prefix": true, "-p": true, "--p": true,
		"-env": true, "--env": true,
		"-read": true, "--read": true, "-get": true, "--get": true,
		"-write": true, "--write": true, "-create": true, "--create": true,
		"-put": true, "--put": true, "-set": true, "--set": true, "-add": true, "--add": true,
		"-delete": true, "--delete": true, "-del": true, "--del": true,
		"-rm": true, "--rm": true, "-remove": true, "--remove": true,
	}

	rawArgs := os.Args[1:]
	var i int
	for i < len(rawArgs) {
		arg := rawArgs[i]

		switch arg {
		case "-V", "--V", "-verbose", "--verbose":
			isVerbose = true
		}

		switch {
		case strings.HasPrefix(arg, "-") && strings.Contains(arg, "="), boolFlags[arg]:
			flagArgs = append(flagArgs, arg)
		case valueFlags[arg]:
			flagArgs = append(flagArgs, arg)
			if i+1 < len(rawArgs) {
				i++
				flagArgs = append(flagArgs, rawArgs[i])
			}
		default:
			cmdArgs = append(cmdArgs, arg)
		}
		i++
	}

	// 1. Define command-line flags
	fs := flag.NewFlagSet("vault", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	envFile := fs.String("env", ".env", "Path to .env configuration file")
	addrFlag := fs.String("addr", "", "Vault server address (overrides VAULT_ADDR in .env)")
	tokenFlag := fs.String("token", "", "Vault authentication token (overrides VAULT_TOKEN in .env)")
	namespaceFlag := fs.String("namespace", "", "Vault namespace (overrides VAULT_NAMESPACE in .env)")
	timeoutFlag := fs.Duration("timeout", 10*time.Second, "Timeout for Vault server requests")
	unsealKeyFlag := fs.String("unseal-key", "", "Vault unseal shard key (overrides VAULT_UNSEAL_KEY in .env)")
	prefixFlag := fs.String("prefix", "", "Secret key prefix/folder (overrides VAULT_PREFIX in .env)")
	fs.StringVar(prefixFlag, "p", "", "Alias for -prefix")
	verboseFlag := fs.Bool("V", false, "Enable verbose output")
	fs.BoolVar(verboseFlag, "verbose", false, "Alias for -V")

	versionFlag := fs.Bool("v", false, "Display program version only")
	fs.BoolVar(versionFlag, "version", false, "Alias for -v")

	fs.Usage = func() {
		envMap, _ := godotenv.Read(*envFile)
		pfx := *prefixFlag
		if pfx == "" {
			pfx = resolveEnv("", envMap, "VAULT_PREFIX", "PREFIX")
		}
		printUsage(pfx)
	}

	// Command flag aliases for backwards compatibility
	readFlag := fs.String("read", "", "Read secret from specified key: -read <key>")
	fs.StringVar(readFlag, "get", "", "Alias for -read")

	writeFlag := fs.String("write", "", "Write secret to specified key: -write <key> [value]")
	fs.StringVar(writeFlag, "create", "", "Alias for -write")
	fs.StringVar(writeFlag, "put", "", "Alias for -write")
	fs.StringVar(writeFlag, "set", "", "Alias for -write")
	fs.StringVar(writeFlag, "add", "", "Alias for -write")

	deleteFlag := fs.String("delete", "", "Delete secret at specified key: -delete <key>")
	fs.StringVar(deleteFlag, "del", "", "Alias for -delete")
	fs.StringVar(deleteFlag, "rm", "", "Alias for -delete")
	fs.StringVar(deleteFlag, "remove", "", "Alias for -delete")

	recursiveFlag := fs.Bool("r", false, "Recursively delete all secrets under a prefix")
	fs.BoolVar(recursiveFlag, "recursive", false, "Alias for -r")
	fs.BoolVar(recursiveFlag, "all", false, "Alias for -r")

	listFlag := fs.Bool("list", false, "List secret keys in Vault: -list [prefix]")
	fs.BoolVar(listFlag, "ls", false, "Alias for -list")

	if err := fs.Parse(flagArgs); err != nil {
		if err == flag.ErrHelp {
			return
		}
		log.Fatalf("Error parsing flags: %v", err)
	}

	if *verboseFlag {
		isVerbose = true
	}

	if *versionFlag {
		fmt.Printf("HashiCorp Vault v%s\n", VERSION)
		return
	}

	// 2. Load configuration from .env file via godotenv
	envMap, err := godotenv.Read(*envFile)
	if err != nil && (*envFile != ".env" || !os.IsNotExist(err)) {
		if isVerbose {
			log.Printf("Note: Could not read env file %q: %v (falling back to environment variables)", *envFile, err)
		}
	}
	// Also populate process environment via godotenv so downstream packages (pkg/vault-lib) can access them
	_ = godotenv.Load(*envFile)

	// 3. Resolve connection parameters using godotenv-parsed values with environment fallbacks
	vaultAddr := strings.TrimRight(resolveEnv(*addrFlag, envMap, "VAULT_ADDR"), "/")
	if vaultAddr == "" {
		log.Fatal("Error: Vault address not configured. Set VAULT_ADDR in .env or provide -addr flag.")
	}

	vaultToken := resolveEnv(*tokenFlag, envMap, "VAULT_TOKEN")
	if vaultToken == "" {
		log.Fatal("Error: Vault token not configured. Set VAULT_TOKEN in .env or provide -token flag.")
	}

	vaultNamespace := resolveEnv(*namespaceFlag, envMap, "VAULT_NAMESPACE", "NAMESPACE")
	unsealKey := resolveEnv(*unsealKeyFlag, envMap, "VAULT_UNSEAL_KEY")

	var prefixExplicit bool
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "prefix", "p":
			prefixExplicit = true
		}
	})

	prefix := *prefixFlag
	if !prefixExplicit && prefix == "" {
		prefix = resolveEnv("", envMap, "VAULT_PREFIX", "PREFIX")
	}
	prefix = strings.Trim(prefix, "/")

	// 4. Handle "set-prefix" or "prefix" subcommand early if only modifying .env
	args := cmdArgs
	if len(args) == 0 {
		args = fs.Args()
	}
	if len(args) > 0 && (args[0] == "set-prefix" || args[0] == "prefix") {
		if len(args) > 1 {
			newPrefix := strings.Trim(args[1], "/")
			if err := vault.SetEnvPrefix(*envFile, newPrefix); err != nil {
				log.Fatalf("Error setting prefix in %s: %v", *envFile, err)
			}
			fmt.Printf("✓ Default prefix updated to %q in %s\n", newPrefix, *envFile)
			return
		}
		switch {
		case prefix != "":
			fmt.Printf("Current default prefix: %s\n", prefix)
		default:
			fmt.Println("No default prefix is currently set in .env.")
		}
		fmt.Printf("To change it: vault prefix <new-prefix>\n")
		return
	}

	// Determine command and parameters
	var command string
	var key string
	var value string

	switch {
	case *writeFlag != "":
		command = "write"
		key = *writeFlag
		if fs.NArg() > 0 {
			value = strings.Join(fs.Args(), " ")
		}
	case *readFlag != "":
		command = "read"
		key = *readFlag
	case *deleteFlag != "":
		command = "delete"
		key = *deleteFlag
	case *listFlag:
		command = "list"
		if fs.NArg() > 0 {
			key = fs.Arg(0)
		}
	case len(args) > 0:
		subcmd := strings.ToLower(args[0])
		switch subcmd {
		case "version":
			fmt.Printf("HashiCorp Vault v%s\n", VERSION)
			return
		case "help":
			fs.Usage()
			return
		case "write", "create", "put", "set", "add":
			command = "write"
			if len(args) > 1 {
				key = args[1]
			}
			if len(args) > 2 {
				value = strings.Join(args[2:], " ")
			}
		case "read", "get":
			command = "read"
			if len(args) > 1 {
				key = args[1]
			}
		case "delete", "del", "rm", "remove":
			command = "delete"
			if len(args) > 1 {
				key = args[1]
			}
		case "delete-prefix", "del-prefix", "rm-prefix", "rmdir":
			command = "delete"
			*recursiveFlag = true
			if len(args) > 1 {
				key = args[1]
			}
		case "list", "ls":
			command = "list"
			if len(args) > 1 {
				key = args[1]
			}
		case "unseal":
			command = "unseal"
			if len(args) > 1 {
				unsealKey = args[1]
			}
		case "status", "health", "info", "check":
			command = "status"
			isVerbose = true // Status command is explicitly verbose
		default:
			log.Fatalf("Unknown command: %q. Run 'vault -help' for usage.", subcmd)
		}
	}

	if command == "" && len(args) == 0 {
		fs.Usage()
		return
	}

	if isVerbose {
		fmt.Printf("Connecting to HashiCorp Vault at: %s\n", vaultAddr)
		if prefix != "" {
			fmt.Printf("Default Prefix: %s\n", prefix)
		}
		if vaultNamespace != "" {
			fmt.Printf("Namespace: %s\n", vaultNamespace)
		}
	}

	// 5. Initialize Vault client via pkg/vault-lib
	client, err := vault.NewClient(vault.Config{
		Address:   vaultAddr,
		Token:     vaultToken,
		Namespace: vaultNamespace,
		Timeout:   *timeoutFlag,
	})
	if err != nil {
		log.Fatalf("Vault connection error: %v", err)
	}

	// 6. Test connectivity, report health, and auto-unseal if key is provided
	vault.CheckConnection(client, *timeoutFlag, unsealKey, isVerbose)

	if isVerbose {
		fmt.Println("\nVault client successfully initialized!")
	}

	// 7. Execute Command
	switch command {
	case "write":
		if key == "" {
			log.Fatal("Error: Key is required for write. Usage: vault write <key> [value]")
		}
		targetKey := vault.ResolveKey(prefix, key)
		vault.CreateSecret(client, targetKey, value, *timeoutFlag, isVerbose)

	case "read":
		if key == "" {
			log.Fatal("Error: Key is required for read. Usage: vault read <key>")
		}
		targetKey := vault.ResolveKey(prefix, key)
		var secretValue string
		vault.ReadSecret(client, targetKey, *timeoutFlag, &secretValue, isVerbose)
		if secretValue != "" {
			fmt.Printf("Secret with key: %s and value: %s\n", targetKey, secretValue)
		}

	case "delete":
		// Handle full prefix deletion if requested
		if *recursiveFlag || strings.HasSuffix(key, "/") || key == "/" || key == "*" || key == "all" {
			targetPrefix := key
			switch {
			case targetPrefix == "/" || targetPrefix == "*" || targetPrefix == "all" || targetPrefix == "":
				targetPrefix = prefix
			case prefix != "" && targetPrefix != prefix && !strings.HasPrefix(targetPrefix, prefix+"/"):
				// If targetPrefix does not already exist at the mount root, resolve under active prefix
				if !vault.PrefixExists(client, targetPrefix) {
					targetPrefix = vault.ResolveKey(prefix, targetPrefix)
				}
			}
			targetPrefix = strings.Trim(targetPrefix, "/")
			if targetPrefix == "" {
				log.Fatal("Error: Target prefix cannot be empty for full prefix delete. Usage: vault delete -r <prefix>")
			}
			if _, err := vault.DeletePrefix(client, targetPrefix, *timeoutFlag, isVerbose); err != nil {
				log.Fatalf("Delete prefix failed: %v", err)
			}
			return
		}

		if key == "" {
			log.Fatal("Error: Key is required for delete. Usage: vault delete <key> or vault delete -r <prefix>")
		}
		targetKey := vault.ResolveKey(prefix, key)
		if err := vault.DeleteSecret(client, targetKey, *timeoutFlag, isVerbose); err != nil {
			log.Fatalf("Delete failed: %v", err)
		}

	case "list":
		switch {
		case key == "" && !prefixExplicit:
			key = prefix
		case prefixExplicit && key == "":
			key = ""
		}

		switch {
		case key == "/" || key == "all" || key == "*":
			key = ""
		case key != "" && prefix != "" && key != prefix && !strings.HasPrefix(key, prefix+"/"):
			if !vault.PrefixExists(client, key) {
				key = vault.ResolveKey(prefix, key)
			}
		}
		vault.ListKeys(client, key, *timeoutFlag, isVerbose)

	case "unseal":
		if unsealKey == "" {
			log.Fatal("Error: Unseal shard key required. Set VAULT_UNSEAL_KEY in .env or pass as argument: vault unseal <key>")
		}
		if _, err := vault.Unseal(client, unsealKey); err != nil {
			log.Fatalf("Error unsealing: %v", err)
		}

	case "status":
		// Checked and printed by CheckConnection with isVerbose=true

	default:
		printUsage(prefix)
	}
}

// resolveEnv returns flagVal if non-empty, otherwise looks up candidate keys in the
// parsed envMap (from godotenv.Read) and falls back to process environment variables (os.Getenv).
func resolveEnv(flagVal string, envMap map[string]string, keys ...string) string {
	if flagVal != "" {
		return flagVal
	}
	for _, key := range keys {
		if envMap != nil {
			if val, ok := envMap[key]; ok && val != "" {
				return val
			}
		}
		if val := os.Getenv(key); val != "" {
			return val
		}
	}
	return ""
}

func printUsage(defaultPrefix string) {
	fmt.Printf("HashiCorp Vault CLI v%s\n", VERSION)
	fmt.Println("A command-line tool for managing secrets, multi-level JSON configurations, prefixes, and server lifecycle in HashiCorp Vault.")
	fmt.Println("\nUsage: vault <command> [arguments] [flags]")
	fmt.Println("\nCommands & Aliases:")
	fmt.Println("  read <key>             Read a secret or JSON component (alias: get)")
	fmt.Println("  write <key> [value]    Write a secret, JSON object, or file (aliases: create, put, set, add)")
	fmt.Println("  delete <key>           Delete a secret or JSON component (aliases: del, rm, remove)")
	fmt.Println("  delete -r <prefix>     Recursively delete all secrets under prefix (aliases: delete-prefix, rmdir)")
	fmt.Println("  list [prefix]          List secret keys or group by prefix (alias: ls)")
	fmt.Println("  prefix [new-prefix]    View or set default prefix in .env (alias: set-prefix)")
	fmt.Println("  unseal [shard-key]     Unseal the Vault server")
	fmt.Println("  status                 Check Vault connectivity, server status, and token (aliases: health, info, check)")
	fmt.Println("  version                Display program version (alias: -v)")
	fmt.Println("\nFlags:")
	fmt.Println("  -h, -help              Show help information and program function")
	fmt.Println("  -v, -version           Display program version only")
	fmt.Println("  -V, -verbose           Enable verbose output")
	fmt.Println("  -r, -recursive, -all   Recursively delete all secrets under the target prefix")
	fmt.Println("  -prefix, -p <path>     Override default secret prefix for this command")
	fmt.Println("  -addr <url>            Override Vault server address")
	fmt.Println("  -token <token>         Override Vault authentication token")
	fmt.Println("  -namespace <ns>        Override Vault namespace")
	fmt.Println("  -timeout <duration>    Request timeout (default: 10s)")
	fmt.Println("  -env <path>            Path to .env file (default: .env)")
	if defaultPrefix != "" {
		fmt.Printf("\nCurrent Default Prefix: %s\n", defaultPrefix)
	}
}
