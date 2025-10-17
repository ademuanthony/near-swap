package deposit

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"near-swap/config"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// EVMDepositor handles deposits on EVM-compatible blockchains
type EVMDepositor struct {
	config      config.EVMConfig
	networkName string
	network     config.EVMNetwork
	client      *ethclient.Client
	privateKey  *ecdsa.PrivateKey
}

// ERC20 transfer function ABI
const erc20TransferABI = `[{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"type":"function"}]`

// NewEVMDepositor creates a new EVM depositor for a specific network
func NewEVMDepositor(cfg config.EVMConfig, networkName string) (*EVMDepositor, error) {
	// Get network configuration
	network, exists := cfg.Networks[networkName]
	if !exists {
		return nil, fmt.Errorf("network %s not configured", networkName)
	}

	// Validate configuration
	if network.RPCUrl == "" {
		return nil, fmt.Errorf("RPC URL not configured for network %s", networkName)
	}
	if network.PrivateKey == "" {
		return nil, fmt.Errorf("private key not configured for network %s", networkName)
	}

	// Connect to the RPC endpoint
	client, err := ethclient.Dial(network.RPCUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC endpoint: %w", err)
	}

	// Parse private key
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(network.PrivateKey, "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	return &EVMDepositor{
		config:      cfg,
		networkName: networkName,
		network:     network,
		client:      client,
		privateKey:  privateKey,
	}, nil
}

// SendDeposit sends a deposit to the specified address
// For native tokens, address is the recipient
// For ERC20 tokens, address format is: "recipient|tokenContract"
func (e *EVMDepositor) SendDeposit(address string, amount string) (string, error) {
	ctx := context.Background()

	// Parse address - check if it contains token contract address for ERC20
	parts := strings.Split(address, "|")
	recipientAddr := parts[0]
	var tokenContract string
	if len(parts) > 1 {
		tokenContract = parts[1]
	}

	// Validate recipient address
	if !common.IsHexAddress(recipientAddr) {
		return "", fmt.Errorf("invalid recipient address: %s", recipientAddr)
	}

	// Get sender address from private key
	publicKey := e.privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("failed to get public key")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// Get nonce
	nonce, err := e.client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return "", fmt.Errorf("failed to get nonce: %w", err)
	}

	// Get gas price
	gasPrice, err := e.getGasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get gas price: %w", err)
	}

	// Determine if this is a native token or ERC20 transfer
	var tx *types.Transaction
	if tokenContract == "" {
		// Native token transfer (ETH, BNB, MATIC, etc.)
		tx, err = e.sendNativeToken(ctx, fromAddress, recipientAddr, amount, nonce, gasPrice)
	} else {
		// ERC20 token transfer
		tx, err = e.sendERC20Token(ctx, fromAddress, recipientAddr, tokenContract, amount, nonce, gasPrice)
	}

	if err != nil {
		return "", err
	}

	// Send transaction
	if err := e.client.SendTransaction(ctx, tx); err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	return tx.Hash().Hex(), nil
}

// sendNativeToken sends native blockchain tokens (ETH, BNB, etc.)
func (e *EVMDepositor) sendNativeToken(ctx context.Context, from common.Address, to string, amount string, nonce uint64, gasPrice *big.Int) (*types.Transaction, error) {
	toAddress := common.HexToAddress(to)

	// Parse amount (assuming it's in Ether/BNB/etc., convert to Wei)
	amountWei, err := parseAmount(amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	// Check balance
	balance, err := e.client.BalanceAt(ctx, from, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	if balance.Cmp(amountWei) < 0 {
		return nil, fmt.Errorf("insufficient balance: have %s wei, need %s wei", balance.String(), amountWei.String())
	}

	// Estimate gas limit if not provided
	gasLimit := uint64(21000) // Standard ETH transfer
	if e.network.GasLimit != nil {
		gasLimit = *e.network.GasLimit
	}

	// Create transaction
	tx := types.NewTransaction(
		nonce,
		toAddress,
		amountWei,
		gasLimit,
		gasPrice,
		nil,
	)

	// Sign transaction
	chainID := big.NewInt(e.network.ChainID)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), e.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	return signedTx, nil
}

// sendERC20Token sends ERC20 tokens
func (e *EVMDepositor) sendERC20Token(ctx context.Context, from common.Address, to string, tokenContract string, amount string, nonce uint64, gasPrice *big.Int) (*types.Transaction, error) {
	toAddress := common.HexToAddress(to)
	tokenAddress := common.HexToAddress(tokenContract)

	// Validate token contract address
	if !common.IsHexAddress(tokenContract) {
		return nil, fmt.Errorf("invalid token contract address: %s", tokenContract)
	}

	// Parse amount (assuming it's in token units, convert to smallest unit)
	// Note: This assumes 18 decimals. For production, you should query the token's decimals() function
	amountTokens, err := parseAmount(amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	// Get token balance
	balance, err := e.getERC20Balance(ctx, tokenAddress, from)
	if err != nil {
		return nil, fmt.Errorf("failed to get token balance: %w", err)
	}

	if balance.Cmp(amountTokens) < 0 {
		return nil, fmt.Errorf("insufficient token balance: have %s, need %s", balance.String(), amountTokens.String())
	}

	// Parse ERC20 ABI
	parsedABI, err := abi.JSON(strings.NewReader(erc20TransferABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ERC20 ABI: %w", err)
	}

	// Pack transfer function data
	data, err := parsedABI.Pack("transfer", toAddress, amountTokens)
	if err != nil {
		return nil, fmt.Errorf("failed to pack transfer data: %w", err)
	}

	// Estimate gas limit if not provided
	gasLimit := uint64(100000) // Typical ERC20 transfer
	if e.network.GasLimit != nil {
		gasLimit = *e.network.GasLimit
	} else {
		// Try to estimate gas
		msg := ethereum.CallMsg{
			From: from,
			To:   &tokenAddress,
			Data: data,
		}
		estimatedGas, err := e.client.EstimateGas(ctx, msg)
		if err == nil {
			gasLimit = estimatedGas * 120 / 100 // Add 20% buffer
		}
	}

	// Create transaction
	tx := types.NewTransaction(
		nonce,
		tokenAddress,
		big.NewInt(0), // No ETH value for ERC20 transfer
		gasLimit,
		gasPrice,
		data,
	)

	// Sign transaction
	chainID := big.NewInt(e.network.ChainID)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), e.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	return signedTx, nil
}

// getGasPrice returns the gas price to use for transactions
func (e *EVMDepositor) getGasPrice(ctx context.Context) (*big.Int, error) {
	// Use configured gas price if available
	if e.network.GasPrice != nil {
		return big.NewInt(*e.network.GasPrice), nil
	}

	// Otherwise, get current gas price from network
	gasPrice, err := e.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	return gasPrice, nil
}

// getERC20Balance gets the balance of an ERC20 token for an address
func (e *EVMDepositor) getERC20Balance(ctx context.Context, tokenAddress common.Address, account common.Address) (*big.Int, error) {
	// balanceOf(address) function signature
	balanceOfABI := `[{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"type":"function"}]`

	parsedABI, err := abi.JSON(strings.NewReader(balanceOfABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse balanceOf ABI: %w", err)
	}

	data, err := parsedABI.Pack("balanceOf", account)
	if err != nil {
		return nil, fmt.Errorf("failed to pack balanceOf data: %w", err)
	}

	msg := ethereum.CallMsg{
		To:   &tokenAddress,
		Data: data,
	}

	result, err := e.client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call balanceOf: %w", err)
	}

	balance := new(big.Int)
	balance.SetBytes(result)

	return balance, nil
}

// parseAmount converts a string amount to wei/smallest unit
// Assumes the amount is in the main unit (e.g., ETH, not wei) with up to 18 decimals
func parseAmount(amount string) (*big.Int, error) {
	// Parse as float and convert to wei (multiply by 10^18)
	amountFloat := new(big.Float)
	_, ok := amountFloat.SetString(amount)
	if !ok {
		return nil, fmt.Errorf("invalid amount format: %s", amount)
	}

	// Multiply by 10^18 to convert to wei
	weiPerEther := new(big.Float).SetInt(big.NewInt(1e18))
	amountWei := new(big.Float).Mul(amountFloat, weiPerEther)

	// Convert to big.Int
	result := new(big.Int)
	amountWei.Int(result)

	return result, nil
}

// GetTransactionInfo retrieves information about a transaction
func (e *EVMDepositor) GetTransactionInfo(txHash string) (map[string]interface{}, error) {
	ctx := context.Background()

	hash := common.HexToHash(txHash)

	// Get transaction
	tx, isPending, err := e.client.TransactionByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	// Get receipt
	receipt, err := e.client.TransactionReceipt(ctx, hash)
	if err != nil && !isPending {
		return nil, fmt.Errorf("failed to get transaction receipt: %w", err)
	}

	info := map[string]interface{}{
		"hash":      tx.Hash().Hex(),
		"nonce":     tx.Nonce(),
		"gas_price": tx.GasPrice().String(),
		"gas_limit": tx.Gas(),
		"to":        "",
		"value":     tx.Value().String(),
		"pending":   isPending,
	}

	if tx.To() != nil {
		info["to"] = tx.To().Hex()
	}

	if receipt != nil {
		info["block_number"] = receipt.BlockNumber.Uint64()
		info["gas_used"] = receipt.GasUsed
		info["status"] = receipt.Status
	}

	return info, nil
}

// Close closes the client connection
func (e *EVMDepositor) Close() {
	if e.client != nil {
		e.client.Close()
	}
}
