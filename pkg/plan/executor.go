package plan

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	oneclick "github.com/defuse-protocol/one-click-sdk-go"
	"near-swap/config"
	"near-swap/pkg/client"
	"near-swap/pkg/deposit"
	"near-swap/pkg/types"
)

const (
	DefaultCheckInterval     = 30 * time.Second // Check prices every 30 seconds
	MinCheckInterval         = 10 * time.Second // Minimum interval to avoid rate limiting
	PlanReloadInterval       = 60 * time.Second // Check for plan changes every 60 seconds
	SwapVerificationInterval = 45 * time.Second // Check swap status every 45 seconds
)

// Executor manages the execution of trading plans
type Executor struct {
	manager        *Manager
	pricer         *Pricer
	apiClient      *client.OneClickClient
	config         *config.Config
	checkInterval  time.Duration
	running        bool
	stopChan       chan struct{}
	mu             sync.RWMutex
	activePlans    map[string]*planExecutor
}

// planExecutor manages execution for a single plan
type planExecutor struct {
	plan      *TradingPlan
	stopChan  chan struct{}
	running   bool
}

// NewExecutor creates a new executor instance
func NewExecutor(manager *Manager, apiClient *client.OneClickClient, cfg *config.Config) *Executor {
	return &Executor{
		manager:       manager,
		pricer:        NewPricer(apiClient),
		apiClient:     apiClient,
		config:        cfg,
		checkInterval: DefaultCheckInterval,
		stopChan:      make(chan struct{}),
		activePlans:   make(map[string]*planExecutor),
	}
}

// SetCheckInterval sets the price check interval
func (e *Executor) SetCheckInterval(interval time.Duration) {
	if interval < MinCheckInterval {
		interval = MinCheckInterval
	}
	e.checkInterval = interval
}

// Start begins monitoring and executing all active plans
func (e *Executor) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return fmt.Errorf("executor is already running")
	}

	e.running = true

	// Load and start all active plans
	activePlans := e.manager.GetActivePlans()
	for _, plan := range activePlans {
		e.startPlanExecutor(plan)
	}

	// Start plan reload monitor in background
	go e.monitorPlanChanges()

	// Start swap verification monitor in background
	go e.monitorSwapVerification()

	return nil
}

// Stop halts all plan executions
func (e *Executor) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return
	}

	// Stop all active plan executors
	for _, pe := range e.activePlans {
		close(pe.stopChan)
	}

	e.activePlans = make(map[string]*planExecutor)
	e.running = false
	close(e.stopChan)
}

// StartPlan starts monitoring and executing a specific plan
func (e *Executor) StartPlan(planName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check if plan is already being executed
	if _, exists := e.activePlans[planName]; exists {
		return fmt.Errorf("plan '%s' is already being executed", planName)
	}

	// Load the plan
	plan, err := e.manager.GetPlan(planName)
	if err != nil {
		return err
	}

	// Verify plan is active
	if !plan.IsActive() {
		return fmt.Errorf("plan '%s' is not active", planName)
	}

	e.startPlanExecutor(plan)
	return nil
}

// StopPlan stops execution of a specific plan
func (e *Executor) StopPlan(planName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	pe, exists := e.activePlans[planName]
	if !exists {
		return fmt.Errorf("plan '%s' is not being executed", planName)
	}

	close(pe.stopChan)
	delete(e.activePlans, planName)

	return nil
}

// startPlanExecutor starts a goroutine to monitor and execute a plan (must be called with lock held)
func (e *Executor) startPlanExecutor(plan *TradingPlan) {
	pe := &planExecutor{
		plan:     plan,
		stopChan: make(chan struct{}),
		running:  true,
	}

	e.activePlans[plan.Name] = pe

	// Start monitoring goroutine
	go e.monitorPlan(pe)
}

// monitorPlan continuously monitors a plan and executes trades when conditions are met
func (e *Executor) monitorPlan(pe *planExecutor) {
	ticker := time.NewTicker(e.checkInterval)
	defer ticker.Stop()

	fmt.Printf("[Executor] Started monitoring plan: %s\n", pe.plan.Name)

	for {
		select {
		case <-pe.stopChan:
			fmt.Printf("[Executor] Stopped monitoring plan: %s\n", pe.plan.Name)
			return
		case <-ticker.C:
			e.checkAndExecutePlan(pe.plan.Name)
		}
	}
}

// checkAndExecutePlan checks if a plan should execute and performs the trade
func (e *Executor) checkAndExecutePlan(planName string) {
	// Reload plan to get latest state
	plan, err := e.manager.GetPlan(planName)
	if err != nil {
		fmt.Printf("[Executor] Error loading plan '%s': %v\n", planName, err)
		return
	}

	// Check if we can execute today (daily limit check)
	if !plan.CanExecuteToday() {
		// Daily limit reached, will try again tomorrow
		return
	}

	// Check if plan should execute
	shouldExecute, priceInfo, err := e.pricer.ShouldExecute(plan)
	if err != nil {
		fmt.Printf("[Executor] Error checking price for plan '%s': %v\n", planName, err)
		return
	}

	if !shouldExecute {
		// Price condition not met, continue monitoring
		return
	}

	fmt.Printf("[Executor] Trigger condition met for plan '%s'! Price: %s %s/%s\n",
		planName, priceInfo.Price, plan.DestToken, plan.SourceToken)

	// Execute the trade
	if err := e.executeTrade(plan, priceInfo); err != nil {
		fmt.Printf("[Executor] Failed to execute trade for plan '%s': %v\n", planName, err)
		return
	}

	// Check if plan is completed after this execution
	plan, _ = e.manager.GetPlan(planName)
	if plan.IsCompleted() {
		fmt.Printf("[Executor] Plan '%s' has completed all trades!\n", planName)
		e.mu.Lock()
		if pe, exists := e.activePlans[planName]; exists {
			close(pe.stopChan)
			delete(e.activePlans, planName)
		}
		e.mu.Unlock()
	}
}

// executeTrade performs a single trade for a plan
func (e *Executor) executeTrade(plan *TradingPlan, priceInfo *PriceInfo) error {
	// Calculate the amount to trade for this execution
	// Use the smaller of: amountPerTrade, remaining daily amount, or remaining total amount
	amountPerTrade, _ := strconv.ParseFloat(plan.AmountPerTrade, 64)
	remainingDaily, _ := strconv.ParseFloat(plan.GetRemainingDailyAmount(), 64)
	remainingTotal, _ := strconv.ParseFloat(plan.RemainingAmount, 64)

	// Find the minimum
	executeAmount := amountPerTrade
	if remainingDaily < executeAmount {
		executeAmount = remainingDaily
	}
	if remainingTotal < executeAmount {
		executeAmount = remainingTotal
	}

	executeAmountStr := fmt.Sprintf("%.8f", executeAmount)

	fmt.Printf("[Executor] Executing trade for plan '%s': %s %s -> %s\n",
		plan.Name, executeAmountStr, plan.SourceToken, plan.DestToken)

	// Create swap request
	swapReq := &types.SwapRequest{
		Amount:        executeAmountStr,
		SourceToken:   plan.SourceToken,
		DestToken:     plan.DestToken,
		SourceChain:   plan.SourceChain,
		DestChain:     plan.DestChain,
		RecipientAddr: plan.RecipientAddr,
		RefundAddr:    plan.RefundAddr,
	}

	// Get quote from API
	quote, err := e.apiClient.GetQuote(swapReq)
	if err != nil {
		return fmt.Errorf("failed to get quote: %w", err)
	}

	quoteDetails := quote.GetQuote()

	// Create execution record
	execution := Execution{
		Amount:          executeAmountStr,
		TriggerPrice:    priceInfo.Price,
		ActualPrice:     priceInfo.Price,
		DepositAddress:  quoteDetails.GetDepositAddress(),
		Status:          ExecutionPending,
		EstimatedOutput: quoteDetails.GetAmountOutFormatted(),
	}

	// Add execution to plan and get the execution ID
	executionID, err := e.manager.AddExecution(plan.Name, execution)
	if err != nil {
		return fmt.Errorf("failed to record execution: %w", err)
	}

	fmt.Printf("[Executor] Deposit address: %s\n", quoteDetails.GetDepositAddress())
	fmt.Printf("[Executor] Expected output: %s %s\n", quoteDetails.GetAmountOutFormatted(), plan.DestToken)

	// Auto-deposit is always enabled for plans
	if e.config.AutoDeposit.Enabled {
		if err := e.handleAutoDeposit(plan, executionID, swapReq, &quoteDetails); err != nil {
			fmt.Printf("[Executor] Auto-deposit failed: %v\n", err)
			fmt.Printf("[Executor] Please manually deposit %s %s to: %s\n",
				executeAmountStr, plan.SourceToken, quoteDetails.GetDepositAddress())
		}
	} else {
		fmt.Printf("[Executor] WARNING: Auto-deposit is not configured. Please enable it in your config.\n")
		fmt.Printf("[Executor] Manual deposit required: Send %s %s to %s\n",
			executeAmountStr, plan.SourceToken, quoteDetails.GetDepositAddress())
	}

	return nil
}

// handleAutoDeposit attempts to automatically send the deposit
func (e *Executor) handleAutoDeposit(plan *TradingPlan, executionID string, swapReq *types.SwapRequest, quoteDetails *oneclick.Quote) error {
	depositMgr := deposit.NewManager(e.config.AutoDeposit)

	if !depositMgr.IsEnabledForChain(plan.SourceChain) {
		return fmt.Errorf("auto-deposit not enabled for chain: %s", plan.SourceChain)
	}

	depositAddress := quoteDetails.GetDepositAddress()
	txid, err := depositMgr.SendDeposit(plan.SourceChain, depositAddress, plan.AmountPerTrade)
	if err != nil {
		// Update execution with failure
		e.manager.UpdateExecutionStatus(plan.Name, executionID, ExecutionFailed, "", err.Error())
		return err
	}

	fmt.Printf("[Executor] Auto-deposit successful! TX: %s\n", txid)

	// Update execution with transaction hash
	e.manager.UpdateExecutionStatus(plan.Name, executionID, ExecutionDeposited, txid, "")

	// Start background verification for this swap
	go e.verifySwapCompletion(plan.Name, executionID, quoteDetails.GetDepositAddress())

	return nil
}

// GetRunningPlans returns a list of plans currently being executed
func (e *Executor) GetRunningPlans() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	plans := make([]string, 0, len(e.activePlans))
	for name := range e.activePlans {
		plans = append(plans, name)
	}

	return plans
}

// IsRunning returns true if the executor is running
func (e *Executor) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.running
}

// IsPlanRunning returns true if a specific plan is being executed
func (e *Executor) IsPlanRunning(planName string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	_, exists := e.activePlans[planName]
	return exists
}

// monitorPlanChanges periodically checks for new/stopped/started plans
func (e *Executor) monitorPlanChanges() {
	ticker := time.NewTicker(PlanReloadInterval)
	defer ticker.Stop()

	for {
		select {
		case <-e.stopChan:
			return
		case <-ticker.C:
			e.reloadPlans()
		}
	}
}

// reloadPlans checks storage for plan changes and adjusts running executors
func (e *Executor) reloadPlans() {
	// Get current active plans from storage
	activePlans := e.manager.GetActivePlans()
	activeMap := make(map[string]*TradingPlan)
	for _, plan := range activePlans {
		activeMap[plan.Name] = plan
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Find plans that should be running but aren't (new or restarted plans)
	for name, plan := range activeMap {
		if _, isRunning := e.activePlans[name]; !isRunning {
			fmt.Printf("[Executor] Detected new active plan: %s\n", name)
			fmt.Printf("[Executor] Starting execution for: %s %s -> %s\n",
				plan.TotalAmount, plan.SourceToken, plan.DestToken)
			e.startPlanExecutor(plan)
		}
	}

	// Find plans that are running but shouldn't be (stopped or deleted plans)
	for name, pe := range e.activePlans {
		if _, shouldRun := activeMap[name]; !shouldRun {
			fmt.Printf("[Executor] Plan '%s' has been stopped or deleted\n", name)
			fmt.Printf("[Executor] Stopping execution for: %s\n", name)
			close(pe.stopChan)
			delete(e.activePlans, name)
		}
	}
}

// monitorSwapVerification periodically checks pending swaps for completion
func (e *Executor) monitorSwapVerification() {
	ticker := time.NewTicker(SwapVerificationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-e.stopChan:
			return
		case <-ticker.C:
			e.verifyPendingSwaps()
		}
	}
}

// verifyPendingSwaps checks all pending executions across all plans
func (e *Executor) verifyPendingSwaps() {
	// Get all active plans
	plans := e.manager.ListPlans()

	for _, plan := range plans {
		// Check each execution in the plan
		for i := range plan.ExecutionHistory {
			exec := &plan.ExecutionHistory[i]

			// Only verify if status is deposited or pending and we have a deposit address
			if (exec.Status == ExecutionDeposited || exec.Status == ExecutionPending) && exec.DepositAddress != "" {
				// Check if this is a recent execution (within last 24 hours)
				if time.Since(exec.Timestamp) < 24*time.Hour {
					e.checkSwapStatus(plan.Name, exec.ID, exec.DepositAddress)
				}
			}
		}
	}
}

// verifySwapCompletion monitors a specific swap until completion (runs in background)
func (e *Executor) verifySwapCompletion(planName, executionID, depositAddress string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	maxAttempts := 120 // Monitor for up to 1 hour (120 * 30s)
	attempts := 0

	for attempts < maxAttempts {
		select {
		case <-e.stopChan:
			return
		case <-ticker.C:
			completed := e.checkSwapStatus(planName, executionID, depositAddress)
			if completed {
				return
			}
			attempts++
		}
	}
}

// checkSwapStatus checks the status of a swap and updates the execution
// Returns true if the swap is in a terminal state (completed/failed)
func (e *Executor) checkSwapStatus(planName, executionID, depositAddress string) bool {
	status, err := e.apiClient.GetSwapStatus(depositAddress)
	if err != nil {
		// Silent failure - will retry next time
		return false
	}

	swapStatus := status.GetStatus()
	swapDetails := status.GetSwapDetails()

	// Extract actual output amount
	actualOutput := ""
	if swapDetails.HasAmountOutFormatted() {
		actualOutput = swapDetails.GetAmountOutFormatted()
	}

	// Extract destination transaction hash
	destTxHash := ""
	destTxs := swapDetails.GetDestinationChainTxHashes()
	if len(destTxs) > 0 {
		destTxHash = destTxs[0].GetHash()
	}

	// Update execution with swap status
	err = e.manager.UpdateExecutionWithSwapStatus(planName, executionID, swapStatus, actualOutput, destTxHash)
	if err != nil {
		fmt.Printf("[Verifier] Error updating execution status: %v\n", err)
		return false
	}

	// Check if swap is in terminal state
	if swapStatus == "SUCCESS" || swapStatus == "COMPLETED" {
		fmt.Printf("[Verifier] ✓ Swap completed for plan '%s'! Received: %s\n", planName, actualOutput)
		return true
	} else if swapStatus == "FAILED" || swapStatus == "REFUNDED" {
		fmt.Printf("[Verifier] ✗ Swap failed for plan '%s': %s\n", planName, swapStatus)
		return true
	}

	return false
}
