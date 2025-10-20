package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// BitcoinConfig holds Bitcoin-specific configuration
type BitcoinConfig struct {
	Enabled  bool     `mapstructure:"enabled"`
	CLIPath  string   `mapstructure:"cli_path"`
	CLIArgs  []string `mapstructure:"cli_args"`
	Wallet   string   `mapstructure:"wallet"`
	FeeRate  float64  `mapstructure:"fee_rate"`
}

// MoneroConfig holds Monero-specific configuration for auto-deposit
type MoneroConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	AccountIndex uint32 `mapstructure:"account_index"`
	Priority     uint32 `mapstructure:"priority"`
	UnlockTime   uint64 `mapstructure:"unlock_time"`
}

// ZcashConfig holds Zcash-specific configuration
type ZcashConfig struct {
	Enabled  bool     `mapstructure:"enabled"`
	CLIPath  string   `mapstructure:"cli_path"`
	CLIArgs  []string `mapstructure:"cli_args"`
}

// EVMConfig holds EVM-specific configuration for auto-deposit
type EVMConfig struct {
	Enabled    bool              `mapstructure:"enabled"`
	Networks   map[string]EVMNetwork `mapstructure:"networks"`
}

// EVMNetwork holds configuration for a specific EVM network
type EVMNetwork struct {
	RPCUrl        string  `mapstructure:"rpc_url"`
	ChainID       int64   `mapstructure:"chain_id"`
	PrivateKeyEnv string  `mapstructure:"private_key_env"` // Environment variable name containing the private key
	PrivateKey    string  // Resolved private key value (populated after loading config)
	GasPrice      *int64  `mapstructure:"gas_price"`   // Optional: wei per gas unit
	GasLimit      *uint64 `mapstructure:"gas_limit"`   // Optional: max gas for transaction
}

// SolanaConfig holds Solana-specific configuration for auto-deposit
type SolanaConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	RPCUrl        string `mapstructure:"rpc_url"`
	WSUrl         string `mapstructure:"ws_url"`             // Optional: WebSocket URL
	PrivateKeyEnv string `mapstructure:"private_key_env"`    // Environment variable name containing the private key
	PrivateKey    string                                     // Resolved private key value (populated after loading config)
	Commitment    string `mapstructure:"commitment"`         // Commitment level: finalized, confirmed, processed
	SkipPreflight bool   `mapstructure:"skip_preflight"`     // Skip preflight transaction checks
}

// AutoDepositConfig holds auto-deposit configuration
type AutoDepositConfig struct {
	Enabled bool          `mapstructure:"enabled"`
	Bitcoin BitcoinConfig `mapstructure:"bitcoin"`
	Monero  MoneroConfig  `mapstructure:"monero"`
	Zcash   ZcashConfig   `mapstructure:"zcash"`
	EVM     EVMConfig     `mapstructure:"evm"`
	Solana  SolanaConfig  `mapstructure:"solana"`
}

// Config holds the application configuration
type Config struct {
	JWTToken        string            `mapstructure:"jwt_token"`
	BaseURL         string            `mapstructure:"base_url"`
	DefaultRecipient string           `mapstructure:"default_recipient"`
	DefaultRefundTo  string           `mapstructure:"default_refund_to"`
	AutoDeposit     AutoDepositConfig `mapstructure:"auto_deposit"`
	OutputFormat    string            `mapstructure:"output_format"`
	Verbose         bool              `mapstructure:"verbose"`
	AutoConfirm     bool              `mapstructure:"auto_confirm"`
	Timeout         int               `mapstructure:"timeout"`
	MaxRetries      int               `mapstructure:"max_retries"`
	PlanStoragePath string            `mapstructure:"plan_storage_path"`
}

var globalConfig *Config

// resolvePrivateKeys resolves environment variable references to actual private key values
func resolvePrivateKeys(cfg *Config) error {
	// Resolve EVM network private keys
	for networkName, network := range cfg.AutoDeposit.EVM.Networks {
		if network.PrivateKeyEnv != "" {
			privateKey := os.Getenv(network.PrivateKeyEnv)
			if privateKey == "" {
				return fmt.Errorf("environment variable '%s' for EVM network '%s' is not set or empty", network.PrivateKeyEnv, networkName)
			}
			network.PrivateKey = privateKey
			cfg.AutoDeposit.EVM.Networks[networkName] = network
		}
	}

	// Resolve Solana private key
	if cfg.AutoDeposit.Solana.PrivateKeyEnv != "" {
		privateKey := os.Getenv(cfg.AutoDeposit.Solana.PrivateKeyEnv)
		if privateKey == "" {
			return fmt.Errorf("environment variable '%s' for Solana is not set or empty", cfg.AutoDeposit.Solana.PrivateKeyEnv)
		}
		cfg.AutoDeposit.Solana.PrivateKey = privateKey
	}

	return nil
}

// Load reads configuration from environment variables and config file
func Load() (*Config, error) {
	viper.SetConfigName(".near-swap")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME")
	viper.AddConfigPath(".")

	// Set default values
	viper.SetDefault("base_url", "https://1click.chaindefuser.com")
	viper.SetDefault("output_format", "text")
	viper.SetDefault("verbose", false)
	viper.SetDefault("auto_confirm", false)
	viper.SetDefault("timeout", 30)
	viper.SetDefault("max_retries", 3)
	viper.SetDefault("plan_storage_path", "") // Empty means use default (~/.near-swap-plans.json)
	viper.SetDefault("auto_deposit.enabled", false)
	viper.SetDefault("auto_deposit.bitcoin.enabled", false)
	viper.SetDefault("auto_deposit.bitcoin.cli_path", "bitcoin-cli")
	viper.SetDefault("auto_deposit.monero.enabled", false)
	viper.SetDefault("auto_deposit.monero.host", "127.0.0.1")
	viper.SetDefault("auto_deposit.monero.port", 18082)
	viper.SetDefault("auto_deposit.monero.account_index", 0)
	viper.SetDefault("auto_deposit.monero.priority", 0)
	viper.SetDefault("auto_deposit.zcash.enabled", false)
	viper.SetDefault("auto_deposit.zcash.cli_path", "zcash-cli")
	viper.SetDefault("auto_deposit.evm.enabled", false)
	viper.SetDefault("auto_deposit.evm.networks", map[string]interface{}{})
	viper.SetDefault("auto_deposit.solana.enabled", false)
	viper.SetDefault("auto_deposit.solana.rpc_url", "https://api.mainnet-beta.solana.com")
	viper.SetDefault("auto_deposit.solana.commitment", "confirmed")
	viper.SetDefault("auto_deposit.solana.skip_preflight", false)

	// Read from environment variables
	viper.SetEnvPrefix("NEAR_SWAP")
	viper.AutomaticEnv()

	// Read config file (optional)
	_ = viper.ReadInConfig()

	// Create config struct
	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Resolve private key environment variables
	if err := resolvePrivateKeys(cfg); err != nil {
		return nil, fmt.Errorf("failed to resolve private keys: %w", err)
	}

	// Validate JWT token
	if cfg.JWTToken == "" {
		return nil, fmt.Errorf("JWT token not found. Please set NEAR_SWAP_JWT_TOKEN environment variable or create a .near-swap.yaml config file")
	}

	globalConfig = cfg
	return cfg, nil
}

// Get returns the global configuration
func Get() *Config {
	if globalConfig == nil {
		cfg, err := Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
			os.Exit(1)
		}
		return cfg
	}
	return globalConfig
}

// Set updates the global configuration
func Set(cfg *Config) {
	globalConfig = cfg
}
