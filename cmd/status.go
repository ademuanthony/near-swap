package cmd

import (
	"encoding/json"
	"fmt"
	"os"
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
	watchStatus bool
	watchInterval int
)

var statusCmd = &cobra.Command{
	Use:   "status <deposit-address>",
	Short: "Check the status of a swap",
	Long: `Check the execution status of a cross-chain swap by its deposit address.

Examples:
  near-swap status 0x1234...abcd
  near-swap status 0x1234...abcd --watch
  near-swap status 0x1234...abcd --watch --interval 10`,
	Args: cobra.ExactArgs(1),
	Run:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)

	statusCmd.Flags().BoolVarP(&watchStatus, "watch", "w", false, "Watch status updates continuously")
	statusCmd.Flags().IntVar(&watchInterval, "interval", 5, "Polling interval in seconds (when watching)")
}

func runStatus(cmd *cobra.Command, args []string) {
	depositAddress := args[0]
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	// Create client
	apiClient := client.NewOneClickClient(cfg.JWTToken)

	if watchStatus {
		watchSwapStatus(apiClient, depositAddress, jsonOutput)
	} else {
		checkSwapStatus(apiClient, depositAddress, jsonOutput)
	}
}

func checkSwapStatus(apiClient *client.OneClickClient, depositAddress string, jsonOutput bool) {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !jsonOutput {
		s.Suffix = " Checking swap status..."
		s.Start()
	}

	status, err := apiClient.GetSwapStatus(depositAddress)
	if !jsonOutput {
		s.Stop()
	}

	if err != nil {
		printError(err)
		os.Exit(1)
	}

	if jsonOutput {
		jsonData, _ := json.MarshalIndent(status, "", "  ")
		fmt.Println(string(jsonData))
	} else {
		displayStatus(status, depositAddress)
	}
}

func watchSwapStatus(apiClient *client.OneClickClient, depositAddress string, jsonOutput bool) {
	if jsonOutput {
		fmt.Println(`{"error": "watch mode not supported with JSON output"}`)
		os.Exit(1)
	}

	fmt.Printf("\nWatching swap status (Deposit Address: %s)\n", color.CyanString(depositAddress))
	fmt.Printf("Checking every %d seconds. Press Ctrl+C to stop.\n\n", watchInterval)

	ticker := time.NewTicker(time.Duration(watchInterval) * time.Second)
	defer ticker.Stop()

	// Check immediately first
	checkAndDisplayStatus(apiClient, depositAddress)

	// Then check periodically
	for range ticker.C {
		checkAndDisplayStatus(apiClient, depositAddress)
	}
}

func checkAndDisplayStatus(apiClient *client.OneClickClient, depositAddress string) {
	status, err := apiClient.GetSwapStatus(depositAddress)
	if err != nil {
		color.Red("Error: %v", err)
		return
	}

	displayStatus(status, depositAddress)
}

func displayStatus(status *oneclick.GetExecutionStatusResponse, depositAddress string) {
	fmt.Println("\n" + strings.Repeat("=", 70))
	color.Green("                        SWAP STATUS")
	fmt.Println(strings.Repeat("=", 70))

	fmt.Printf("\n  Deposit Address: %s\n", color.CyanString(depositAddress))
	fmt.Printf("  Status:          %s\n", getColoredStatus(status.GetStatus()))
	fmt.Printf("  Last Updated:    %s\n", status.GetUpdatedAt().Format("2006-01-02 15:04:05"))

	// Display swap details if available
	swapDetails := status.GetSwapDetails()

	// Display origin chain transactions (deposits)
	originTxs := swapDetails.GetOriginChainTxHashes()
	if len(originTxs) > 0 {
		for _, tx := range originTxs {
			hash := tx.GetHash()
			if hash != "" {
				fmt.Printf("  Deposit Tx:      %s\n", color.HiBlackString(hash))
			}
		}
	}

	// Display destination chain transactions (withdrawals)
	destTxs := swapDetails.GetDestinationChainTxHashes()
	if len(destTxs) > 0 {
		for _, tx := range destTxs {
			hash := tx.GetHash()
			if hash != "" {
				fmt.Printf("  Withdrawal Tx:   %s\n", color.HiBlackString(hash))
			}
		}
	}

	// Display amounts if available
	if swapDetails.HasAmountInFormatted() {
		fmt.Printf("  Amount In:       %s\n", swapDetails.GetAmountInFormatted())
	}
	if swapDetails.HasAmountOutFormatted() {
		fmt.Printf("  Amount Out:      %s\n", swapDetails.GetAmountOutFormatted())
	}

	fmt.Println("\n" + strings.Repeat("=", 70) + "\n")
}

func getColoredStatus(status string) string {
	status = strings.ToUpper(status)

	switch status {
	case "SUCCESS", "COMPLETED":
		return color.GreenString(status)
	case "PENDING_DEPOSIT", "PENDING", "PROCESSING":
		return color.YellowString(status)
	case "FAILED", "REFUNDED":
		return color.RedString(status)
	case "INCOMPLETE_DEPOSIT":
		return color.MagentaString(status)
	default:
		return status
	}
}
