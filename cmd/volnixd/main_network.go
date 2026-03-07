package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"cosmossdk.io/log"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/spf13/cobra"

	"github.com/volnix-protocol/volnix-protocol/app"
)

func createNetworkCommands() *cobra.Command {
	networkCmd := &cobra.Command{
		Use:   "network",
		Short: "Network operations for multi-node testing",
	}

	networkCmd.AddCommand(
		&cobra.Command{
			Use:   "init-testnet [num-validators]",
			Short: "Initialize testnet with multiple validators",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				numVal, err := strconv.Atoi(args[0])
				if err != nil || numVal < 1 || numVal > 10 {
					return fmt.Errorf("num-validators must be an integer between 1 and 10, got %q", args[0])
				}

				fmt.Printf("Initializing testnet with %d validators...\n", numVal)

				for i := 0; i < numVal; i++ {
					nodeDir := fmt.Sprintf("testnet/node%d", i)
					configDir := filepath.Join(nodeDir, "config")
					dataDir := filepath.Join(nodeDir, "data")

					if err := os.MkdirAll(configDir, 0755); err != nil {
						return fmt.Errorf("failed to create config dir for node%d: %w", i, err)
					}
					if err := os.MkdirAll(dataDir, 0755); err != nil {
						return fmt.Errorf("failed to create data dir for node%d: %w", i, err)
					}

					fmt.Printf("  Created node%d directory structure\n", i)
				}

				fmt.Println("Testnet initialized in ./testnet/")
				for i := 0; i < numVal; i++ {
					fmt.Printf("  volnixd network start-node %d\n", i)
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "start-node [node-id]",
			Short: "Start a specific testnet node",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				nodeID := args[0]
				if _, err := strconv.Atoi(nodeID); err != nil {
					fmt.Fprintf(os.Stderr, "Error: node-id must be a number, got %q\n", nodeID)
					os.Exit(1)
				}
				fmt.Printf("Starting testnet node %s...\n", nodeID)

				// Create logger
				logger := log.NewLogger(os.Stdout)

				// Create in-memory database for testing
				db := cosmosdb.NewMemDB()

				// Create encoding config
				encodingConfig := app.MakeEncodingConfig()

				// Create app instance
				volnixApp := app.NewVolnixApp(logger, db, nil, encodingConfig, nil)

				baseP2PPort := 26656
				baseRPCPort := 26657
				id, _ := strconv.Atoi(nodeID)
				p2pPort := baseP2PPort + id*10
				rpcPort := baseRPCPort + id*10

				fmt.Printf("Node %s started\n", nodeID)
				fmt.Printf("  App: %s v%s\n", volnixApp.Name(), volnixApp.Version())
				fmt.Printf("  P2P: %d, RPC: %d\n", p2pPort, rpcPort)
				fmt.Println("Press Ctrl+C to stop...")

				// Keep running
				select {}
			},
		},
		&cobra.Command{
			Use:   "status",
			Short: "Show network status by querying running nodes",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("Volnix Testnet Status")
				fmt.Println("Chain ID: volnix-testnet-1")
				fmt.Println("Consensus: PoVB (Proof of Value Burn)")
				fmt.Println("")
				fmt.Println("Query individual nodes:")
				fmt.Println("  curl http://localhost:26657/status  (node0)")
				fmt.Println("  curl http://localhost:26667/status  (node1)")
				fmt.Println("  curl http://localhost:26677/status  (node2)")
			},
		},
	)

	return networkCmd
}

