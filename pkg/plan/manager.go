package plan

import (
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// Manager provides high-level operations for trading plans
type Manager struct {
	storage *Storage
}

// NewManager creates a new plan manager
func NewManager(storagePath string) (*Manager, error) {
	storage, err := NewStorage(storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	return &Manager{
		storage: storage,
	}, nil
}

// CreatePlan creates a new trading plan with validation
func (m *Manager) CreatePlan(
	name string,
	sourceToken, destToken string,
	sourceChain, destChain string,
	totalAmount, amountPerTrade, amountPerDay string,
	triggerPrice string,
	priceCondition PriceCondition,
	recipientAddr, refundAddr string,
	description string,
) (*TradingPlan, error) {
	// Check if plan already exists
	if m.storage.Exists(name) {
		return nil, fmt.Errorf("plan '%s' already exists", name)
	}

	// Validate amounts
	if err := validateAmount(totalAmount); err != nil {
		return nil, fmt.Errorf("invalid total amount: %w", err)
	}
	if err := validateAmount(amountPerTrade); err != nil {
		return nil, fmt.Errorf("invalid amount per trade: %w", err)
	}
	if err := validateAmount(amountPerDay); err != nil {
		return nil, fmt.Errorf("invalid amount per day: %w", err)
	}
	if err := validateAmount(triggerPrice); err != nil {
		return nil, fmt.Errorf("invalid trigger price: %w", err)
	}

	// Verify that amountPerTrade <= amountPerDay <= totalAmount
	totalFloat, _ := strconv.ParseFloat(totalAmount, 64)
	perTradeFloat, _ := strconv.ParseFloat(amountPerTrade, 64)
	perDayFloat, _ := strconv.ParseFloat(amountPerDay, 64)

	if perTradeFloat > perDayFloat {
		return nil, fmt.Errorf("amount per trade cannot be greater than amount per day")
	}
	if perDayFloat > totalFloat {
		return nil, fmt.Errorf("amount per day cannot be greater than total amount")
	}

	now := time.Now()

	plan := &TradingPlan{
		Name:              name,
		Description:       description,
		Created:           now,
		LastUpdated:       now,
		SourceToken:       sourceToken,
		DestToken:         destToken,
		SourceChain:       sourceChain,
		DestChain:         destChain,
		TotalAmount:       totalAmount,
		AmountPerTrade:    amountPerTrade,
		AmountPerDay:      amountPerDay,
		TriggerPrice:      triggerPrice,
		PriceCondition:    priceCondition,
		RecipientAddr:     recipientAddr,
		RefundAddr:        refundAddr,
		Status:            StatusPaused, // Start in paused state
		TotalExecuted:     "0",
		RemainingAmount:   totalAmount,
		ExecutionHistory:  []Execution{},
		ExecutionCount:    0,
		LastExecutionDate: "",
		TodayExecuted:     "0",
	}

	// Validate the plan
	if err := plan.Validate(); err != nil {
		return nil, err
	}

	// Save to storage
	if err := m.storage.Create(plan); err != nil {
		return nil, err
	}

	return plan, nil
}

// GetPlan retrieves a plan by name
func (m *Manager) GetPlan(name string) (*TradingPlan, error) {
	return m.storage.Get(name)
}

// ListPlans returns all plans
func (m *Manager) ListPlans() []*TradingPlan {
	return m.storage.List()
}

// ListPlansByStatus returns plans filtered by status
func (m *Manager) ListPlansByStatus(status PlanStatus) []*TradingPlan {
	return m.storage.ListByStatus(status)
}

// UpdatePlan updates an existing plan
func (m *Manager) UpdatePlan(plan *TradingPlan) error {
	plan.LastUpdated = time.Now()
	return m.storage.Update(plan)
}

// DeletePlan removes a plan
func (m *Manager) DeletePlan(name string) error {
	// Don't allow deletion of active plans
	plan, err := m.storage.Get(name)
	if err != nil {
		return err
	}

	if plan.Status == StatusActive {
		return fmt.Errorf("cannot delete active plan '%s', stop it first", name)
	}

	return m.storage.Delete(name)
}

// StartPlan activates a plan for execution
func (m *Manager) StartPlan(name string) error {
	plan, err := m.storage.Get(name)
	if err != nil {
		return err
	}

	if plan.Status == StatusActive {
		return fmt.Errorf("plan '%s' is already active", name)
	}

	if plan.Status == StatusCompleted {
		return fmt.Errorf("plan '%s' has already completed all trades", name)
	}

	plan.Status = StatusActive
	plan.LastUpdated = time.Now()

	return m.storage.Update(plan)
}

// StopPlan pauses a running plan
func (m *Manager) StopPlan(name string) error {
	plan, err := m.storage.Get(name)
	if err != nil {
		return err
	}

	if plan.Status != StatusActive {
		return fmt.Errorf("plan '%s' is not active", name)
	}

	plan.Status = StatusPaused
	plan.LastUpdated = time.Now()

	return m.storage.Update(plan)
}

// CancelPlan marks a plan as cancelled
func (m *Manager) CancelPlan(name string) error {
	plan, err := m.storage.Get(name)
	if err != nil {
		return err
	}

	plan.Status = StatusCancelled
	plan.LastUpdated = time.Now()

	return m.storage.Update(plan)
}

// AddExecution records a new execution for a plan
func (m *Manager) AddExecution(name string, execution Execution) error {
	plan, err := m.storage.Get(name)
	if err != nil {
		return err
	}

	// Add execution to history
	execution.ID = uuid.New().String()
	execution.Timestamp = time.Now()
	plan.ExecutionHistory = append(plan.ExecutionHistory, execution)
	plan.ExecutionCount++

	// Get today's date
	today := time.Now().Format("2006-01-02")

	// Reset daily counter if it's a new day
	if plan.LastExecutionDate != today {
		plan.LastExecutionDate = today
		plan.TodayExecuted = "0"
	}

	// Update amounts if execution is successful
	if execution.Status == ExecutionCompleted || execution.Status == ExecutionDeposited {
		executionAmount, _ := strconv.ParseFloat(execution.Amount, 64)

		// Update total executed
		totalExecuted, _ := strconv.ParseFloat(plan.TotalExecuted, 64)
		totalExecuted += executionAmount
		plan.TotalExecuted = fmt.Sprintf("%.8f", totalExecuted)

		// Update remaining amount
		remaining, _ := strconv.ParseFloat(plan.RemainingAmount, 64)
		remaining -= executionAmount
		plan.RemainingAmount = fmt.Sprintf("%.8f", remaining)

		// Update today's executed amount
		todayExecuted, _ := strconv.ParseFloat(plan.TodayExecuted, 64)
		todayExecuted += executionAmount
		plan.TodayExecuted = fmt.Sprintf("%.8f", todayExecuted)

		// Check if plan is completed
		if remaining <= 0.00000001 { // Small tolerance for floating point
			plan.Status = StatusCompleted
			plan.RemainingAmount = "0"
		}
	}

	plan.LastUpdated = time.Now()

	return m.storage.Update(plan)
}

// UpdateExecutionStatus updates the status of a specific execution
func (m *Manager) UpdateExecutionStatus(planName, executionID string, status ExecutionStatus, txHash string, errorMsg string) error {
	plan, err := m.storage.Get(planName)
	if err != nil {
		return err
	}

	// Find and update the execution
	found := false
	for i := range plan.ExecutionHistory {
		if plan.ExecutionHistory[i].ID == executionID {
			plan.ExecutionHistory[i].Status = status
			if txHash != "" {
				plan.ExecutionHistory[i].TxHash = txHash
			}
			if errorMsg != "" {
				plan.ExecutionHistory[i].ErrorMessage = errorMsg
			}
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("execution '%s' not found in plan '%s'", executionID, planName)
	}

	plan.LastUpdated = time.Now()

	return m.storage.Update(plan)
}

// GetExecutionHistory returns the execution history for a plan
func (m *Manager) GetExecutionHistory(name string) ([]Execution, error) {
	plan, err := m.storage.Get(name)
	if err != nil {
		return nil, err
	}

	return plan.ExecutionHistory, nil
}

// validateAmount checks if an amount string is valid
func validateAmount(amount string) error {
	if amount == "" {
		return fmt.Errorf("amount cannot be empty")
	}

	value, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return fmt.Errorf("invalid amount format: %w", err)
	}

	if value <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}

	return nil
}

// GetActivePlans returns all active plans
func (m *Manager) GetActivePlans() []*TradingPlan {
	return m.storage.ListByStatus(StatusActive)
}

// GetStorage returns the storage instance (useful for executor)
func (m *Manager) GetStorage() *Storage {
	return m.storage
}

// UpdateExecutionWithSwapStatus updates an execution with swap status details
func (m *Manager) UpdateExecutionWithSwapStatus(planName, executionID string, swapStatus, actualOutput, destTxHash string) error {
	plan, err := m.storage.Get(planName)
	if err != nil {
		return err
	}

	// Find and update the execution
	found := false
	for i := range plan.ExecutionHistory {
		if plan.ExecutionHistory[i].ID == executionID {
			plan.ExecutionHistory[i].SwapStatus = swapStatus

			if actualOutput != "" {
				plan.ExecutionHistory[i].ActualOutput = actualOutput
			}

			if destTxHash != "" {
				plan.ExecutionHistory[i].DestinationTxHash = destTxHash
			}

			// If status is completed/success, mark execution as completed and set completion time
			if swapStatus == "SUCCESS" || swapStatus == "COMPLETED" {
				plan.ExecutionHistory[i].Status = ExecutionCompleted
				now := time.Now()
				plan.ExecutionHistory[i].CompletionTime = &now
			} else if swapStatus == "FAILED" || swapStatus == "REFUNDED" {
				plan.ExecutionHistory[i].Status = ExecutionFailed
			}

			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("execution '%s' not found in plan '%s'", executionID, planName)
	}

	plan.LastUpdated = time.Now()

	return m.storage.Update(plan)
}
