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

// AutoDepositConfig holds auto-deposit configuration
type AutoDepositConfig struct {
	Enabled bool          `mapstructure:"enabled"`
	Bitcoin BitcoinConfig `mapstructure:"bitcoin"`
	Monero  MoneroConfig  `mapstructure:"monero"`
	Zcash   ZcashConfig   `mapstructure:"zcash"`
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
}

var globalConfig *Config

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
