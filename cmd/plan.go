package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"near-swap/config"
	"near-swap/pkg/client"
	"near-swap/pkg/plan"
)

var (
	// Plan creation flags
	planFromToken      string
	planToToken        string
	planFromChain      string
	planToChain        string
	planTotalAmount    string
	planAmountPerTrade string
	planAmountPerDay   string
	planTriggerPrice   string
	planRecipient      string
	planRefundTo       string
	planDescription    string

	// Plan list flags
	planStatusFilter string

	// Plan stats flags
	statsPage     int
	statsPageSize int
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Manage automated trading plans",
	Long: `Create and manage trading plans that automatically execute swaps when price conditions are met.

Trading plans allow you to set up automated strategies like:
- Sell 10 BTC when price reaches $150k (1 BTC per trade)
- Buy 1000 USDC worth of ETH when ETH drops below $3000
- Dollar-cost average into SOL over multiple trades

Plans are persisted across restarts and track execution history.`,
}

var planCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new trading plan",
	Long: `Create a new automated trading plan with specific conditions.

All plans use auto-deposit, so ensure your auto-deposit is configured.

Examples:
  # Sell 10 BTC at minimum $150k, 1 BTC per trade, max 2 BTC per day
  near-swap plan create sell-btc-high \
    --from BTC --to USDC \
    --from-chain btc --to-chain near \
    --total 10 --per-trade 1 --per-day 2 \
    --when-price above 150000 \
    --recipient your.near \
    --refund-to <btc-address>

  # Buy ETH when price drops below $3000, max $1000 per day
  near-swap plan create buy-eth-dip \
    --from USDC --to ETH \
    --from-chain near --to-chain eth \
    --total 5000 --per-trade 500 --per-day 1000 \
    --when-price below 3000 \
    --recipient 0x123...`,
	Args: cobra.ExactArgs(1),
	Run:  runPlanCreate,
}

var planListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all trading plans",
	Long: `Display all trading plans with their current status.

Examples:
  # List all plans
  near-swap plan list

  # List only active plans
  near-swap plan list --status active

  # List in JSON format
  near-swap plan list --json`,
	Run: runPlanList,
}

var planViewCmd = &cobra.Command{
	Use:   "view <name>",
	Short: "View details of a specific plan",
	Long: `Display detailed information about a trading plan including execution history.

Examples:
  near-swap plan view sell-btc-high
  near-swap plan view sell-btc-high --json`,
	Args: cobra.ExactArgs(1),
	Run:  runPlanView,
}

var planStartCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Start executing a trading plan",
	Long: `Activate a trading plan to begin monitoring prices and executing trades.

The plan will run in the background and automatically execute trades when
price conditions are met.

Examples:
  near-swap plan start sell-btc-high`,
	Args: cobra.ExactArgs(1),
	Run:  runPlanStart,
}

var planStopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop executing a trading plan",
	Long: `Pause a trading plan to stop monitoring and executing trades.

The plan can be restarted later with 'plan start'.

Examples:
  near-swap plan stop sell-btc-high`,
	Args: cobra.ExactArgs(1),
	Run:  runPlanStop,
}

var planDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a trading plan",
	Long: `Permanently remove a trading plan.

Note: Active plans must be stopped before deletion.

Examples:
  near-swap plan delete sell-btc-high`,
	Args: cobra.ExactArgs(1),
	Run:  runPlanDelete,
}

var planHistoryCmd = &cobra.Command{
	Use:   "history <name>",
	Short: "View execution history for a plan",
	Long: `Display the execution history of a trading plan showing all past trades.

Examples:
  near-swap plan history sell-btc-high
  near-swap plan history sell-btc-high --json`,
	Args: cobra.ExactArgs(1),
	Run:  runPlanHistory,
}

var planStatsCmd = &cobra.Command{
	Use:   "stats <name>",
	Short: "View statistics and transaction history for a plan",
	Long: `Display comprehensive statistics for a trading plan including:
- Total number of swaps
- Total amount deposited
- Total amount received
- Recent transaction history with pagination

Examples:
  near-swap plan stats sell-btc-high
  near-swap plan stats sell-btc-high --page 2
  near-swap plan stats sell-btc-high --json`,
	Args: cobra.ExactArgs(1),
	Run:  runPlanStats,
}

var planDaemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run daemon to monitor and execute all active plans",
	Long: `Start a background daemon that monitors and executes all active trading plans.

The daemon will:
- Load all plans with status "active"
- Resume execution from where they stopped (using saved history)
- Monitor prices every 30 seconds
- Check for plan changes every 60 seconds (new/started/stopped plans)
- Execute trades when conditions are met
- Respect daily limits for each plan
- Handle graceful shutdown on Ctrl+C

Dynamic Plan Management:
- Create new plans in another terminal - daemon auto-detects and starts them
- Start/stop plans anytime - daemon adjusts execution automatically
- No need to restart daemon when managing plans

This is useful for running multiple plans simultaneously and ensuring
they continue executing even after system restarts.

Examples:
  # Start daemon in foreground
  near-swap plan daemon

  # Run in background (Linux/Mac)
  nohup near-swap plan daemon > ~/near-swap-daemon.log 2>&1 &

  # In another terminal, manage plans while daemon runs:
  near-swap plan create new-plan ...
  near-swap plan start new-plan      # Daemon auto-detects in <60s
  near-swap plan stop old-plan       # Daemon auto-stops in <60s`,
	Run: runPlanDaemon,
}

func init() {
	rootCmd.AddCommand(planCmd)

	// Add subcommands
	planCmd.AddCommand(planCreateCmd)
	planCmd.AddCommand(planListCmd)
	planCmd.AddCommand(planViewCmd)
	planCmd.AddCommand(planStartCmd)
	planCmd.AddCommand(planStopCmd)
	planCmd.AddCommand(planDeleteCmd)
	planCmd.AddCommand(planHistoryCmd)
	planCmd.AddCommand(planStatsCmd)
	planCmd.AddCommand(planDaemonCmd)

	// Create command flags
	planCreateCmd.Flags().StringVar(&planFromToken, "from", "", "Source token symbol (e.g., BTC, SOL)")
	planCreateCmd.Flags().StringVar(&planToToken, "to", "", "Destination token symbol (e.g., USDC, ETH)")
	planCreateCmd.Flags().StringVar(&planFromChain, "from-chain", "", "Source blockchain")
	planCreateCmd.Flags().StringVar(&planToChain, "to-chain", "", "Destination blockchain")
	planCreateCmd.Flags().StringVar(&planTotalAmount, "total", "", "Total amount to trade")
	planCreateCmd.Flags().StringVar(&planAmountPerTrade, "per-trade", "", "Amount per trade execution")
	planCreateCmd.Flags().StringVar(&planAmountPerDay, "per-day", "", "Maximum amount to trade per day")
	planCreateCmd.Flags().StringVar(&planTriggerPrice, "when-price", "", "Price trigger condition (e.g., 'above 150000', 'below 3000')")
	planCreateCmd.Flags().StringVar(&planRecipient, "recipient", "", "Recipient address for swapped tokens")
	planCreateCmd.Flags().StringVar(&planRefundTo, "refund-to", "", "Refund address (optional, defaults to recipient)")
	planCreateCmd.Flags().StringVar(&planDescription, "description", "", "Plan description (optional)")

	planCreateCmd.MarkFlagRequired("from")
	planCreateCmd.MarkFlagRequired("to")
	planCreateCmd.MarkFlagRequired("from-chain")
	planCreateCmd.MarkFlagRequired("to-chain")
	planCreateCmd.MarkFlagRequired("total")
	planCreateCmd.MarkFlagRequired("per-trade")
	planCreateCmd.MarkFlagRequired("per-day")
	planCreateCmd.MarkFlagRequired("when-price")
	planCreateCmd.MarkFlagRequired("recipient")

	// List command flags
	planListCmd.Flags().StringVar(&planStatusFilter, "status", "", "Filter by status (active, paused, completed, cancelled)")

	// Stats command flags
	planStatsCmd.Flags().IntVar(&statsPage, "page", 1, "Page number for transaction history")
	planStatsCmd.Flags().IntVar(&statsPageSize, "page-size", 10, "Number of transactions per page")
}

func runPlanCreate(cmd *cobra.Command, args []string) {
	planName := args[0]
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Parse price condition
	condition, price, err := parsePriceCondition(planTriggerPrice)
	if err != nil {
		printError(fmt.Errorf("invalid price condition: %w", err))
		os.Exit(1)
	}

	// Set refund address to recipient if not provided
	if planRefundTo == "" {
		planRefundTo = planRecipient
	}

	// Load config to get storage path
	cfg, err := config.Load()
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	// Create plan manager
	manager, err := plan.NewManager(cfg.PlanStoragePath)
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	// Create the plan
	newPlan, err := manager.CreatePlan(
		planName,
		planFromToken, planToToken,
		planFromChain, planToChain,
		planTotalAmount, planAmountPerTrade, planAmountPerDay,
		price, condition,
		planRecipient, planRefundTo,
		planDescription,
	)
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	if jsonOutput {
		output, _ := json.MarshalIndent(newPlan, "", "  ")
		fmt.Println(string(output))
	} else {
		fmt.Println("\n" + strings.Repeat("=", 60))
		color.Green("           TRADING PLAN CREATED SUCCESSFULLY")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Printf("\n  Name:             %s\n", color.CyanString(newPlan.Name))
		fmt.Printf("  Strategy:         Swap %s %s -> %s\n", newPlan.TotalAmount, newPlan.SourceToken, newPlan.DestToken)
		fmt.Printf("  Per Trade:        %s %s\n", newPlan.AmountPerTrade, newPlan.SourceToken)
		fmt.Printf("  Per Day:          %s %s\n", newPlan.AmountPerDay, newPlan.SourceToken)
		fmt.Printf("  Trigger:          When price is %s %s %s/%s\n",
			condition, price, newPlan.DestToken, newPlan.SourceToken)
		fmt.Printf("  Status:           %s\n", color.YellowString(string(newPlan.Status)))
		fmt.Printf("  Auto-deposit:     %s\n", color.GreenString("enabled (required)"))
		if newPlan.Description != "" {
			fmt.Printf("  Description:      %s\n", newPlan.Description)
		}
		fmt.Println("\n" + strings.Repeat("=", 60))
		color.Yellow("\nIMPORTANT: Ensure auto-deposit is configured for %s in your .near-swap.yaml\n", newPlan.SourceChain)
		fmt.Println("\nTo start the plan, run:")
		color.Cyan("  near-swap plan start %s\n", planName)
	}
}

func runPlanList(cmd *cobra.Command, args []string) {
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Load config
	cfg, err := config.Load()
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	// Create plan manager
	manager, err := plan.NewManager(cfg.PlanStoragePath)
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	// Get plans
	var plans []*plan.TradingPlan
	if planStatusFilter != "" {
		status := plan.PlanStatus(planStatusFilter)
		plans = manager.ListPlansByStatus(status)
	} else {
		plans = manager.ListPlans()
	}

	if jsonOutput {
		summaries := make([]*plan.PlanSummary, len(plans))
		for i, p := range plans {
			summaries[i] = p.ToSummary()
		}
		output, _ := json.MarshalIndent(summaries, "", "  ")
		fmt.Println(string(output))
		return
	}

	if len(plans) == 0 {
		color.Yellow("No trading plans found.\n")
		fmt.Println("\nCreate a new plan with:")
		color.Cyan("  near-swap plan create <name> --from <token> --to <token> ...\n")
		return
	}

	fmt.Println("\n" + strings.Repeat("=", 120))
	color.Green("                                              TRADING PLANS")
	fmt.Println(strings.Repeat("=", 120))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "\nNAME\tSTRATEGY\tPROGRESS\tTRIGGER\tSTATUS\tEXECUTIONS")
	fmt.Fprintln(w, strings.Repeat("-", 120))

	for _, p := range plans {
		strategy := fmt.Sprintf("%s -> %s", p.SourceToken, p.DestToken)
		progress := fmt.Sprintf("%s / %s", p.TotalExecuted, p.TotalAmount)
		trigger := fmt.Sprintf("%s %s", p.PriceCondition, p.TriggerPrice)

		statusColor := getStatusColor(p.Status)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\n",
			p.Name, strategy, progress, trigger, statusColor, p.ExecutionCount)
	}

	w.Flush()
	fmt.Println("\n" + strings.Repeat("=", 120) + "\n")
}

func runPlanView(cmd *cobra.Command, args []string) {
	planName := args[0]
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Load config
	cfg, err := config.Load()
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	// Create plan manager
	manager, err := plan.NewManager(cfg.PlanStoragePath)
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	// Get plan
	p, err := manager.GetPlan(planName)
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	if jsonOutput {
		output, _ := json.MarshalIndent(p, "", "  ")
		fmt.Println(string(output))
		return
	}

	// Display plan details
	fmt.Println("\n" + strings.Repeat("=", 70))
	color.Green("                        TRADING PLAN DETAILS")
	fmt.Println(strings.Repeat("=", 70))

	fmt.Printf("\n  Name:              %s\n", color.CyanString(p.Name))
	if p.Description != "" {
		fmt.Printf("  Description:       %s\n", p.Description)
	}
	fmt.Printf("  Status:            %s\n", getStatusColor(p.Status))
	fmt.Printf("  Created:           %s\n", p.Created.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Last Updated:      %s\n", p.LastUpdated.Format("2006-01-02 15:04:05"))

	fmt.Printf("\n  Trading Strategy:\n")
	fmt.Printf("    From:            %s %s (on %s)\n", p.TotalAmount, p.SourceToken, p.SourceChain)
	fmt.Printf("    To:              %s (on %s)\n", p.DestToken, p.DestChain)
	fmt.Printf("    Per Trade:       %s %s\n", p.AmountPerTrade, p.SourceToken)
	fmt.Printf("    Per Day:         %s %s\n", p.AmountPerDay, p.SourceToken)
	fmt.Printf("    Trigger:         When price %s %s %s/%s\n",
		p.PriceCondition, p.TriggerPrice, p.DestToken, p.SourceToken)

	fmt.Printf("\n  Addresses:\n")
	fmt.Printf("    Recipient:       %s\n", p.RecipientAddr)
	fmt.Printf("    Refund:          %s\n", p.RefundAddr)

	fmt.Printf("\n  Execution Progress:\n")
	fmt.Printf("    Total Amount:    %s %s\n", p.TotalAmount, p.SourceToken)
	fmt.Printf("    Executed:        %s %s\n", p.TotalExecuted, p.SourceToken)
	fmt.Printf("    Remaining:       %s %s\n", p.RemainingAmount, p.SourceToken)
	fmt.Printf("    Today Executed:  %s %s (limit: %s %s)\n", p.TodayExecuted, p.SourceToken, p.AmountPerDay, p.SourceToken)
	fmt.Printf("    Executions:      %d\n", p.ExecutionCount)
	fmt.Printf("    Auto-deposit:    %s\n", color.GreenString("enabled (required)"))

	fmt.Println("\n" + strings.Repeat("=", 70) + "\n")

	// Show recent executions
	if len(p.ExecutionHistory) > 0 {
		fmt.Println(strings.Repeat("=", 70))
		color.Green("                      RECENT EXECUTIONS")
		fmt.Println(strings.Repeat("=", 70))

		// Show last 5 executions
		start := 0
		if len(p.ExecutionHistory) > 5 {
			start = len(p.ExecutionHistory) - 5
		}

		for i := len(p.ExecutionHistory) - 1; i >= start; i-- {
			exec := p.ExecutionHistory[i]
			fmt.Printf("\n  [%s] %s\n", exec.Timestamp.Format("2006-01-02 15:04:05"), getExecutionStatusColor(exec.Status))
			fmt.Printf("    Amount In:       %s %s\n", exec.Amount, p.SourceToken)
			fmt.Printf("    Price:           %s %s/%s\n", exec.ActualPrice, p.DestToken, p.SourceToken)

			// Show actual output if available, otherwise estimated
			if exec.ActualOutput != "" {
				fmt.Printf("    Amount Out:      %s %s\n", color.GreenString(exec.ActualOutput), p.DestToken)
			} else if exec.EstimatedOutput != "" {
				fmt.Printf("    Expected Output: %s %s\n", exec.EstimatedOutput, p.DestToken)
			}

			if exec.DepositAddress != "" {
				fmt.Printf("    Deposit Addr:    %s\n", exec.DepositAddress)
			}
			if exec.TxHash != "" {
				fmt.Printf("    Deposit TX:      %s\n", color.CyanString(exec.TxHash))
			}
			if exec.DestinationTxHash != "" {
				fmt.Printf("    Dest TX:         %s\n", color.CyanString(exec.DestinationTxHash))
			}
			if exec.SwapStatus != "" {
				fmt.Printf("    Swap Status:     %s\n", exec.SwapStatus)
			}
			if exec.CompletionTime != nil {
				fmt.Printf("    Completed At:    %s\n", exec.CompletionTime.Format("2006-01-02 15:04:05"))
			}
			if exec.ErrorMessage != "" {
				fmt.Printf("    Error:           %s\n", color.RedString(exec.ErrorMessage))
			}
		}

		fmt.Println("\n" + strings.Repeat("=", 70) + "\n")
	}
}

func runPlanStart(cmd *cobra.Command, args []string) {
	planName := args[0]

	// Load config
	cfg, err := config.Load()
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	// Create plan manager
	manager, err := plan.NewManager(cfg.PlanStoragePath)
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	// Start the plan
	if err := manager.StartPlan(planName); err != nil {
		printError(err)
		os.Exit(1)
	}

	color.Green("\n✓ Trading plan '%s' has been activated!\n", planName)
	fmt.Println("\nThe plan will now monitor prices and execute trades automatically.")
	fmt.Println("To stop the plan, run:")
	color.Cyan("  near-swap plan stop %s\n", planName)
}

func runPlanStop(cmd *cobra.Command, args []string) {
	planName := args[0]

	// Load config
	cfg, err := config.Load()
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	// Create plan manager
	manager, err := plan.NewManager(cfg.PlanStoragePath)
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	// Stop the plan
	if err := manager.StopPlan(planName); err != nil {
		printError(err)
		os.Exit(1)
	}

	color.Green("\n✓ Trading plan '%s' has been stopped.\n", planName)
	fmt.Println("\nTo restart the plan, run:")
	color.Cyan("  near-swap plan start %s\n", planName)
}

func runPlanDelete(cmd *cobra.Command, args []string) {
	planName := args[0]

	// Load config
	cfg, err := config.Load()
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	// Create plan manager
	manager, err := plan.NewManager(cfg.PlanStoragePath)
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	// Delete the plan
	if err := manager.DeletePlan(planName); err != nil {
		printError(err)
		os.Exit(1)
	}

	color.Green("\n✓ Trading plan '%s' has been deleted.\n", planName)
}

func runPlanHistory(cmd *cobra.Command, args []string) {
	planName := args[0]
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Load config
	cfg, err := config.Load()
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	// Create plan manager
	manager, err := plan.NewManager(cfg.PlanStoragePath)
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	// Get execution history
	history, err := manager.GetExecutionHistory(planName)
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	if jsonOutput {
		output, _ := json.MarshalIndent(history, "", "  ")
		fmt.Println(string(output))
		return
	}

	if len(history) == 0 {
		color.Yellow("\nNo execution history found for plan '%s'.\n", planName)
		return
	}

	// Get plan details for token symbols
	p, _ := manager.GetPlan(planName)

	// Calculate totals
	var totalSold, totalReceived float64
	completedCount := 0

	for _, exec := range history {
		// Sum all amounts sold
		if amount, err := strconv.ParseFloat(exec.Amount, 64); err == nil {
			totalSold += amount
		}

		// Sum amounts received (only for completed with actual output)
		if exec.Status == plan.ExecutionCompleted && exec.ActualOutput != "" {
			if amount, err := strconv.ParseFloat(exec.ActualOutput, 64); err == nil {
				totalReceived += amount
				completedCount++
			}
		}
	}

	// Display header with summary
	fmt.Println("\n" + strings.Repeat("=", 120))
	color.Green("                                EXECUTION HISTORY: %s", planName)
	fmt.Println(strings.Repeat("=", 120))

	fmt.Printf("\n  Total Transactions:  %s\n", color.CyanString("%d", len(history)))
	fmt.Printf("  Completed:           %s\n", color.GreenString("%d", completedCount))
	fmt.Printf("  Total Sold:          %s %s\n", color.YellowString("%.8f", totalSold), p.SourceToken)
	if totalReceived > 0 {
		fmt.Printf("  Total Received:      %s %s\n", color.GreenString("%.8f", totalReceived), p.DestToken)
		avgPrice := totalReceived / totalSold
		fmt.Printf("  Average Price:       %s %s/%s\n", color.CyanString("%.8f", avgPrice), p.DestToken, p.SourceToken)
	}
	fmt.Println()

	// Display transaction table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TIMESTAMP\tAMOUNT IN\tAMOUNT OUT\tPRICE\tSTATUS\tDEPOSIT TX\tDEST TX")
	fmt.Fprintln(w, strings.Repeat("-", 120))

	for _, exec := range history {
		timestamp := exec.Timestamp.Format("2006-01-02 15:04")
		amountIn := fmt.Sprintf("%s %s", exec.Amount, p.SourceToken)

		// Show actual output if available, otherwise estimated
		amountOut := ""
		if exec.ActualOutput != "" {
			amountOut = fmt.Sprintf("%s %s", exec.ActualOutput, p.DestToken)
		} else if exec.EstimatedOutput != "" {
			amountOut = fmt.Sprintf("~%s %s", exec.EstimatedOutput, p.DestToken)
		}

		price := exec.ActualPrice
		status := getExecutionStatusColor(exec.Status)
		depositTx := truncateString(exec.TxHash, 12)
		destTx := truncateString(exec.DestinationTxHash, 12)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			timestamp, amountIn, amountOut, price, status, depositTx, destTx)
	}

	w.Flush()
	fmt.Println("\n" + strings.Repeat("=", 120) + "\n")
}

// Helper functions

func parsePriceCondition(input string) (plan.PriceCondition, string, error) {
	parts := strings.Fields(input)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("price condition must be in format '<condition> <price>' (e.g., 'above 150000')")
	}

	conditionStr := strings.ToLower(parts[0])
	price := parts[1]

	var condition plan.PriceCondition
	switch conditionStr {
	case "above", ">":
		condition = plan.PriceAbove
	case "below", "<":
		condition = plan.PriceBelow
	case "at", "=", "==":
		condition = plan.PriceAt
	default:
		return "", "", fmt.Errorf("invalid condition '%s', must be 'above', 'below', or 'at'", conditionStr)
	}

	return condition, price, nil
}

func getStatusColor(status plan.PlanStatus) string {
	switch status {
	case plan.StatusActive:
		return color.GreenString(string(status))
	case plan.StatusPaused:
		return color.YellowString(string(status))
	case plan.StatusCompleted:
		return color.BlueString(string(status))
	case plan.StatusCancelled:
		return color.RedString(string(status))
	default:
		return string(status)
	}
}

func getExecutionStatusColor(status plan.ExecutionStatus) string {
	switch status {
	case plan.ExecutionCompleted:
		return color.GreenString(string(status))
	case plan.ExecutionDeposited:
		return color.CyanString(string(status))
	case plan.ExecutionPending:
		return color.YellowString(string(status))
	case plan.ExecutionFailed:
		return color.RedString(string(status))
	default:
		return string(status)
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func runPlanStats(cmd *cobra.Command, args []string) {
	planName := args[0]
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Load config
	cfg, err := config.Load()
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	// Create plan manager
	manager, err := plan.NewManager(cfg.PlanStoragePath)
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	// Get plan
	p, err := manager.GetPlan(planName)
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	history := p.ExecutionHistory

	if jsonOutput {
		statsData := calculateStats(p, history)
		output, _ := json.MarshalIndent(statsData, "", "  ")
		fmt.Println(string(output))
		return
	}

	// Calculate statistics
	totalSwaps := len(history)
	completedSwaps := 0
	var totalDeposited, totalReceived float64

	for _, exec := range history {
		if exec.Status == plan.ExecutionCompleted {
			completedSwaps++
		}

		// Sum deposited amounts
		if amount, err := strconv.ParseFloat(exec.Amount, 64); err == nil {
			totalDeposited += amount
		}

		// Sum received amounts (only for completed swaps with actual output)
		if exec.ActualOutput != "" {
			if amount, err := strconv.ParseFloat(exec.ActualOutput, 64); err == nil {
				totalReceived += amount
			}
		}
	}

	// Display statistics header
	fmt.Println("\n" + strings.Repeat("=", 100))
	color.Green("                          PLAN STATISTICS: %s", planName)
	fmt.Println(strings.Repeat("=", 100))

	fmt.Printf("\n  Plan Status:        %s\n", getStatusColor(p.Status))
	fmt.Printf("  Trading Pair:       %s -> %s\n", p.SourceToken, p.DestToken)
	fmt.Println()
	fmt.Printf("  Total Swaps:        %s\n", color.CyanString("%d", totalSwaps))
	fmt.Printf("  Completed Swaps:    %s\n", color.GreenString("%d", completedSwaps))
	fmt.Printf("  Pending Swaps:      %s\n", color.YellowString("%d", totalSwaps-completedSwaps))
	fmt.Println()
	fmt.Printf("  Total Deposited:    %s %s\n", color.CyanString("%.8f", totalDeposited), p.SourceToken)
	if totalReceived > 0 {
		fmt.Printf("  Total Received:     %s %s\n", color.GreenString("%.8f", totalReceived), p.DestToken)
	}
	fmt.Printf("  Remaining:          %s %s\n", p.RemainingAmount, p.SourceToken)

	if totalSwaps == 0 {
		fmt.Println("\n" + strings.Repeat("=", 100))
		color.Yellow("\nNo transactions found for this plan.\n")
		return
	}

	// Pagination
	totalPages := (totalSwaps + statsPageSize - 1) / statsPageSize
	if statsPage < 1 {
		statsPage = 1
	}
	if statsPage > totalPages {
		statsPage = totalPages
	}

	startIdx := (statsPage - 1) * statsPageSize
	endIdx := startIdx + statsPageSize
	if endIdx > totalSwaps {
		endIdx = totalSwaps
	}

	// Display transactions table
	fmt.Println("\n" + strings.Repeat("=", 100))
	color.Green("                    RECENT TRANSACTIONS (Page %d of %d)", statsPage, totalPages)
	fmt.Println(strings.Repeat("=", 100))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "\nTIMESTAMP\tAMOUNT IN\tAMOUNT OUT\tPRICE\tSTATUS\tTX HASH\tDEST TX")
	fmt.Fprintln(w, strings.Repeat("-", 100))

	// Show transactions in reverse order (most recent first)
	for i := len(history) - 1 - startIdx; i >= 0 && i >= len(history)-endIdx; i-- {
		exec := history[i]
		timestamp := exec.Timestamp.Format("2006-01-02 15:04")
		amountIn := fmt.Sprintf("%s %s", exec.Amount, p.SourceToken)

		// Show actual output if available, otherwise estimated
		amountOut := ""
		if exec.ActualOutput != "" {
			amountOut = fmt.Sprintf("%s %s", exec.ActualOutput, p.DestToken)
		} else if exec.EstimatedOutput != "" {
			amountOut = fmt.Sprintf("~%s %s", exec.EstimatedOutput, p.DestToken)
		}

		price := exec.ActualPrice
		status := getExecutionStatusColor(exec.Status)
		txHash := truncateString(exec.TxHash, 12)
		destTx := truncateString(exec.DestinationTxHash, 12)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			timestamp, amountIn, amountOut, price, status, txHash, destTx)
	}

	w.Flush()
	fmt.Println("\n" + strings.Repeat("=", 100))

	// Pagination info
	if totalPages > 1 {
		fmt.Printf("\nShowing transactions %d-%d of %d total\n", startIdx+1, endIdx, totalSwaps)
		if statsPage < totalPages {
			color.Cyan("To see more transactions, use: near-swap plan stats %s --page %d\n", planName, statsPage+1)
		}
		if statsPage > 1 {
			color.Cyan("To see previous transactions, use: near-swap plan stats %s --page %d\n", planName, statsPage-1)
		}
	}
	fmt.Println()
}

func calculateStats(p *plan.TradingPlan, history []plan.Execution) map[string]interface{} {
	totalSwaps := len(history)
	completedSwaps := 0
	var totalDeposited, totalReceived float64

	transactions := make([]map[string]interface{}, 0, len(history))

	for _, exec := range history {
		if exec.Status == plan.ExecutionCompleted {
			completedSwaps++
		}

		if amount, err := strconv.ParseFloat(exec.Amount, 64); err == nil {
			totalDeposited += amount
		}

		if exec.ActualOutput != "" {
			if amount, err := strconv.ParseFloat(exec.ActualOutput, 64); err == nil {
				totalReceived += amount
			}
		}

		txData := map[string]interface{}{
			"id":                  exec.ID,
			"timestamp":           exec.Timestamp,
			"amount_in":           exec.Amount,
			"amount_out":          exec.ActualOutput,
			"estimated_output":    exec.EstimatedOutput,
			"price":               exec.ActualPrice,
			"status":              exec.Status,
			"deposit_address":     exec.DepositAddress,
			"tx_hash":             exec.TxHash,
			"destination_tx_hash": exec.DestinationTxHash,
			"swap_status":         exec.SwapStatus,
		}
		transactions = append(transactions, txData)
	}

	return map[string]interface{}{
		"plan_name":        p.Name,
		"status":           p.Status,
		"source_token":     p.SourceToken,
		"dest_token":       p.DestToken,
		"total_swaps":      totalSwaps,
		"completed_swaps":  completedSwaps,
		"pending_swaps":    totalSwaps - completedSwaps,
		"total_deposited":  fmt.Sprintf("%.8f", totalDeposited),
		"total_received":   fmt.Sprintf("%.8f", totalReceived),
		"remaining_amount": p.RemainingAmount,
		"transactions":     transactions,
	}
}

func runPlanDaemon(cmd *cobra.Command, args []string) {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	// Create plan manager
	manager, err := plan.NewManager(cfg.PlanStoragePath)
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	// Get all active plans
	activePlans := manager.GetActivePlans()

	if len(activePlans) == 0 {
		color.Yellow("\nNo active plans found.\n")
		fmt.Println("\nTo create and start a plan:")
		color.Cyan("  1. Create: near-swap plan create <name> ...")
		color.Cyan("  2. Start:  near-swap plan start <name>")
		color.Cyan("  3. Run:    near-swap plan daemon\n")
		return
	}

	fmt.Println("\n" + strings.Repeat("=", 70))
	color.Green("              NEAR-SWAP TRADING PLAN DAEMON")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("\nLoading %d active plan(s)...\n\n", len(activePlans))

	// Display loaded plans with their current state
	for _, p := range activePlans {
		fmt.Printf("  [%s] %s\n", color.GreenString("ACTIVE"), color.CyanString(p.Name))
		fmt.Printf("      Strategy:  %s %s -> %s\n", p.TotalAmount, p.SourceToken, p.DestToken)
		fmt.Printf("      Progress:  %s / %s executed\n", p.TotalExecuted, p.TotalAmount)
		fmt.Printf("      Today:     %s / %s (daily limit)\n", p.TodayExecuted, p.AmountPerDay)
		fmt.Printf("      Trigger:   Price %s %s %s/%s\n", p.PriceCondition, p.TriggerPrice, p.DestToken, p.SourceToken)
		if p.ExecutionCount > 0 {
			fmt.Printf("      History:   %d execution(s)\n", p.ExecutionCount)
		}
		fmt.Println()
	}

	// Check auto-deposit configuration
	if !cfg.AutoDeposit.Enabled {
		color.Red("\n⚠ WARNING: Auto-deposit is not enabled in your configuration!")
		color.Yellow("Plans will not be able to execute trades automatically.")
		color.Yellow("Please configure auto-deposit in your .near-swap.yaml file.\n")
	}

	fmt.Println(strings.Repeat("=", 70))
	color.Green("\nStarting executor...")
	color.Cyan("• Monitoring prices every 30 seconds")
	color.Cyan("• Checking for plan changes every 60 seconds")
	color.Magenta("• You can create/start/stop plans in another terminal")
	color.Yellow("• Press Ctrl+C to stop gracefully\n")
	fmt.Println(strings.Repeat("=", 70) + "\n")

	// Create API client
	apiClient := client.NewOneClickClient(cfg.JWTToken)

	// Create executor
	executor := plan.NewExecutor(manager, apiClient, cfg)

	// Start executor
	if err := executor.Start(); err != nil {
		printError(err)
		os.Exit(1)
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for interrupt signal
	<-sigChan

	fmt.Println("\n" + strings.Repeat("=", 70))
	color.Yellow("\nReceived shutdown signal. Stopping executor gracefully...")

	// Stop executor
	executor.Stop()

	// Save final state
	fmt.Println("Saving plan states...")

	color.Green("\n✓ Daemon stopped successfully.")
	fmt.Println("\nAll plan states have been saved. You can restart with:")
	color.Cyan("  near-swap plan daemon\n")
	fmt.Println(strings.Repeat("=", 70) + "\n")
}
