package plan

import (
	"fmt"
	"strconv"
	"time"
)

// PriceCondition defines when a trade should be triggered
type PriceCondition string

const (
	PriceAbove PriceCondition = "above" // Trigger when price goes above target
	PriceBelow PriceCondition = "below" // Trigger when price goes below target
	PriceAt    PriceCondition = "at"    // Trigger when price equals target (with tolerance)
)

// PlanStatus defines the current state of a trading plan
type PlanStatus string

const (
	StatusActive    PlanStatus = "active"    // Plan is running
	StatusPaused    PlanStatus = "paused"    // Plan is paused
	StatusCompleted PlanStatus = "completed" // Plan has executed all trades
	StatusCancelled PlanStatus = "cancelled" // Plan was cancelled
)

// ExecutionStatus defines the status of a single execution
type ExecutionStatus string

const (
	ExecutionPending   ExecutionStatus = "pending"    // Execution initiated
	ExecutionDeposited ExecutionStatus = "deposited"  // Deposit sent
	ExecutionCompleted ExecutionStatus = "completed"  // Swap completed
	ExecutionFailed    ExecutionStatus = "failed"     // Execution failed
)

// TradingPlan represents a user's automated trading strategy
type TradingPlan struct {
	// Identity
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Created     time.Time `json:"created"`
	LastUpdated time.Time `json:"last_updated"`

	// Trading parameters
	SourceToken    string  `json:"source_token"`     // Token to sell (e.g., "BTC")
	DestToken      string  `json:"dest_token"`       // Token to buy (e.g., "USDC")
	SourceChain    string  `json:"source_chain"`     // Source blockchain
	DestChain      string  `json:"dest_chain"`       // Destination blockchain
	TotalAmount    string  `json:"total_amount"`     // Total amount to trade
	AmountPerTrade string  `json:"amount_per_trade"` // Amount per execution
	AmountPerDay   string  `json:"amount_per_day"`   // Maximum amount to trade per day
	TriggerPrice   string  `json:"trigger_price"`    // Price target
	PriceCondition PriceCondition `json:"price_condition"` // When to trigger

	// Addresses
	RecipientAddr string `json:"recipient_addr"` // Where to receive tokens
	RefundAddr    string `json:"refund_addr"`    // Where to refund if swap fails

	// Execution tracking
	Status           PlanStatus   `json:"status"`
	TotalExecuted    string       `json:"total_executed"`     // Amount already executed
	RemainingAmount  string       `json:"remaining_amount"`   // Amount left to execute
	ExecutionHistory []Execution  `json:"execution_history"`  // History of executions
	ExecutionCount   int          `json:"execution_count"`    // Number of executions

	// Daily execution tracking
	LastExecutionDate string `json:"last_execution_date"` // Date of last execution (YYYY-MM-DD)
	TodayExecuted     string `json:"today_executed"`      // Amount executed today
}

// Execution represents a single trade execution within a plan
type Execution struct {
	ID                string          `json:"id"`               // Unique execution ID
	Timestamp         time.Time       `json:"timestamp"`        // When execution occurred
	Amount            string          `json:"amount"`           // Amount traded
	TriggerPrice      string          `json:"trigger_price"`    // Price at trigger
	ActualPrice       string          `json:"actual_price"`     // Actual execution price
	DepositAddress    string          `json:"deposit_address"`  // Deposit address from quote
	TxHash            string          `json:"tx_hash"`          // Deposit transaction hash
	Status            ExecutionStatus `json:"status"`           // Execution status
	ErrorMessage      string          `json:"error_message,omitempty"` // Error if failed
	EstimatedOutput   string          `json:"estimated_output"` // Expected output amount
	ActualOutput      string          `json:"actual_output,omitempty"` // Actual received amount
	DestinationTxHash string          `json:"destination_tx_hash,omitempty"` // Withdrawal transaction hash
	CompletionTime    *time.Time      `json:"completion_time,omitempty"` // When swap completed
	SwapStatus        string          `json:"swap_status,omitempty"` // Latest status from API
}

// Validate checks if the trading plan has valid parameters
func (tp *TradingPlan) Validate() error {
	if tp.Name == "" {
		return fmt.Errorf("plan name is required")
	}
	if tp.SourceToken == "" {
		return fmt.Errorf("source token is required")
	}
	if tp.DestToken == "" {
		return fmt.Errorf("destination token is required")
	}
	if tp.SourceChain == "" {
		return fmt.Errorf("source chain is required")
	}
	if tp.DestChain == "" {
		return fmt.Errorf("destination chain is required")
	}
	if tp.TotalAmount == "" || tp.TotalAmount == "0" {
		return fmt.Errorf("total amount must be greater than 0")
	}
	if tp.AmountPerTrade == "" || tp.AmountPerTrade == "0" {
		return fmt.Errorf("amount per trade must be greater than 0")
	}
	if tp.AmountPerDay == "" || tp.AmountPerDay == "0" {
		return fmt.Errorf("amount per day must be greater than 0")
	}
	if tp.TriggerPrice == "" || tp.TriggerPrice == "0" {
		return fmt.Errorf("trigger price must be greater than 0")
	}
	if tp.PriceCondition != PriceAbove && tp.PriceCondition != PriceBelow && tp.PriceCondition != PriceAt {
		return fmt.Errorf("price condition must be 'above', 'below', or 'at'")
	}
	if tp.RecipientAddr == "" {
		return fmt.Errorf("recipient address is required")
	}
	return nil
}

// IsActive returns true if the plan is currently active
func (tp *TradingPlan) IsActive() bool {
	return tp.Status == StatusActive
}

// IsCompleted returns true if the plan has completed all trades
func (tp *TradingPlan) IsCompleted() bool {
	return tp.Status == StatusCompleted
}

// CanExecute returns true if the plan can execute more trades
func (tp *TradingPlan) CanExecute() bool {
	return tp.Status == StatusActive && tp.RemainingAmount != "0"
}

// PlanSummary provides a simplified view of a plan for listing
type PlanSummary struct {
	Name            string     `json:"name"`
	SourceToken     string     `json:"source_token"`
	DestToken       string     `json:"dest_token"`
	TotalAmount     string     `json:"total_amount"`
	RemainingAmount string     `json:"remaining_amount"`
	TriggerPrice    string     `json:"trigger_price"`
	PriceCondition  PriceCondition `json:"price_condition"`
	Status          PlanStatus `json:"status"`
	ExecutionCount  int        `json:"execution_count"`
	Created         time.Time  `json:"created"`
}

// ToSummary converts a TradingPlan to a PlanSummary
func (tp *TradingPlan) ToSummary() *PlanSummary {
	return &PlanSummary{
		Name:            tp.Name,
		SourceToken:     tp.SourceToken,
		DestToken:       tp.DestToken,
		TotalAmount:     tp.TotalAmount,
		RemainingAmount: tp.RemainingAmount,
		TriggerPrice:    tp.TriggerPrice,
		PriceCondition:  tp.PriceCondition,
		Status:          tp.Status,
		ExecutionCount:  tp.ExecutionCount,
		Created:         tp.Created,
	}
}

// CanExecuteToday returns true if the plan can execute more trades today
func (tp *TradingPlan) CanExecuteToday() bool {
	if !tp.CanExecute() {
		return false
	}

	// Get today's date
	today := time.Now().Format("2006-01-02")

	// If last execution was on a different day, reset daily counter
	if tp.LastExecutionDate != today {
		return true
	}

	// Check if we've reached the daily limit
	todayExecuted, _ := strconv.ParseFloat(tp.TodayExecuted, 64)
	dailyLimit, _ := strconv.ParseFloat(tp.AmountPerDay, 64)

	return todayExecuted < dailyLimit
}

// GetRemainingDailyAmount returns how much can still be executed today
func (tp *TradingPlan) GetRemainingDailyAmount() string {
	today := time.Now().Format("2006-01-02")

	// If this is a new day, full daily amount is available
	if tp.LastExecutionDate != today {
		return tp.AmountPerDay
	}

	// Calculate remaining for today
	todayExecuted, _ := strconv.ParseFloat(tp.TodayExecuted, 64)
	dailyLimit, _ := strconv.ParseFloat(tp.AmountPerDay, 64)
	remaining := dailyLimit - todayExecuted

	if remaining < 0 {
		return "0"
	}

	return fmt.Sprintf("%.8f", remaining)
}
