package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cometbft/cometbft/libs/log"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	sdklog "cosmossdk.io/log"

	apppkg "github.com/volnix-protocol/volnix-protocol/app"
)

// Application version and git commit. Commit is injected via -ldflags at build time.
var (
	Version = "0.1.0"
	Commit  = ""
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
	rootCmd.AddCommand(newTxCmd())
	rootCmd.AddCommand(newQueryCmd())

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
				"genesis_time":   "2024-08-14T20:00:00Z",
				"chain_id":       "test-volnix",
				"initial_height": "1",
				"consensus_params": map[string]interface{}{
					"block": map[string]interface{}{
						"max_bytes":    "22020096",
						"max_gas":      "-1",
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
				"validators": []map[string]interface{}{
					{
						"address": "73DB49F602BBEAF45B2F56AE13A44B462D0F1EC0",
						"pub_key": map[string]interface{}{
							"type":  "tendermint/PubKeySecp256k1",
							"value": "Aqj+0BaJ0xAbIcUQpVPQ9hBM1qQ/Nn3Bkyo7hOkuI0Xb",
						},
						"power":     "1000000",
						"name":      "test-validator",
						"proposer_priority": "0",
					},
				},
				"app_hash": "",
				"app_state": map[string]interface{}{
					"anteil": map[string]interface{}{
						"params": map[string]interface{}{
							"min_ant_amount":                 "1000000",
							"max_ant_amount":                 "1000000000000",
							"trading_fee_rate":               "0.001",
							"order_expiry":                   "3600s",
							"identity_verification_required": true,
							"ant_denom":                      "uant",
							"max_open_orders":                100,
							"price_precision":                8,
						},
						"orders":         []interface{}{},
						"trades":         []interface{}{},
						"user_positions": []interface{}{},
						"auctions":       []interface{}{},
						"order_book": map[string]interface{}{
							"buy_orders":   []interface{}{},
							"sell_orders":  []interface{}{},
							"last_price":   "0",
							"volume_24h":   "0",
							"total_orders": 0,
						},
					},
					"ident": map[string]interface{}{
						"params": map[string]interface{}{
							"verification_cost":          "1000000uvx",
							"migration_fee":              "500000uvx",
							"citizen_activity_period":    "31536000s",
							"validator_activity_period":  "15768000s",
							"max_identities_per_address": 1,
						},
						"identities": []interface{}{},
						"roles":      []interface{}{},
						"migrations": []interface{}{},
					},
					"lizenz": map[string]interface{}{
						"params": map[string]interface{}{
							"activation_cost":          "1000000uvx",
							"deactivation_fee":         "1000000uvx",
							"min_activity_period":      "2592000s",
							"max_lizenz_per_validator": 10,
						},
						"lizenz":      []interface{}{},
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
			fmt.Printf("volnixd %s (%s)\n", Version, Commit)
		},
	}
	return cmd
}

func newStartCmd() *cobra.Command {
	var homeDir string

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start Волникс Протокол node with CometBFT blockchain",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Bech32 prefixes
			cfg := sdk.GetConfig()
			cfg.SetBech32PrefixForAccount("vx", "vxpub")
			cfg.SetBech32PrefixForValidator("vxvaloper", "vxvaloperpub")
			cfg.SetBech32PrefixForConsensusNode("vxvalcons", "vxvalconspub")
			cfg.Seal()

			// Get home directory
			if homeDir == "" {
				homeDir = os.Getenv("HOME") + "/.volnix"
			}
			if _, err := os.Stat(homeDir); os.IsNotExist(err) {
				return fmt.Errorf("home directory %s does not exist. Please run 'volnixd init [moniker]' first", homeDir)
			}

			// Create database directory
			dbPath := filepath.Join(homeDir, "data")
			if err := os.MkdirAll(dbPath, 0755); err != nil {
				return fmt.Errorf("failed to create data directory: %w", err)
			}

			// Use goleveldb for persistence
			database, err := dbm.NewDB("application", dbm.GoLevelDBBackend, dbPath)
			if err != nil {
				return fmt.Errorf("failed to create database: %w", err)
			}
			defer database.Close()

			// Build app with persistent storage
			encoding := apppkg.MakeEncodingConfig()
			logger := sdklog.NewNopLogger()
			app := apppkg.NewVolnixApp(logger, database, nil, encoding)

			// Load latest version
			if err := app.LoadLatestVersion(); err != nil {
				return fmt.Errorf("failed to load latest version: %w", err)
			}

			// Create CometBFT node
			fmt.Println("🚀 Starting Волникс Протокол with CometBFT consensus...")
			
			// Create CometBFT logger
			cometLogger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
			
			// Create CometBFT node
			node, err := NewCometBFTNode(homeDir, cometLogger)
			if err != nil {
				return fmt.Errorf("failed to create CometBFT node: %w", err)
			}

			// Start CometBFT node
			if err := node.Start(); err != nil {
				return fmt.Errorf("failed to start CometBFT node: %w", err)
			}

			// Wait for shutdown signal
			node.WaitForShutdown()
			return nil
		},
	}

	cmd.Flags().StringVar(&homeDir, "home", "", "Directory for config and data (default: $HOME/.volnix)")

	return cmd
}

func newTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tx",
		Short: "Volnix Protocol transactions",
		Long:  "Send and manage transactions on Volnix Protocol",
	}

	cmd.AddCommand(newIdentTxCmd())
	cmd.AddCommand(newLizenzTxCmd())
	cmd.AddCommand(newAnteilTxCmd())

	return cmd
}

func newIdentTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ident",
		Short: "Identity module transactions",
		Long:  "Manage identity verification and role changes",
	}

	cmd.AddCommand(newVerifyIdentityCmd())
	cmd.AddCommand(newMigrateRoleCmd())
	cmd.AddCommand(newChangeRoleCmd())

	return cmd
}

func newVerifyIdentityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify-identity [address] [role]",
		Short: "Verify identity for an address",
		Long:  "Verify identity and assign role (guest/citizen/validator)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			address := args[0]
			role := args[1]

			fmt.Printf("🔐 Verifying identity for address: %s\n", address)
			fmt.Printf("🎭 Role: %s\n", role)
			fmt.Printf("📝 This would send a MsgVerifyIdentity transaction\n")
			fmt.Printf("⚠️  ZKP proof verification not yet implemented\n")

			return nil
		},
	}
	return cmd
}

func newMigrateRoleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate-role [new-address]",
		Short: "Migrate role to new address",
		Long:  "Migrate identity and role to a new address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			newAddress := args[0]

			fmt.Printf("🔄 Migrating role to new address: %s\n", newAddress)
			fmt.Printf("📝 This would send a MsgMigrateRole transaction\n")
			fmt.Printf("⚠️  ZKP proof verification not yet implemented\n")

			return nil
		},
	}
	return cmd
}

func newChangeRoleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "change-role [address] [new-role]",
		Short: "Change role for an address",
		Long:  "Change the role of an existing verified address",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			address := args[0]
			newRole := args[1]

			fmt.Printf("🔄 Changing role for address: %s\n", address)
			fmt.Printf("🎭 New role: %s\n", newRole)
			fmt.Printf("📝 This would send a MsgChangeRole transaction\n")

			return nil
		},
	}
	return cmd
}

func newLizenzTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lizenz",
		Short: "Lizenz module transactions",
		Long:  "Manage LZN token activation and deactivation",
	}

	cmd.AddCommand(newActivateLizenzCmd())
	cmd.AddCommand(newDeactivateLizenzCmd())

	return cmd
}

func newActivateLizenzCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "activate [validator] [amount]",
		Short: "Activate LZN tokens for validator",
		Long:  "Activate LZN tokens to participate in consensus",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			validator := args[0]
			amount := args[1]

			fmt.Printf("🔓 Activating LZN for validator: %s\n", validator)
			fmt.Printf("💰 Amount: %s\n", amount)
			fmt.Printf("📝 This would send a MsgActivateLZN transaction\n")
			fmt.Printf("⚠️  Identity verification not yet implemented\n")

			return nil
		},
	}
	return cmd
}

func newDeactivateLizenzCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deactivate [validator] [amount] [reason]",
		Short: "Deactivate LZN tokens for validator",
		Long:  "Deactivate LZN tokens with optional reason",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			validator := args[0]
			amount := args[1]
			reason := args[2]

			fmt.Printf("🔒 Deactivating LZN for validator: %s\n", validator)
			fmt.Printf("💰 Amount: %s\n", amount)
			fmt.Printf("📝 Reason: %s\n", reason)
			fmt.Printf("📝 This would send a MsgDeactivateLZN transaction\n")

			return nil
		},
	}
	return cmd
}

func newAnteilTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "anteil",
		Short: "Anteil module transactions",
		Long:  "Manage ANT trading and auctions",
	}

	cmd.AddCommand(newPlaceOrderCmd())
	cmd.AddCommand(newCancelOrderCmd())
	cmd.AddCommand(newPlaceBidCmd())

	return cmd
}

func newPlaceOrderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "place-order [owner] [type] [side] [amount] [price]",
		Short: "Place a new order",
		Long:  "Place a buy or sell order on the ANT market",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner := args[0]
			orderType := args[1]
			side := args[2]
			amount := args[3]
			price := args[4]

			fmt.Printf("📊 Placing order for owner: %s\n", owner)
			fmt.Printf("📝 Type: %s\n", orderType)
			fmt.Printf("📈 Side: %s\n", side)
			fmt.Printf("💰 Amount: %s ANT\n", amount)
			fmt.Printf("💵 Price: %s\n", price)
			fmt.Printf("📝 This would send a MsgPlaceOrder transaction\n")

			return nil
		},
	}
	return cmd
}

func newCancelOrderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel-order [owner] [order-id]",
		Short: "Cancel an existing order",
		Long:  "Cancel an order by ID",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner := args[0]
			orderID := args[1]

			fmt.Printf("❌ Cancelling order for owner: %s\n", owner)
			fmt.Printf("🆔 Order ID: %s\n", orderID)
			fmt.Printf("📝 This would send a MsgCancelOrder transaction\n")

			return nil
		},
	}
	return cmd
}

func newPlaceBidCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "place-bid [bidder] [auction-id] [amount]",
		Short: "Place a bid in auction",
		Long:  "Place a bid in an ANT auction",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			bidder := args[0]
			auctionID := args[1]
			amount := args[2]

			fmt.Printf("🏆 Placing bid in auction\n")
			fmt.Printf("👤 Bidder: %s\n", bidder)
			fmt.Printf("🎯 Auction ID: %s\n", auctionID)
			fmt.Printf("💰 Amount: %s\n", amount)
			fmt.Printf("📝 This would send a MsgPlaceBid transaction\n")

			return nil
		},
	}
	return cmd
}

func newQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query Volnix Protocol state",
		Long:  "Query the current state of Volnix Protocol modules",
	}

	cmd.AddCommand(newIdentQueryCmd())
	cmd.AddCommand(newLizenzQueryCmd())
	cmd.AddCommand(newAnteilQueryCmd())

	return cmd
}

func newIdentQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ident",
		Short: "Query identity module state",
		Long:  "Query verified accounts and roles",
	}

	cmd.AddCommand(newQueryVerifiedAccountCmd())
	cmd.AddCommand(newQueryVerifiedAccountsCmd())

	return cmd
}

func newQueryVerifiedAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account [address]",
		Short: "Query verified account by address",
		Long:  "Get details of a verified account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			address := args[0]
			fmt.Printf("🔍 Querying verified account: %s\n", address)
			fmt.Printf("📝 This would query the ident module state\n")
			return nil
		},
	}
	return cmd
}

func newQueryVerifiedAccountsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "List all verified accounts",
		Long:  "Get list of all verified accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("📋 Listing all verified accounts\n")
			fmt.Printf("📝 This would query the ident module state\n")
			return nil
		},
	}
	return cmd
}

func newLizenzQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lizenz",
		Short: "Query lizenz module state",
		Long:  "Query LZN activations and MOA status",
	}

	cmd.AddCommand(newQueryActivatedLizenzCmd())
	cmd.AddCommand(newQueryMOAStatusCmd())

	return cmd
}

func newQueryActivatedLizenzCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "activated [validator]",
		Short: "Query activated LZN for validator",
		Long:  "Get activated LZN details for a validator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			validator := args[0]
			fmt.Printf("🔍 Querying activated LZN for validator: %s\n", validator)
			fmt.Printf("📝 This would query the lizenz module state\n")
			return nil
		},
	}
	return cmd
}

func newQueryMOAStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "moa [validator]",
		Short: "Query MOA status for validator",
		Long:  "Get Minimum Obligation of Activity status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			validator := args[0]
			fmt.Printf("🔍 Querying MOA status for validator: %s\n", validator)
			fmt.Printf("📝 This would query the lizenz module state\n")
			return nil
		},
	}
	return cmd
}

func newAnteilQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "anteil",
		Short: "Query anteil module state",
		Long:  "Query orders, trades, and auctions",
	}

	cmd.AddCommand(newQueryOrderBookCmd())
	cmd.AddCommand(newQueryUserPositionCmd())

	return cmd
}

func newQueryOrderBookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "orderbook",
		Short: "Query current order book",
		Long:  "Get the current state of the ANT order book",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("📊 Querying ANT order book\n")
			fmt.Printf("📝 This would query the anteil module state\n")
			return nil
		},
	}
	return cmd
}

func newQueryUserPositionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "position [user]",
		Short: "Query user position",
		Long:  "Get trading position for a specific user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			user := args[0]
			fmt.Printf("🔍 Querying position for user: %s\n", user)
			fmt.Printf("📝 This would query the anteil module state\n")
			return nil
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
				"name":     name,
				"type":     "secp256k1",
				"address":  address.String(),
				"pubkey":   pubKey.String(),
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
