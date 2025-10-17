package deposit

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"near-swap/config"

	"github.com/gagliardetto/solana-go"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
)

// SolanaDepositor handles deposits on Solana blockchain
type SolanaDepositor struct {
	config     config.SolanaConfig
	client     *rpc.Client
	privateKey solana.PrivateKey
	publicKey  solana.PublicKey
}

// NewSolanaDepositor creates a new Solana depositor
func NewSolanaDepositor(cfg config.SolanaConfig) (*SolanaDepositor, error) {
	// Validate configuration
	if cfg.RPCUrl == "" {
		return nil, fmt.Errorf("RPC URL not configured for Solana")
	}
	if cfg.PrivateKey == "" {
		return nil, fmt.Errorf("private key not configured for Solana")
	}

	// Connect to Solana RPC
	client := rpc.New(cfg.RPCUrl)

	// Parse private key (Base58 encoded)
	privateKey, err := solana.PrivateKeyFromBase58(cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	publicKey := privateKey.PublicKey()

	return &SolanaDepositor{
		config:     cfg,
		client:     client,
		privateKey: privateKey,
		publicKey:  publicKey,
	}, nil
}

// SendDeposit sends a deposit to the specified address
// For native SOL, address is just the recipient
// For SPL tokens, address format is: "recipient|tokenMint"
func (s *SolanaDepositor) SendDeposit(address string, amount string) (string, error) {
	ctx := context.Background()

	// Parse address - check if it contains token mint address for SPL tokens
	parts := strings.Split(address, "|")
	recipientAddr := parts[0]
	var tokenMint string
	if len(parts) > 1 {
		tokenMint = parts[1]
	}

	// Validate recipient address
	recipient, err := solana.PublicKeyFromBase58(recipientAddr)
	if err != nil {
		return "", fmt.Errorf("invalid recipient address: %w", err)
	}

	// Determine if this is a native SOL or SPL token transfer
	var signature solana.Signature
	if tokenMint == "" {
		// Native SOL transfer
		signature, err = s.sendNativeSOL(ctx, recipient, amount)
	} else {
		// SPL token transfer
		signature, err = s.sendSPLToken(ctx, recipient, tokenMint, amount)
	}

	if err != nil {
		return "", err
	}

	return signature.String(), nil
}

// sendNativeSOL sends native SOL tokens
func (s *SolanaDepositor) sendNativeSOL(ctx context.Context, recipient solana.PublicKey, amount string) (solana.Signature, error) {
	// Parse amount (in SOL, convert to lamports: 1 SOL = 1e9 lamports)
	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("invalid amount: %w", err)
	}

	// Convert to lamports
	lamports := uint64(amountFloat * 1e9)

	// Get balance
	balance, err := s.getBalance(ctx)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to get balance: %w", err)
	}

	// Check if we have enough balance (including for fees)
	// Solana fees are typically 5000 lamports per signature
	minRequired := lamports + 5000
	if balance < minRequired {
		balanceSOL := float64(balance) / 1e9
		requiredSOL := float64(minRequired) / 1e9
		return solana.Signature{}, fmt.Errorf("insufficient balance: have %.9f SOL, need %.9f SOL (including fees)", balanceSOL, requiredSOL)
	}

	// Get recent blockhash
	recent, err := s.client.GetRecentBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to get recent blockhash: %w", err)
	}

	// Create transfer instruction
	instruction := system.NewTransferInstruction(
		lamports,
		s.publicKey,
		recipient,
	).Build()

	// Create transaction
	tx, err := solana.NewTransaction(
		[]solana.Instruction{instruction},
		recent.Value.Blockhash,
		solana.TransactionPayer(s.publicKey),
	)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Sign transaction
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(s.publicKey) {
			return &s.privateKey
		}
		return nil
	})
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	opts := rpc.TransactionOpts{
		SkipPreflight:       s.config.SkipPreflight,
		PreflightCommitment: s.getCommitment(),
	}

	sig, err := s.client.SendTransactionWithOpts(ctx, tx, opts)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to send transaction: %w", err)
	}

	return sig, nil
}

// sendSPLToken sends SPL tokens
func (s *SolanaDepositor) sendSPLToken(ctx context.Context, recipient solana.PublicKey, tokenMintStr string, amount string) (solana.Signature, error) {
	// Parse token mint address
	tokenMint, err := solana.PublicKeyFromBase58(tokenMintStr)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("invalid token mint address: %w", err)
	}

	// Parse amount
	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("invalid amount: %w", err)
	}

	// Get token decimals
	decimals, err := s.getTokenDecimals(ctx, tokenMint)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to get token decimals: %w", err)
	}

	// Convert to token smallest unit
	multiplier := uint64(1)
	for i := uint8(0); i < decimals; i++ {
		multiplier *= 10
	}
	tokenAmount := uint64(amountFloat * float64(multiplier))

	// Get source token account (our token account)
	sourceTokenAccount, err := s.getAssociatedTokenAddress(s.publicKey, tokenMint)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to get source token account: %w", err)
	}

	// Check token balance
	balance, err := s.getTokenBalance(ctx, sourceTokenAccount)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to get token balance: %w", err)
	}

	if balance < tokenAmount {
		balanceFormatted := float64(balance) / float64(multiplier)
		amountFormatted := float64(tokenAmount) / float64(multiplier)
		return solana.Signature{}, fmt.Errorf("insufficient token balance: have %f, need %f", balanceFormatted, amountFormatted)
	}

	// Get or create destination token account
	destTokenAccount, err := s.getAssociatedTokenAddress(recipient, tokenMint)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to get destination token account: %w", err)
	}

	// Check if destination token account exists
	destAccountExists, err := s.accountExists(ctx, destTokenAccount)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to check destination account: %w", err)
	}

	// Get recent blockhash
	recent, err := s.client.GetRecentBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to get recent blockhash: %w", err)
	}

	// Build instructions
	instructions := []solana.Instruction{}

	// Create associated token account if it doesn't exist
	if !destAccountExists {
		createAccountIx := associatedtokenaccount.NewCreateInstruction(
			s.publicKey,      // payer
			recipient,        // wallet
			tokenMint,        // mint
		).Build()
		instructions = append(instructions, createAccountIx)
	}

	// Create transfer instruction
	transferIx := token.NewTransferInstruction(
		tokenAmount,
		sourceTokenAccount,
		destTokenAccount,
		s.publicKey,
		[]solana.PublicKey{}, // no multisig
	).Build()
	instructions = append(instructions, transferIx)

	// Create transaction
	tx, err := solana.NewTransaction(
		instructions,
		recent.Value.Blockhash,
		solana.TransactionPayer(s.publicKey),
	)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Sign transaction
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(s.publicKey) {
			return &s.privateKey
		}
		return nil
	})
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	opts := rpc.TransactionOpts{
		SkipPreflight:       s.config.SkipPreflight,
		PreflightCommitment: s.getCommitment(),
	}

	sig, err := s.client.SendTransactionWithOpts(ctx, tx, opts)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to send transaction: %w", err)
	}

	return sig, nil
}

// getBalance returns the SOL balance in lamports
func (s *SolanaDepositor) getBalance(ctx context.Context) (uint64, error) {
	balance, err := s.client.GetBalance(ctx, s.publicKey, rpc.CommitmentFinalized)
	if err != nil {
		return 0, fmt.Errorf("failed to get balance: %w", err)
	}
	return balance.Value, nil
}

// getTokenBalance returns the token balance for a token account
func (s *SolanaDepositor) getTokenBalance(ctx context.Context, tokenAccount solana.PublicKey) (uint64, error) {
	accountInfo, err := s.client.GetTokenAccountBalance(ctx, tokenAccount, rpc.CommitmentFinalized)
	if err != nil {
		return 0, fmt.Errorf("failed to get token balance: %w", err)
	}

	amount, err := strconv.ParseUint(accountInfo.Value.Amount, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse token balance: %w", err)
	}

	return amount, nil
}

// getTokenDecimals gets the decimals for a token mint
func (s *SolanaDepositor) getTokenDecimals(ctx context.Context, mint solana.PublicKey) (uint8, error) {
	accountInfo, err := s.client.GetAccountInfo(ctx, mint)
	if err != nil {
		return 0, fmt.Errorf("failed to get mint account info: %w", err)
	}

	if accountInfo.Value == nil {
		return 0, fmt.Errorf("mint account not found")
	}

	// Parse mint data to get decimals
	// The decimals field is at byte offset 44 in the mint account data
	data := accountInfo.Value.Data.GetBinary()
	if len(data) < 45 {
		return 0, fmt.Errorf("invalid mint account data")
	}

	decimals := data[44]
	return decimals, nil
}

// getAssociatedTokenAddress derives the associated token account address
func (s *SolanaDepositor) getAssociatedTokenAddress(wallet solana.PublicKey, mint solana.PublicKey) (solana.PublicKey, error) {
	addr, _, err := solana.FindAssociatedTokenAddress(wallet, mint)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("failed to derive associated token address: %w", err)
	}
	return addr, nil
}

// accountExists checks if an account exists on-chain
func (s *SolanaDepositor) accountExists(ctx context.Context, account solana.PublicKey) (bool, error) {
	accountInfo, err := s.client.GetAccountInfo(ctx, account)
	if err != nil {
		// If the error indicates account doesn't exist, return false
		if strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, err
	}

	return accountInfo.Value != nil, nil
}

// getCommitment returns the commitment level from config
func (s *SolanaDepositor) getCommitment() rpc.CommitmentType {
	switch strings.ToLower(s.config.Commitment) {
	case "finalized":
		return rpc.CommitmentFinalized
	case "confirmed":
		return rpc.CommitmentConfirmed
	case "processed":
		return rpc.CommitmentProcessed
	default:
		return rpc.CommitmentConfirmed
	}
}

// GetTransactionInfo retrieves information about a transaction
func (s *SolanaDepositor) GetTransactionInfo(txSignature string) (map[string]interface{}, error) {
	ctx := context.Background()

	sig, err := solana.SignatureFromBase58(txSignature)
	if err != nil {
		return nil, fmt.Errorf("invalid transaction signature: %w", err)
	}

	txInfo, err := s.client.GetTransaction(ctx, sig, &rpc.GetTransactionOpts{
		Encoding: solana.EncodingBase64,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	info := map[string]interface{}{
		"signature": txSignature,
		"slot":      txInfo.Slot,
	}

	if txInfo.Meta != nil {
		info["fee"] = txInfo.Meta.Fee
		info["err"] = txInfo.Meta.Err

		if txInfo.BlockTime != nil {
			info["block_time"] = *txInfo.BlockTime
		}
	}

	return info, nil
}

// Close closes any open connections
func (s *SolanaDepositor) Close() {
	// The Solana RPC client doesn't require explicit cleanup
}
