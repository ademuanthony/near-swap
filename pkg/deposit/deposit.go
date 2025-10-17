package deposit

import (
	"fmt"
	"strings"

	"near-swap/config"
)

// Depositor interface for blockchain-specific depositors
type Depositor interface {
	SendDeposit(address string, amount string) (string, error)
}

// Manager handles auto-deposit for different blockchains
type Manager struct {
	config config.AutoDepositConfig
}

// NewManager creates a new deposit manager
func NewManager(cfg config.AutoDepositConfig) *Manager {
	return &Manager{
		config: cfg,
	}
}

// IsEnabled returns whether auto-deposit is enabled globally
func (m *Manager) IsEnabled() bool {
	return m.config.Enabled
}

// IsEnabledForChain returns whether auto-deposit is enabled for a specific blockchain
func (m *Manager) IsEnabledForChain(chain string) bool {
	if !m.config.Enabled {
		return false
	}

	chain = strings.ToLower(chain)
	switch chain {
	case "btc", "bitcoin":
		return m.config.Bitcoin.Enabled
	case "xmr", "monero":
		return m.config.Monero.Enabled
	case "zec", "zcash":
		return m.config.Zcash.Enabled
	// Add more chains here as they're implemented
	default:
		return false
	}
}

// SendDeposit sends a deposit for the specified chain
func (m *Manager) SendDeposit(chain, address, amount string) (string, error) {
	if !m.IsEnabled() {
		return "", fmt.Errorf("auto-deposit is not enabled in configuration")
	}

	if !m.IsEnabledForChain(chain) {
		return "", fmt.Errorf("auto-deposit is not enabled for chain: %s", chain)
	}

	chain = strings.ToLower(chain)
	switch chain {
	case "btc", "bitcoin":
		return m.sendBitcoinDeposit(address, amount)
	case "xmr", "monero":
		return m.sendMoneroDeposit(address, amount)
	case "zec", "zcash":
		return m.sendZcashDeposit(address, amount)
	// Add more chains here as they're implemented
	default:
		return "", fmt.Errorf("auto-deposit not supported for chain: %s", chain)
	}
}

// sendBitcoinDeposit sends a Bitcoin deposit
func (m *Manager) sendBitcoinDeposit(address, amount string) (string, error) {
	depositor := NewBitcoinDepositor(m.config.Bitcoin)
	return depositor.SendDeposit(address, amount)
}

// sendMoneroDeposit sends a Monero deposit
func (m *Manager) sendMoneroDeposit(address, amount string) (string, error) {
	depositor := NewMoneroDepositor(m.config.Monero)
	return depositor.SendDeposit(address, amount)
}

// sendZcashDeposit sends a Zcash deposit
func (m *Manager) sendZcashDeposit(address, amount string) (string, error) {
	depositor := NewZcashDepositor(m.config.Zcash)
	return depositor.SendDeposit(address, amount)
}

// GetSupportedChains returns a list of chains that support auto-deposit
func (m *Manager) GetSupportedChains() []string {
	supported := make([]string, 0)

	if m.config.Bitcoin.Enabled {
		supported = append(supported, "bitcoin")
	}

	if m.config.Monero.Enabled {
		supported = append(supported, "monero")
	}

	if m.config.Zcash.Enabled {
		supported = append(supported, "zcash")
	}

	// Add more chains as they're implemented

	return supported
}
