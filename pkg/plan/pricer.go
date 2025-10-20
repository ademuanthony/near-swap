package plan

import (
	"fmt"
	"math"
	"strconv"

	"near-swap/pkg/client"
	"near-swap/pkg/types"
)

// Pricer handles price fetching for trading plans
type Pricer struct {
	client *client.OneClickClient
}

// NewPricer creates a new pricer instance
func NewPricer(apiClient *client.OneClickClient) *Pricer {
	return &Pricer{
		client: apiClient,
	}
}

// PriceInfo contains price information for a token pair
type PriceInfo struct {
	Price          string  // Price of 1 unit of source token in dest tokens
	PriceFloat     float64 // Price as float for comparison
	SourceToken    string
	DestToken      string
	SourceChain    string
	DestChain      string
}

// GetPrice fetches the current price for a token pair using a small test amount
func (p *Pricer) GetPrice(plan *TradingPlan) (*PriceInfo, error) {
	// Use a small test amount (0.1 of amountPerTrade) to get the price
	testAmountFloat, err := strconv.ParseFloat(plan.AmountPerTrade, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount per trade: %w", err)
	}

	// Use 10% of the amount per trade for price checking (or minimum 0.01)
	testAmountFloat = testAmountFloat * 0.1
	if testAmountFloat < 0.01 {
		testAmountFloat = 0.01
	}
	testAmount := fmt.Sprintf("%.8f", testAmountFloat)

	// Create a dummy swap request to get a quote
	swapReq := &types.SwapRequest{
		Amount:        testAmount,
		SourceToken:   plan.SourceToken,
		DestToken:     plan.DestToken,
		SourceChain:   plan.SourceChain,
		DestChain:     plan.DestChain,
		RecipientAddr: plan.RecipientAddr,
		RefundAddr:    plan.RefundAddr,
	}

	// Get quote from API (with dry=true to avoid creating actual deposit address)
	quote, err := p.client.GetQuote(swapReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get quote: %w", err)
	}

	// Extract price from quote
	quoteDetails := quote.GetQuote()

	// Parse input and output amounts
	amountIn := quoteDetails.GetAmountInFormatted()
	amountOut := quoteDetails.GetAmountOutFormatted()

	amountInFloat, err := strconv.ParseFloat(amountIn, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse amount in: %w", err)
	}

	amountOutFloat, err := strconv.ParseFloat(amountOut, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse amount out: %w", err)
	}

	// Calculate price: how many dest tokens for 1 source token
	if amountInFloat == 0 {
		return nil, fmt.Errorf("invalid amount in: 0")
	}

	price := amountOutFloat / amountInFloat
	priceStr := fmt.Sprintf("%.8f", price)

	return &PriceInfo{
		Price:       priceStr,
		PriceFloat:  price,
		SourceToken: plan.SourceToken,
		DestToken:   plan.DestToken,
		SourceChain: plan.SourceChain,
		DestChain:   plan.DestChain,
	}, nil
}

// CheckTriggerCondition checks if the current price meets the plan's trigger condition
func (p *Pricer) CheckTriggerCondition(plan *TradingPlan, currentPrice *PriceInfo) (bool, error) {
	triggerPrice, err := strconv.ParseFloat(plan.TriggerPrice, 64)
	if err != nil {
		return false, fmt.Errorf("invalid trigger price: %w", err)
	}

	switch plan.PriceCondition {
	case PriceAbove:
		return currentPrice.PriceFloat >= triggerPrice, nil
	case PriceBelow:
		return currentPrice.PriceFloat <= triggerPrice, nil
	case PriceAt:
		// Use a 0.5% tolerance for "at" condition
		tolerance := triggerPrice * 0.005
		diff := math.Abs(currentPrice.PriceFloat - triggerPrice)
		return diff <= tolerance, nil
	default:
		return false, fmt.Errorf("unknown price condition: %s", plan.PriceCondition)
	}
}

// ShouldExecute determines if a plan should execute a trade based on current price
func (p *Pricer) ShouldExecute(plan *TradingPlan) (bool, *PriceInfo, error) {
	// Check if plan can execute
	if !plan.CanExecute() {
		return false, nil, nil
	}

	// Get current price
	currentPrice, err := p.GetPrice(plan)
	if err != nil {
		return false, nil, err
	}

	// Check trigger condition
	triggered, err := p.CheckTriggerCondition(plan, currentPrice)
	if err != nil {
		return false, nil, err
	}

	return triggered, currentPrice, nil
}
