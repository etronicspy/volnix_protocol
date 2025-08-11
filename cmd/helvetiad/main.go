package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Application version and git commit. Commit is injected via -ldflags at build time.
var (
	version = "0.1.0"
	commit  = "dev"
)

func main() {
	rootCmd := &cobra.Command{
		Use:           "helvetiad",
		Short:         "Helvetia Protocol daemon",
		Long:          "Helvetia Protocol (H•P) — sovereign L1 blockchain on Cosmos SDK. Bootstrap daemon.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newStartCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print helvetiad version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("helvetiad %s (%s)\n", version, commit)
		},
	}
	return cmd
}

func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start Helvetia node (placeholder)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Placeholder: In later iterations this will initialize and start a Cosmos SDK app.
			fmt.Println("Starting Helvetia node (placeholder). Cosmos SDK app wiring will be added in later iterations.")
			return nil
		},
	}
	return cmd
}
