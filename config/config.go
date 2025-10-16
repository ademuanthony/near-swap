package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	JWTToken string
	BaseURL  string
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

	// Read from environment variables
	viper.SetEnvPrefix("NEAR_SWAP")
	viper.AutomaticEnv()

	// Read config file (optional)
	_ = viper.ReadInConfig()

	// Create config struct
	cfg := &Config{
		JWTToken: viper.GetString("jwt_token"),
		BaseURL:  viper.GetString("base_url"),
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
