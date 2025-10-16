package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	oneclick "github.com/defuse-protocol/one-click-sdk-go"
	"near-swap/pkg/types"
)

// OneClickClient wraps the 1Click SDK
type OneClickClient struct {
	client *oneclick.APIClient
	ctx    context.Context
}

// NewOneClickClient creates a new 1Click API client
func NewOneClickClient(jwtToken string) *OneClickClient {
	config := oneclick.NewConfiguration()

	// Create authenticated context
	ctx := context.WithValue(context.Background(), oneclick.ContextAccessToken, jwtToken)

	client := oneclick.NewAPIClient(config)

	return &OneClickClient{
		client: client,
		ctx:    ctx,
	}
}

// GetSupportedTokens retrieves all supported tokens
func (c *OneClickClient) GetSupportedTokens() ([]oneclick.TokenResponse, error) {
	resp, httpResp, err := c.client.OneClickAPI.GetTokens(c.ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get tokens: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status code %d", httpResp.StatusCode)
	}

	return resp, nil
}

// FindToken searches for a token by symbol across all chains
func (c *OneClickClient) FindToken(symbol string) (*oneclick.TokenResponse, error) {
	tokens, err := c.GetSupportedTokens()
	if err != nil {
		return nil, err
	}

	symbol = strings.ToUpper(symbol)

	// Try exact match first
	for _, token := range tokens {
		if strings.ToUpper(token.GetSymbol()) == symbol {
			return &token, nil
		}
	}

	// Try partial match
	for _, token := range tokens {
		if strings.Contains(strings.ToUpper(token.GetSymbol()), symbol) {
			return &token, nil
		}
	}

	return nil, fmt.Errorf("token '%s' not found", symbol)
}

// FindTokenOnChain searches for a token by symbol on a specific chain
func (c *OneClickClient) FindTokenOnChain(symbol, chain string) (*oneclick.TokenResponse, error) {
	tokens, err := c.GetSupportedTokens()
	if err != nil {
		return nil, err
	}

	symbol = strings.ToUpper(symbol)
	chain = strings.ToLower(chain)

	for _, token := range tokens {
		if strings.ToUpper(token.GetSymbol()) == symbol &&
			strings.ToLower(token.GetBlockchain()) == chain {
			return &token, nil
		}
	}

	return nil, fmt.Errorf("token '%s' not found on chain '%s'", symbol, chain)
}

// GetQuote generates a swap quote
func (c *OneClickClient) GetQuote(req *types.SwapRequest) (*oneclick.QuoteResponse, error) {
	// Find source and destination tokens
	var sourceToken, destToken *oneclick.TokenResponse
	var err error

	if req.SourceChain != "" {
		sourceToken, err = c.FindTokenOnChain(req.SourceToken, req.SourceChain)
	} else {
		sourceToken, err = c.FindToken(req.SourceToken)
	}
	if err != nil {
		return nil, fmt.Errorf("source token error: %w", err)
	}

	if req.DestChain != "" {
		destToken, err = c.FindTokenOnChain(req.DestToken, req.DestChain)
	} else {
		destToken, err = c.FindToken(req.DestToken)
	}
	if err != nil {
		return nil, fmt.Errorf("destination token error: %w", err)
	}

	// Convert amount to smallest unit (wei-like format)
	amountFloat, err := strconv.ParseFloat(req.Amount, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	// Multiply by 10^decimals to get smallest unit
	smallestUnit := amountFloat * math.Pow(10, float64(sourceToken.GetDecimals()))
	amountStr := fmt.Sprintf("%.0f", smallestUnit)

	// Set recipient - required for the API
	recipient := req.RecipientAddr
	if recipient == "" {
		return nil, fmt.Errorf("recipient address is required. Use --recipient flag to specify where you want to receive the tokens")
	}

	// Set refund address - use provided refund address or default to recipient
	refundTo := req.RefundAddr
	if refundTo == "" {
		refundTo = recipient
	}

	// Calculate deadline (24 hours from now)
	deadline := time.Now().Add(24 * time.Hour)

	// Build quote request with all required parameters
	quoteReq := oneclick.NewQuoteRequest(
		false,                     // dry - false to get a real deposit address
		"EXACT_INPUT",             // swapType
		100,                       // slippageTolerance (1%)
		sourceToken.GetAssetId(),  // originAsset
		"ORIGIN_CHAIN",            // depositType
		destToken.GetAssetId(),    // destinationAsset
		amountStr,                 // amount in smallest unit
		refundTo,                  // refundTo
		"ORIGIN_CHAIN",            // refundType
		recipient,                 // recipient
		"DESTINATION_CHAIN",       // recipientType
		deadline,                  // deadline
	)

	// Execute quote request
	resp, httpResp, err := c.client.OneClickAPI.GetQuote(c.ctx).QuoteRequest(*quoteReq).Execute()
	if err != nil {
		// Try to extract the actual error message from the response
		if httpResp != nil {
			defer httpResp.Body.Close()
			bodyBytes, readErr := io.ReadAll(httpResp.Body)
			if readErr == nil && len(bodyBytes) > 0 {
				// Try to parse as a generic error response
				var errorResp map[string]interface{}
				if jsonErr := json.Unmarshal(bodyBytes, &errorResp); jsonErr == nil {
					if message, ok := errorResp["message"].(string); ok {
						return nil, fmt.Errorf("API error (status %d): %s", httpResp.StatusCode, message)
					}
					if errors, ok := errorResp["errors"]; ok {
						return nil, fmt.Errorf("API error (status %d): %v", httpResp.StatusCode, errors)
					}
				}
				// If we can't parse it, show the raw body
				return nil, fmt.Errorf("API error (status %d): %s", httpResp.StatusCode, string(bodyBytes))
			}
			return nil, fmt.Errorf("failed to get quote from API (status: %d): %w", httpResp.StatusCode, err)
		}
		return nil, fmt.Errorf("failed to get quote from API: %w", err)
	}
	defer httpResp.Body.Close()

	// Check for successful status codes (200-299)
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, fmt.Errorf("API returned status code %d", httpResp.StatusCode)
	}

	if resp == nil {
		return nil, fmt.Errorf("empty quote response")
	}

	return resp, nil
}

// GetSwapStatus checks the execution status of a swap
func (c *OneClickClient) GetSwapStatus(depositAddress string) (*oneclick.GetExecutionStatusResponse, error) {
	resp, httpResp, err := c.client.OneClickAPI.GetExecutionStatus(c.ctx).DepositAddress(depositAddress).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status code %d", httpResp.StatusCode)
	}

	return resp, nil
}

// SubmitDepositTx submits the deposit transaction hash
func (c *OneClickClient) SubmitDepositTx(depositAddress, txHash string) error {
	req := oneclick.NewSubmitDepositTxRequest(depositAddress, txHash)

	_, httpResp, err := c.client.OneClickAPI.SubmitDepositTx(c.ctx).SubmitDepositTxRequest(*req).Execute()
	if err != nil {
		return fmt.Errorf("failed to submit deposit: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != 200 && httpResp.StatusCode != 201 {
		return fmt.Errorf("API returned status code %d", httpResp.StatusCode)
	}

	return nil
}
