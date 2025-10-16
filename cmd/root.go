package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "near-swap",
	Short: "A CLI for cross-chain swaps using NEAR Intents 1Click API",
	Long: `near-swap is a command-line tool that enables easy cross-chain token swaps
using the NEAR Intents 1Click API. Simply specify what you want to swap and let
the NEAR Intents protocol handle the complexity.

Examples:
  near-swap swap 1 SOL to USDC
  near-swap swap 0.5 ETH to BTC --from-chain ethereum --to-chain bitcoin
  near-swap list-tokens
  near-swap status <intent-id>`,
	Version: "0.1.0",
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Add global flags
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolP("json", "j", false, "Output in JSON format")
}

func printError(err error) {
	fmt.Printf("\nError: %v\n\n", err)
}

func printSuccess(message string) {
	fmt.Printf("\n%s\n\n", message)
}
