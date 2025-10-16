package deposit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"near-swap/config"
)

// MoneroDepositor handles Monero deposits using monero-wallet-rpc
type MoneroDepositor struct {
	config config.MoneroConfig
	client *http.Client
}

// NewMoneroDepositor creates a new Monero depositor
func NewMoneroDepositor(cfg config.MoneroConfig) *MoneroDepositor {
	return &MoneroDepositor{
		config: cfg,
		client: &http.Client{},
	}
}

// MoneroRPCRequest represents a JSON-RPC request to monero-wallet-rpc
type MoneroRPCRequest struct {
	JSONRpc string        `json:"jsonrpc"`
	ID      string        `json:"id"`
	Method  string        `json:"method"`
	Params  interface{}   `json:"params,omitempty"`
}

// MoneroRPCResponse represents a JSON-RPC response from monero-wallet-rpc
type MoneroRPCResponse struct {
	JSONRpc string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *MoneroRPCError `json:"error,omitempty"`
}

// MoneroRPCError represents an error in the RPC response
type MoneroRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// SendDeposit sends Monero to the specified address
func (m *MoneroDepositor) SendDeposit(address string, amount string) (string, error) {
	// Validate RPC connection
	if err := m.validateRPC(); err != nil {
		return "", fmt.Errorf("monero-wallet-rpc validation failed: %w", err)
	}

	// Get wallet balance first
	balance, err := m.getBalance()
	if err != nil {
		return "", fmt.Errorf("failed to get wallet balance: %w", err)
	}

	// Parse amount - Monero uses atomic units (1 XMR = 1e12 atomic units)
	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return "", fmt.Errorf("invalid amount: %w", err)
	}

	// Convert to atomic units
	amountAtomic := uint64(amountFloat * 1e12)

	// Check if we have enough balance
	if balance < amountAtomic {
		balanceXMR := float64(balance) / 1e12
		return "", fmt.Errorf("insufficient balance: have %.12f XMR, need %.12f XMR", balanceXMR, amountFloat)
	}

	// Build transfer parameters
	transferParams := map[string]interface{}{
		"destinations": []map[string]interface{}{
			{
				"amount":  amountAtomic,
				"address": address,
			},
		},
		"account_index": m.config.AccountIndex,
		"priority":      m.config.Priority,
		"get_tx_key":    true,
	}

	// Add unlock_time if specified
	if m.config.UnlockTime > 0 {
		transferParams["unlock_time"] = m.config.UnlockTime
	}

	// Execute transfer
	result, err := m.callRPC("transfer", transferParams)
	if err != nil {
		return "", fmt.Errorf("monero-wallet-rpc transfer failed: %w", err)
	}

	// Parse the result to get transaction hash
	var transferResult struct {
		TxHash string `json:"tx_hash"`
		TxKey  string `json:"tx_key"`
	}

	if err := json.Unmarshal(result, &transferResult); err != nil {
		return "", fmt.Errorf("failed to parse transfer result: %w", err)
	}

	if transferResult.TxHash == "" {
		return "", fmt.Errorf("empty transaction hash returned")
	}

	return transferResult.TxHash, nil
}

// getBalance returns the wallet balance in atomic units
func (m *MoneroDepositor) getBalance() (uint64, error) {
	params := map[string]interface{}{
		"account_index": m.config.AccountIndex,
	}

	result, err := m.callRPC("get_balance", params)
	if err != nil {
		return 0, fmt.Errorf("monero-wallet-rpc get_balance failed: %w", err)
	}

	var balanceResult struct {
		Balance          uint64 `json:"balance"`
		UnlockedBalance  uint64 `json:"unlocked_balance"`
	}

	if err := json.Unmarshal(result, &balanceResult); err != nil {
		return 0, fmt.Errorf("failed to parse balance: %w", err)
	}

	// Return unlocked balance (available for spending)
	return balanceResult.UnlockedBalance, nil
}

// validateRPC checks if monero-wallet-rpc is accessible
func (m *MoneroDepositor) validateRPC() error {
	_, err := m.callRPC("get_version", nil)
	if err != nil {
		return fmt.Errorf("monero-wallet-rpc not accessible: %w", err)
	}
	return nil
}

// callRPC makes a JSON-RPC call to monero-wallet-rpc
func (m *MoneroDepositor) callRPC(method string, params interface{}) (json.RawMessage, error) {
	// Build RPC request
	rpcReq := MoneroRPCRequest{
		JSONRpc: "2.0",
		ID:      "0",
		Method:  method,
		Params:  params,
	}

	// Marshal request
	reqBody, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build URL
	url := fmt.Sprintf("http://%s:%d/json_rpc", m.config.Host, m.config.Port)

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add authentication if configured
	if m.config.Username != "" && m.config.Password != "" {
		req.SetBasicAuth(m.config.Username, m.config.Password)
	}

	// Execute request
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("RPC request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RPC returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var rpcResp MoneroRPCResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for RPC error
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error (code %d): %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

// GetTransactionInfo retrieves information about a transaction
func (m *MoneroDepositor) GetTransactionInfo(txid string) (map[string]interface{}, error) {
	params := map[string]interface{}{
		"txid": txid,
	}

	result, err := m.callRPC("get_transfer_by_txid", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction info: %w", err)
	}

	var txInfo map[string]interface{}
	if err := json.Unmarshal(result, &txInfo); err != nil {
		return nil, fmt.Errorf("failed to parse transaction info: %w", err)
	}

	return txInfo, nil
}
