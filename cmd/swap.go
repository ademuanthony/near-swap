package cmd

import (
	"bufio"
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
	"near-swap/pkg/deposit"
	"near-swap/pkg/parser"
	"near-swap/pkg/types"
)

var (
	fromChain     string
	toChain       string
	recipientAddr string
	refundAddr    string
	noConfirm     bool
	autoDeposit   bool
)

var swapCmd = &cobra.Command{
	Use:   "swap <amount> <source-token> to <dest-token>",
	Short: "Perform a cross-chain token swap",
	Long: `Swap tokens across different blockchains using NEAR Intents 1Click API.

IMPORTANT:
  - You MUST specify --recipient (where you'll receive tokens)
  - You SHOULD specify --refund-to for cross-chain swaps (where refunds go if swap fails)
  - Both addresses must be valid for their respective blockchains

Examples:
  # Cross-chain swap
  near-swap swap 1 SOL to USDC --from-chain sol --to-chain near --recipient your.near --refund-to <solana-addr>

  # Same-chain swap
  near-swap swap 0.5 ETH to USDC --from-chain eth --to-chain eth --recipient 0x123... --refund-to 0x123...

  # With auto-deposit (Bitcoin example)
  near-swap swap 0.01 BTC to USDC --from-chain btc --to-chain near --recipient your.near --refund-to <btc-addr> --auto-deposit

  # Skip all confirmations
  near-swap swap 1 SOL to USDC --from-chain sol --to-chain near --recipient your.near --refund-to <sol-addr> --yes`,
	Args: cobra.MinimumNArgs(1),
	Run:  runSwap,
}

func init() {
	rootCmd.AddCommand(swapCmd)

	swapCmd.Flags().StringVar(&fromChain, "from-chain", "", "Source blockchain (optional)")
	swapCmd.Flags().StringVar(&toChain, "to-chain", "", "Destination blockchain (optional)")
	swapCmd.Flags().StringVar(&recipientAddr, "recipient", "", "Recipient address (REQUIRED - where you'll receive tokens)")
	swapCmd.Flags().StringVar(&refundAddr, "refund-to", "", "Refund address on source chain (optional - where refunds go if swap fails)")
	swapCmd.Flags().BoolVarP(&noConfirm, "yes", "y", false, "Skip confirmation prompt")
	swapCmd.Flags().BoolVar(&autoDeposit, "auto-deposit", false, "Automatically send deposit (requires configuration)")
}

func runSwap(cmd *cobra.Command, args []string) {
	// Parse the command
	commandStr := strings.Join(args, " ")
	swapReq, err := parser.ParseSwapCommand(commandStr)
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	// Set chain, recipient, and refund address if provided via flags
	if fromChain != "" {
		swapReq.SourceChain = fromChain
	}
	if toChain != "" {
		swapReq.DestChain = toChain
	}
	if recipientAddr != "" {
		swapReq.RecipientAddr = recipientAddr
	}
	if refundAddr != "" {
		swapReq.RefundAddr = refundAddr
	}

	verbose, _ := cmd.Flags().GetBool("verbose")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	// Create client
	apiClient := client.NewOneClickClient(cfg.JWTToken)

	// Get quote with spinner
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !jsonOutput {
		s.Suffix = " Fetching quote..."
		s.Start()
	}

	if verbose {
		fmt.Printf("\nDebug: Fetching tokens for SOL and USDC...\n")
	}

	quote, err := apiClient.GetQuote(swapReq)
	if !jsonOutput {
		s.Stop()
	}

	if err != nil {
		if verbose {
			fmt.Printf("\nDebug: Error getting quote: %v\n", err)
			fmt.Printf("Debug: This might be due to:\n")
			fmt.Printf("  1. Invalid JWT token\n")
			fmt.Printf("  2. Token not found (try: near-swap list-tokens)\n")
			fmt.Printf("  3. API version mismatch\n")
		}
		printError(err)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("\nQuote received:\n")
		quoteJSON, _ := json.MarshalIndent(quote, "", "  ")
		fmt.Println(string(quoteJSON))
	}

	// Get the quote details
	quoteDetails := quote.GetQuote()

	// Display quote
	if jsonOutput {
		output := map[string]interface{}{
			"deposit_address":   quoteDetails.GetDepositAddress(),
			"source_amount":     swapReq.Amount,
			"source_token":      swapReq.SourceToken,
			"dest_amount":       quoteDetails.GetAmountOutFormatted(),
			"dest_token":        swapReq.DestToken,
			"time_estimate_sec": quoteDetails.GetTimeEstimate(),
			"status":            "quote_generated",
		}
		jsonData, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(jsonData))
	} else {
		displayQuote(&quoteDetails, swapReq)
	}

	// Ask for confirmation
	if !noConfirm && !jsonOutput {
		if !confirmSwap() {
			fmt.Println("\nSwap cancelled.")
			os.Exit(0)
		}
	}

	// Display deposit instructions
	if !jsonOutput {
		displayDepositInstructions(&quoteDetails, swapReq)
	}

	// Handle auto-deposit if enabled
	if autoDeposit || cfg.AutoDeposit.Enabled {
		if err := handleAutoDeposit(cfg, swapReq, &quoteDetails, verbose, noConfirm); err != nil {
			color.Red("\nAuto-deposit failed: %v", err)
			color.Yellow("Please send the deposit manually to: %s\n", quoteDetails.GetDepositAddress())
		}
	}

	// Monitor swap status (optional, in background)
	if !jsonOutput {
		fmt.Println("\nYou can monitor the swap status using:")
		color.Cyan("  near-swap status %s\n", quoteDetails.GetDepositAddress())
	}
}

func handleAutoDeposit(cfg *config.Config, swapReq *types.SwapRequest, quoteDetails *oneclick.Quote, verbose bool, skipConfirm bool) error {
	depositMgr := deposit.NewManager(cfg.AutoDeposit)

	// Check if auto-deposit is supported for the source chain
	if !depositMgr.IsEnabledForChain(swapReq.SourceChain) {
		return fmt.Errorf("auto-deposit not enabled for chain: %s", swapReq.SourceChain)
	}

	depositAddress := quoteDetails.GetDepositAddress()
	amount := swapReq.Amount

	color.Yellow("\n🔄 Initiating auto-deposit...\n")
	fmt.Printf("  Chain:   %s\n", swapReq.SourceChain)
	fmt.Printf("  Amount:  %s %s\n", amount, swapReq.SourceToken)
	fmt.Printf("  To:      %s\n", depositAddress)

	// Confirm auto-deposit (skip if --yes flag is set or auto_confirm is enabled in config)
	if !skipConfirm && !cfg.AutoConfirm {
		if !confirmAutoDeposit() {
			return fmt.Errorf("auto-deposit cancelled by user")
		}
	}

	// Send the deposit
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Sending deposit..."
	s.Start()

	txid, err := depositMgr.SendDeposit(swapReq.SourceChain, depositAddress, amount)
	s.Stop()

	if err != nil {
		return err
	}

	color.Green("\n✓ Deposit sent successfully!")
	fmt.Printf("  Transaction ID: %s\n", color.CyanString(txid))

	if verbose {
		fmt.Printf("\nDeposit transaction details:\n")
		fmt.Printf("  Chain:      %s\n", swapReq.SourceChain)
		fmt.Printf("  Amount:     %s %s\n", amount, swapReq.SourceToken)
		fmt.Printf("  To:         %s\n", depositAddress)
		fmt.Printf("  Tx Hash:    %s\n", txid)
	}

	return nil
}

func confirmAutoDeposit() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nProceed with auto-deposit? (y/N): ")

	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func displayQuote(quote *oneclick.Quote, swapReq *types.SwapRequest) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	color.Green("                     SWAP QUOTE")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Printf("\n  Deposit Address:   %s\n", color.CyanString(quote.GetDepositAddress()))
	fmt.Printf("  From:              %s %s\n", quote.GetAmountInFormatted(), color.YellowString(swapReq.SourceToken))
	fmt.Printf("  To:                ~%s %s\n", quote.GetAmountOutFormatted(), color.YellowString(swapReq.DestToken))
	fmt.Printf("  Estimated Time:    %.0f seconds\n", quote.GetTimeEstimate())

	if swapReq.SourceChain != "" {
		fmt.Printf("  Source Chain:      %s\n", swapReq.SourceChain)
	}
	if swapReq.DestChain != "" {
		fmt.Printf("  Destination Chain: %s\n", swapReq.DestChain)
	}

	fmt.Println("\n" + strings.Repeat("=", 60) + "\n")
}

func displayDepositInstructions(quote *oneclick.Quote, swapReq *types.SwapRequest) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	color.Yellow("                 DEPOSIT INSTRUCTIONS")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("\nTo complete the swap, send %s %s to:\n\n", quote.GetAmountInFormatted(), swapReq.SourceToken)
	color.Cyan("  %s\n", quote.GetDepositAddress())

	if quote.HasDepositMemo() {
		fmt.Printf("\nMemo (REQUIRED): %s\n", color.MagentaString(quote.GetDepositMemo()))
	}

	fmt.Println("\n" + strings.Repeat("=", 60) + "\n")
}

func confirmSwap() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nProceed with swap? (y/N): ")

	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
