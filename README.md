# Near-Swap CLI

A command-line interface for cross-chain token swaps using the NEAR Intents 1Click API.

## Features

- 🔄 **Cross-chain swaps**: Swap tokens across different blockchains
- 📝 **Natural language commands**: Simple syntax like "swap 1 SOL to USDC"
- 🎨 **Rich terminal output**: Colorized output with progress indicators
- 📊 **Token discovery**: List all supported tokens across chains
- 🔍 **Status tracking**: Monitor swap execution in real-time
- 🌐 **Multi-chain support**: Works with multiple blockchains via NEAR Intents

## Installation

### Prerequisites

- Go 1.23.0 or higher
- NEAR Intents 1Click API JWT token (required for authentication)

### Build from Source

```bash
# Clone the repository or navigate to the project directory
cd near-swap

# Build the CLI
go build -o near-swap

# Optional: Install globally
go install
```

## Configuration

Before using the CLI, you need to set up your JWT token for authentication:

### Option 1: Environment Variable

```bash
export NEAR_SWAP_JWT_TOKEN="your-jwt-token-here"
```

### Option 2: Configuration File

Create a `.near-swap.yaml` file in your home directory or current directory:

```yaml
jwt_token: "your-jwt-token-here"
base_url: "https://1click.chaindefuser.com"  # optional, this is the default
```

### Obtaining a JWT Token

To get a JWT token for the NEAR Intents 1Click API, visit:
https://docs.near-intents.org/near-intents/integration/distribution-channels/1click-api

**Note**: Without a JWT token, you'll incur a 0.1% fee (10 basis points) on all swaps.

## Usage

### Swap Tokens

Perform a cross-chain token swap with natural language syntax.

**IMPORTANT**:
- You must specify a `--recipient` address (where you'll receive the swapped tokens)
- For cross-chain swaps, you should also specify a `--refund-to` address on the source chain (where refunds go if the swap fails)
- Both addresses must be valid for their respective blockchains

```bash
# Cross-chain swap from Solana to NEAR
near-swap swap 1 SOL to USDC \
  --from-chain sol \
  --to-chain near \
  --recipient your-address.near \
  --refund-to <your-solana-address>

# Swap on the same chain (refund address can be same as recipient)
near-swap swap 100 USDC to ETH \
  --from-chain eth \
  --to-chain eth \
  --recipient 0x1234... \
  --refund-to 0x1234...

# Skip confirmation prompt
near-swap swap 1 SOL to USDC \
  --from-chain sol \
  --to-chain near \
  --recipient your-address.near \
  --refund-to <your-solana-address> \
  --yes
```

### List Supported Tokens

View all tokens supported by the 1Click API:

```bash
# List all tokens
near-swap list-tokens

# Filter by blockchain
near-swap list-tokens --chain solana

# Filter by symbol
near-swap list-tokens --symbol USDC

# Get JSON output
near-swap list-tokens --json
```

### Check Swap Status

Monitor the status of a swap using its deposit address:

```bash
# Check status once
near-swap status <deposit-address>

# Watch status continuously (polls every 5 seconds)
near-swap status <deposit-address> --watch

# Custom polling interval (in seconds)
near-swap status <deposit-address> --watch --interval 10

# Get JSON output
near-swap status <deposit-address> --json
```

## How It Works

1. **Quote Generation**: The CLI fetches a swap quote from the 1Click API
2. **Deposit Address**: You receive a unique deposit address for your swap
3. **Token Transfer**: Send your tokens to the deposit address
4. **Solver Network**: The NEAR Intents solver network competes to fulfill your swap
5. **Execution**: The best solution is executed and tokens are delivered to your destination address

## Swap Flow Example

```bash
$ near-swap swap 1 SOL to USDC

 Fetching quote...

============================================================
                     SWAP QUOTE
============================================================

  Deposit Address:   So11111...ABC123
  From:              1.00 SOL
  To:                ~150.25 USDC
  Estimated Time:    120 seconds

============================================================

Proceed with swap? (y/N): y

============================================================
                 DEPOSIT INSTRUCTIONS
============================================================

To complete the swap, send 1.00 SOL to:

  So11111...ABC123

============================================================

You can monitor the swap status using:
  near-swap status So11111...ABC123
```

## Global Flags

- `--verbose, -v`: Enable verbose output for debugging
- `--json, -j`: Output results in JSON format
- `--help, -h`: Show help information
- `--version`: Show version information

## Project Structure

```
near-swap/
├── main.go              # CLI entry point
├── cmd/
│   ├── root.go         # Root command
│   ├── swap.go         # Swap command
│   ├── tokens.go       # List tokens command
│   └── status.go       # Status check command
├── pkg/
│   ├── client/
│   │   └── oneclick.go # 1Click API client wrapper
│   ├── parser/
│   │   └── command.go  # Command parser
│   └── types/
│       └── swap.go     # Type definitions
├── config/
│   └── config.go       # Configuration management
└── go.mod              # Dependencies
```

## Dependencies

- [github.com/defuse-protocol/one-click-sdk-go](https://github.com/defuse-protocol/one-click-sdk-go) - 1Click API SDK
- [github.com/spf13/cobra](https://github.com/spf13/cobra) - CLI framework
- [github.com/spf13/viper](https://github.com/spf13/viper) - Configuration management
- [github.com/fatih/color](https://github.com/fatih/color) - Terminal colors
- [github.com/briandowns/spinner](https://github.com/briandowns/spinner) - Progress indicators

## Troubleshooting

### "JWT token not found" error

Make sure you've set your JWT token either as an environment variable or in a config file:

```bash
export NEAR_SWAP_JWT_TOKEN="your-token-here"
```

### "Token not found" error

The token symbol you're trying to swap might not be supported or the name might be different. Use the `list-tokens` command to see all available tokens:

```bash
near-swap list-tokens --symbol <your-token>
```

### Swap not completing

Check the status of your swap:

```bash
near-swap status <deposit-address> --watch
```

Common reasons for delays:
- Blockchain confirmation times
- Network congestion
- Insufficient liquidity

## Contributing

Contributions are welcome! Feel free to open issues or submit pull requests.

## License

This project uses the MIT License.

## Resources

- [NEAR Intents Documentation](https://docs.near-intents.org/)
- [1Click API Documentation](https://docs.near-intents.org/near-intents/integration/distribution-channels/1click-api)
- [NEAR Protocol](https://near.org/)

## Support

For issues and questions:
- Check the troubleshooting section above
- Review the [NEAR Intents documentation](https://docs.near-intents.org/)
- Open an issue in this repository
