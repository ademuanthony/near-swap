package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	oneclick "github.com/defuse-protocol/one-click-sdk-go"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"near-swap/config"
	"near-swap/pkg/client"
)

var (
	filterChain  string
	filterSymbol string
)

var tokensCmd = &cobra.Command{
	Use:     "list-tokens",
	Aliases: []string{"tokens", "ls"},
	Short:   "List all supported tokens",
	Long: `List all tokens supported by the NEAR Intents 1Click API.

You can filter tokens by blockchain or symbol.

Examples:
  near-swap list-tokens
  near-swap list-tokens --chain solana
  near-swap list-tokens --symbol USDC`,
	Run: runListTokens,
}

func init() {
	rootCmd.AddCommand(tokensCmd)

	tokensCmd.Flags().StringVar(&filterChain, "chain", "", "Filter by blockchain")
	tokensCmd.Flags().StringVar(&filterSymbol, "symbol", "", "Filter by token symbol")
}

func runListTokens(cmd *cobra.Command, args []string) {
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	// Create client
	apiClient := client.NewOneClickClient(cfg.JWTToken)

	// Get tokens with spinner
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !jsonOutput {
		s.Suffix = " Fetching supported tokens..."
		s.Start()
	}

	tokens, err := apiClient.GetSupportedTokens()
	if !jsonOutput {
		s.Stop()
	}

	if err != nil {
		printError(err)
		os.Exit(1)
	}

	// Apply filters
	filtered := tokens
	if filterChain != "" {
		var temp []oneclick.TokenResponse
		for _, token := range filtered {
			if strings.EqualFold(token.GetBlockchain(), filterChain) {
				temp = append(temp, token)
			}
		}
		filtered = temp
	}

	if filterSymbol != "" {
		var temp []oneclick.TokenResponse
		for _, token := range filtered {
			if strings.Contains(strings.ToUpper(token.GetSymbol()), strings.ToUpper(filterSymbol)) {
				temp = append(temp, token)
			}
		}
		filtered = temp
	}

	// Output
	if jsonOutput {
		jsonData, _ := json.MarshalIndent(filtered, "", "  ")
		fmt.Println(string(jsonData))
	} else {
		displayTokens(filtered)
	}
}

func displayTokens(tokens []oneclick.TokenResponse) {
	if len(tokens) == 0 {
		fmt.Println("\nNo tokens found matching the criteria.")
		return
	}

	fmt.Println("\n" + strings.Repeat("=", 90))
	color.Green("                            SUPPORTED TOKENS")
	fmt.Println(strings.Repeat("=", 90))

	// Group tokens by blockchain
	tokensByChain := make(map[string][]oneclick.TokenResponse)
	for _, token := range tokens {
		chain := token.GetBlockchain()
		tokensByChain[chain] = append(tokensByChain[chain], token)
	}

	// Sort chains alphabetically
	chains := make([]string, 0, len(tokensByChain))
	for chain := range tokensByChain {
		chains = append(chains, chain)
	}
	sort.Strings(chains)

	// Display tokens grouped by chain
	for _, chain := range chains {
		color.Cyan("\n%s", strings.ToUpper(chain))
		fmt.Println(strings.Repeat("-", 90))

		chainTokens := tokensByChain[chain]
		for _, token := range chainTokens {
			symbol := token.GetSymbol()
			decimals := token.GetDecimals()
			address := token.GetContractAddress()

			// Truncate address if too long
			if len(address) > 40 {
				address = address[:37] + "..."
			}

			fmt.Printf("  %-10s  %2.0f decimals  %s\n",
				color.YellowString(symbol),
				decimals,
				color.HiBlackString(address))
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 90))
	fmt.Printf("\nTotal: %d tokens across %d blockchains\n\n", len(tokens), len(chains))
}
