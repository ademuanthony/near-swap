package parser

import (
	"fmt"
	"regexp"
	"strings"

	"near-swap/pkg/types"
)

// ParseSwapCommand parses a natural language swap command
// Examples:
//   - "swap 1 SOL to USDC"
//   - "1.5 ETH to BTC"
//   - "100 USDC to SOL"
func ParseSwapCommand(command string) (*types.SwapRequest, error) {
	// Normalize the command
	command = strings.TrimSpace(strings.ToUpper(command))

	// Remove the word "SWAP" if present at the beginning
	command = strings.TrimPrefix(command, "SWAP ")

	// Pattern: <amount> <source_token> TO <dest_token>
	// Matches: "1 SOL TO USDC", "1.5 ETH TO BTC", "100.25 USDC TO SOL"
	pattern := regexp.MustCompile(`^(\d+\.?\d*)\s+([A-Z0-9]+)\s+TO\s+([A-Z0-9]+)$`)

	matches := pattern.FindStringSubmatch(command)
	if matches == nil {
		return nil, fmt.Errorf("invalid swap command format. Expected: 'swap <amount> <token> to <token>' (e.g., 'swap 1 SOL to USDC')")
	}

	return &types.SwapRequest{
		Amount:      matches[1],
		SourceToken: matches[2],
		DestToken:   matches[3],
	}, nil
}

// ValidateSwapRequest validates that a swap request has all required fields
func ValidateSwapRequest(req *types.SwapRequest) error {
	if req.Amount == "" {
		return fmt.Errorf("amount is required")
	}
	if req.SourceToken == "" {
		return fmt.Errorf("source token is required")
	}
	if req.DestToken == "" {
		return fmt.Errorf("destination token is required")
	}
	return nil
}

// NormalizeTokenSymbol normalizes token symbols to standard format
func NormalizeTokenSymbol(symbol string) string {
	// Convert to uppercase for consistency
	symbol = strings.TrimSpace(strings.ToUpper(symbol))

	// Handle common aliases
	aliases := map[string]string{
		"WBTC":     "BTC",
		"WETH":     "ETH",
		"WSOL":     "SOL",
	}

	if normalized, exists := aliases[symbol]; exists {
		return normalized
	}

	return symbol
}
