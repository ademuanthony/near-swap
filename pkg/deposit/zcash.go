package deposit

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"near-swap/config"
)

// ZcashDepositor handles Zcash deposits using zcash-cli
type ZcashDepositor struct {
	config config.ZcashConfig
}

// NewZcashDepositor creates a new Zcash depositor
func NewZcashDepositor(cfg config.ZcashConfig) *ZcashDepositor {
	return &ZcashDepositor{
		config: cfg,
	}
}

// SendDeposit sends Zcash to the specified address
func (z *ZcashDepositor) SendDeposit(address string, amount string) (string, error) {
	// Validate zcash-cli is available
	if err := z.validateCLI(); err != nil {
		return "", fmt.Errorf("zcash-cli validation failed: %w", err)
	}

	// Get wallet balance first
	balance, err := z.getBalance()
	if err != nil {
		return "", fmt.Errorf("failed to get wallet balance: %w", err)
	}

	// Parse amount
	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return "", fmt.Errorf("invalid amount: %w", err)
	}

	// Check if we have enough balance
	if balance < amountFloat {
		return "", fmt.Errorf("insufficient balance: have %.8f ZEC, need %.8f ZEC", balance, amountFloat)
	}

	// Build the sendtoaddress command
	args := z.buildBaseArgs()
	args = append(args, "sendtoaddress", address, amount)

	// Note: zcash-cli sendtoaddress has optional parameters like comment, comment_to,
	// subtractfeefromamount, and replaceable. We're using the simple form here.
	// For advanced options, users can customize via CLI args in config

	// Execute the command
	cmd := exec.Command(z.config.CLIPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("zcash-cli sendtoaddress failed: %w\nOutput: %s", err, string(output))
	}

	// Extract transaction ID
	txid := strings.TrimSpace(string(output))
	if txid == "" {
		return "", fmt.Errorf("empty transaction ID returned")
	}

	return txid, nil
}

// GetBalance returns the wallet balance
func (z *ZcashDepositor) getBalance() (float64, error) {
	args := z.buildBaseArgs()
	args = append(args, "getbalance")

	cmd := exec.Command(z.config.CLIPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("zcash-cli getbalance failed: %w\nOutput: %s", err, string(output))
	}

	balance, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse balance: %w", err)
	}

	return balance, nil
}

// validateCLI checks if zcash-cli is available and working
func (z *ZcashDepositor) validateCLI() error {
	args := z.buildBaseArgs()
	args = append(args, "getblockchaininfo")

	cmd := exec.Command(z.config.CLIPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("zcash-cli not accessible: %w\nOutput: %s", err, string(output))
	}

	// Try to parse the output as JSON to verify it's working
	var info map[string]interface{}
	if err := json.Unmarshal(output, &info); err != nil {
		return fmt.Errorf("invalid zcash-cli response: %w", err)
	}

	return nil
}

// buildBaseArgs constructs the base arguments for zcash-cli
func (z *ZcashDepositor) buildBaseArgs() []string {
	args := make([]string, 0)

	// Add any custom CLI arguments from config
	if len(z.config.CLIArgs) > 0 {
		args = append(args, z.config.CLIArgs...)
	}

	return args
}

// GetTransactionInfo retrieves information about a transaction
func (z *ZcashDepositor) GetTransactionInfo(txid string) (map[string]interface{}, error) {
	args := z.buildBaseArgs()
	args = append(args, "gettransaction", txid)

	cmd := exec.Command(z.config.CLIPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("zcash-cli gettransaction failed: %w\nOutput: %s", err, string(output))
	}

	var txInfo map[string]interface{}
	if err := json.Unmarshal(output, &txInfo); err != nil {
		return nil, fmt.Errorf("failed to parse transaction info: %w", err)
	}

	return txInfo, nil
}

// ListAddresses lists all addresses in the wallet
func (z *ZcashDepositor) ListAddresses() ([]string, error) {
	args := z.buildBaseArgs()
	args = append(args, "listaddresses")

	cmd := exec.Command(z.config.CLIPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("zcash-cli listaddresses failed: %w\nOutput: %s", err, string(output))
	}

	var addresses []string
	if err := json.Unmarshal(output, &addresses); err != nil {
		return nil, fmt.Errorf("failed to parse addresses: %w", err)
	}

	return addresses, nil
}
