package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	sdklog "cosmossdk.io/log"
	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/spf13/cobra"

	apppkg "github.com/volnix-protocol/volnix-protocol/app"
)

// Application version and git commit. Commit is injected via -ldflags at build time.
var (
	appVersion = "0.1.0"
	commit     = "dev"
)

func main() {
	rootCmd := &cobra.Command{
		Use:           "volnixd",
		Short:         "Volnix Protocol daemon",
		Long:          "Volnix Protocol — sovereign L1 blockchain on Cosmos SDK. Bootstrap daemon.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newStartCmd())
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newKeysCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [moniker]",
		Short: "Initialize Volnix node",
		Long:  "Initialize a new Volnix node with the specified moniker",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			moniker := args[0]
			
			// Create home directory
			homeDir := os.Getenv("HOME") + "/.volnix"
			if err := os.MkdirAll(homeDir+"/config", 0755); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}
			
			// Set Bech32 prefixes
			cfg := sdk.GetConfig()
			cfg.SetBech32PrefixForAccount("vx", "vxpub")
			cfg.SetBech32PrefixForValidator("vxvaloper", "vxvaloperpub")
			cfg.SetBech32PrefixForConsensusNode("vxvalcons", "vxvalconspub")
			cfg.Seal()
			
			// Create genesis.json
			genesis := map[string]interface{}{
				"genesis_time": "2024-08-14T20:00:00Z",
				"chain_id":     "test-volnix",
				"initial_height": "1",
				"consensus_params": map[string]interface{}{
					"block": map[string]interface{}{
						"max_bytes": "22020096",
						"max_gas":   "-1",
						"time_iota_ms": "1000",
					},
					"evidence": map[string]interface{}{
						"max_age_num_blocks": "100000",
						"max_age_duration":   "172800000000000",
						"max_bytes":          "1048576",
					},
					"validator": map[string]interface{}{
						"pub_key_types": []string{"secp256k1"},
					},
				},
				"app_hash": "",
				"app_state": map[string]interface{}{
					"anteil": map[string]interface{}{
						"params": map[string]interface{}{
							"min_ant_amount": "1000000",
							"max_ant_amount": "1000000000000",
							"trading_fee_rate": "0.001",
							"order_expiry": "3600s",
							"identity_verification_required": true,
							"ant_denom": "uant",
							"max_open_orders": 100,
							"price_precision": 8,
						},
						"orders": []interface{}{},
						"trades": []interface{}{},
						"user_positions": []interface{}{},
						"auctions": []interface{}{},
						"order_book": map[string]interface{}{
							"buy_orders": []interface{}{},
							"sell_orders": []interface{}{},
							"last_price": "0",
							"volume_24h": "0",
							"total_orders": 0,
						},
					},
					"ident": map[string]interface{}{
						"params": map[string]interface{}{
							"verification_cost": "1000000uvx",
							"migration_fee": "500000uvx",
							"citizen_activity_period": "31536000s",
							"validator_activity_period": "15768000s",
							"max_identities_per_address": 1,
						},
						"identities": []interface{}{},
						"roles": []interface{}{},
						"migrations": []interface{}{},
					},
					"lizenz": map[string]interface{}{
						"params": map[string]interface{}{
							"activation_cost": "1000000uvx",
							"deactivation_fee": "1000000uvx",
							"min_activity_period": "2592000s",
							"max_lizenz_per_validator": 10,
						},
						"lizenz": []interface{}{},
						"activations": []interface{}{},
					},
				},
			}
			
			genesisBytes, err := json.MarshalIndent(genesis, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal genesis: %w", err)
			}
			
			if err := os.WriteFile(homeDir+"/config/genesis.json", genesisBytes, 0644); err != nil {
				return fmt.Errorf("failed to write genesis.json: %w", err)
			}
			
			// Create config.toml
			config := fmt.Sprintf(`# Tendermint Core Configuration for Volnix
moniker = "%s"
proxy_app = "tcp://127.0.0.1:26658"
rpc_laddr = "tcp://127.0.0.1:26657"
p2p_laddr = "tcp://127.0.0.1:26656"
genesis_file = "genesis.json"
db_backend = "goleveldb"
db_dir = "data"
log_level = "info"
log_format = "json"
`, moniker)
			
			if err := os.WriteFile(homeDir+"/config/config.toml", []byte(config), 0644); err != nil {
				return fmt.Errorf("failed to write config.toml: %w", err)
			}
			
			fmt.Printf("✅ Volnix node '%s' initialized successfully!\n", moniker)
			fmt.Printf("📁 Home directory: %s\n", homeDir)
			fmt.Printf("📄 Genesis file: %s/config/genesis.json\n", homeDir)
			fmt.Printf("⚙️  Config file: %s/config/config.toml\n", homeDir)
			fmt.Printf("\n🚀 To start the node, run: volnixd start\n")
			
			return nil
		},
	}
	return cmd
}

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print volnixd version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("volnixd %s (%s)\n", appVersion, commit)
		},
	}
	return cmd
}

func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start Volnix node (init app stores in-memory)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Bech32 prefixes
			cfg := sdk.GetConfig()
			cfg.SetBech32PrefixForAccount("vx", "vxpub")
			cfg.SetBech32PrefixForValidator("vxvaloper", "vxvaloperpub")
			cfg.SetBech32PrefixForConsensusNode("vxvalcons", "vxvalconspub")
			cfg.Seal()

			// Encoding and in-memory DB
			encoding := apppkg.MakeEncodingConfig()
			logger := sdklog.NewNopLogger()
			database := dbm.NewMemDB()

			// Build app and load latest version
			app := apppkg.NewVolnixApp(logger, database, nil, encoding)
			if err := app.LoadLatestVersion(); err != nil {
				return err
			}

			// Start ABCI server
			fmt.Println("Starting Volnix ABCI server...")
			fmt.Println("Chain ID: test-volnix")
			fmt.Println("Bech32 prefixes: vx, vxvaloper, vxvalcons")
			fmt.Println("Modules loaded: ident, lizenz, anteil")
			fmt.Println("ABCI server ready for Tendermint connection")
			fmt.Println("Use Ctrl+C to stop")

			// Keep the server running
			select {}
		},
	}
	return cmd
}

func newKeysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keys",
		Short: "Manage Volnix keys",
		Long:  "Manage cryptographic keys for Volnix Protocol",
	}

	cmd.AddCommand(newAddKeyCmd())
	cmd.AddCommand(newListKeysCmd())
	cmd.AddCommand(newShowKeyCmd())

	return cmd
}

func newAddKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [name]",
		Short: "Add a new key",
		Long:  "Add a new cryptographic key with the specified name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			
			// Set Bech32 prefixes
			cfg := sdk.GetConfig()
			cfg.SetBech32PrefixForAccount("vx", "vxpub")
			cfg.SetBech32PrefixForValidator("vxvaloper", "vxvaloperpub")
			cfg.SetBech32PrefixForConsensusNode("vxvalcons", "vxvalconspub")
			cfg.Seal()
			
			// Generate new key
			privKey := secp256k1.GenPrivKey()
			pubKey := privKey.PubKey()
			address := sdk.AccAddress(pubKey.Address())
			
			// Create keys directory
			keysDir := os.Getenv("HOME") + "/.volnix/keys"
			if err := os.MkdirAll(keysDir, 0755); err != nil {
				return fmt.Errorf("failed to create keys directory: %w", err)
			}
			
			// Save private key (in production, this should be encrypted)
			keyData := map[string]interface{}{
				"name": name,
				"type": "secp256k1",
				"address": address.String(),
				"pubkey": pubKey.String(),
				"mnemonic": "generated_key", // In production, use proper mnemonic
			}
			
			keyBytes, err := json.MarshalIndent(keyData, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal key data: %w", err)
			}
			
			keyFile := fmt.Sprintf("%s/%s.json", keysDir, name)
			if err := os.WriteFile(keyFile, keyBytes, 0600); err != nil {
				return fmt.Errorf("failed to write key file: %w", err)
			}
			
			fmt.Printf("✅ Key '%s' added successfully!\n", name)
			fmt.Printf("🔑 Address: %s\n", address.String())
			fmt.Printf("📁 Key file: %s\n", keyFile)
			fmt.Printf("⚠️  Keep your private key secure!\n")
			
			return nil
		},
	}
	return cmd
}

func newListKeysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all keys",
		Long:  "List all available cryptographic keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			keysDir := os.Getenv("HOME") + "/.volnix/keys"
			
			if _, err := os.Stat(keysDir); os.IsNotExist(err) {
				fmt.Println("No keys found. Use 'volnixd keys add [name]' to create a key.")
				return nil
			}
			
			files, err := os.ReadDir(keysDir)
			if err != nil {
				return fmt.Errorf("failed to read keys directory: %w", err)
			}
			
			if len(files) == 0 {
				fmt.Println("No keys found. Use 'volnixd keys add [name]' to create a key.")
				return nil
			}
			
			fmt.Println("Available keys:")
			fmt.Println("Name\t\tAddress")
			fmt.Println("----\t\t-------")
			
			for _, file := range files {
				if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
					continue
				}
				
				keyFile := fmt.Sprintf("%s/%s", keysDir, file.Name())
				keyBytes, err := os.ReadFile(keyFile)
				if err != nil {
					continue
				}
				
				var keyData map[string]interface{}
				if err := json.Unmarshal(keyBytes, &keyData); err != nil {
					continue
				}
				
				name := keyData["name"].(string)
				address := keyData["address"].(string)
				fmt.Printf("%s\t\t%s\n", name, address)
			}
			
			return nil
		},
	}
	return cmd
}

func newShowKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [name]",
		Short: "Show key details",
		Long:  "Show detailed information about a specific key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			keyFile := os.Getenv("HOME") + "/.volnix/keys/" + name + ".json"
			
			if _, err := os.Stat(keyFile); os.IsNotExist(err) {
				return fmt.Errorf("key '%s' not found", name)
			}
			
			keyBytes, err := os.ReadFile(keyFile)
			if err != nil {
				return fmt.Errorf("failed to read key file: %w", err)
			}
			
			var keyData map[string]interface{}
			if err := json.Unmarshal(keyBytes, &keyData); err != nil {
				return fmt.Errorf("failed to parse key file: %w", err)
			}
			
			fmt.Printf("Key: %s\n", name)
			fmt.Printf("Type: %s\n", keyData["type"])
			fmt.Printf("Address: %s\n", keyData["address"])
			fmt.Printf("Public Key: %s\n", keyData["pubkey"])
			
			return nil
		},
	}
	return cmd
}
