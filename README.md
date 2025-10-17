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
# Or use a .env file
echo "NEAR_SWAP_JWT_TOKEN=your-jwt-token-here" > .env
```

### Option 2: Configuration File

Create a `.near-swap.yaml` file in your home directory or current directory:

```bash
# Copy the example config
cp .near-swap.yaml.example ~/.near-swap.yaml

# Edit with your values
nano ~/.near-swap.yaml
```

Example configuration:

```yaml
jwt_token: "your-jwt-token-here"
base_url: "https://1click.chaindefuser.com"

# Optional: Configure auto-deposit for Bitcoin
auto_deposit:
  enabled: true
  bitcoin:
    enabled: true
    cli_path: "bitcoin-cli"
    # wallet: "default"
    # fee_rate: 1
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

## Auto-Deposit Feature

The CLI supports automatically sending your deposit for supported blockchains:

### Supported Blockchains
- **Bitcoin** (BTC) - via `bitcoin-cli`
- **Monero** (XMR) - via `monero-wallet-rpc`
- **Zcash** (ZEC) - via `zcash-cli`
- More chains coming soon!

### Setup Auto-Deposit for Bitcoin

1. Ensure `bitcoin-cli` is installed and configured
2. Enable auto-deposit in your `.near-swap.yaml`:

```yaml
auto_deposit:
  enabled: true
  bitcoin:
    enabled: true
    cli_path: "bitcoin-cli"  # Path to bitcoin-cli (default uses PATH)
    wallet: "default"        # Optional: wallet name
    fee_rate: 1              # Optional: fee rate in sat/vB
```

3. Use the `--auto-deposit` flag:

```bash
near-swap swap 0.01 BTC to USDC \
  --from-chain btc \
  --to-chain near \
  --recipient your.near \
  --refund-to <your-btc-address> \
  --auto-deposit
```

The CLI will:
- Verify bitcoin-cli connectivity
- Check your wallet balance
- Confirm the deposit with you
- Send the transaction
- Display the transaction ID

### Setup Auto-Deposit for Monero

1. Ensure `monero-wallet-rpc` is installed and running
2. Enable auto-deposit in your `.near-swap.yaml`:

```yaml
auto_deposit:
  enabled: true
  monero:
    enabled: true
    host: "127.0.0.1"      # RPC host
    port: 18082            # RPC port (default: 18082)
    username: "user"       # Optional: RPC username
    password: "pass"       # Optional: RPC password
    account_index: 0       # Optional: account index
    priority: 0            # Optional: tx priority (0-4)
```

3. Start monero-wallet-rpc with your wallet:

```bash
monero-wallet-rpc --rpc-bind-port 18082 --wallet-file /path/to/wallet --password yourpassword
```

4. Use the `--auto-deposit` flag:

```bash
near-swap swap 0.1 XMR to USDC \
  --from-chain xmr \
  --to-chain near \
  --recipient your.near \
  --refund-to <your-xmr-address> \
  --auto-deposit
```

The CLI will:
- Verify monero-wallet-rpc connectivity
- Check your wallet balance
- Confirm the deposit with you
- Send the transaction
- Display the transaction hash

### Setup Auto-Deposit for Zcash

1. Ensure `zcash-cli` is installed and configured
2. Enable auto-deposit in your `.near-swap.yaml`:

```yaml
auto_deposit:
  enabled: true
  zcash:
    enabled: true
    cli_path: "zcash-cli"    # Path to zcash-cli (default uses PATH)
    cli_args: []             # Optional: custom args like ["-testnet"]
```

3. Use the `--auto-deposit` flag:

```bash
near-swap swap 0.5 ZEC to USDC \
  --from-chain zec \
  --to-chain near \
  --recipient your.near \
  --refund-to <your-zec-address> \
  --auto-deposit
```

The CLI will:
- Verify zcash-cli connectivity
- Check your wallet balance
- Confirm the deposit with you
- Send the transaction
- Display the transaction ID

## How It Works

1. **Quote Generation**: The CLI fetches a swap quote from the 1Click API
2. **Deposit Address**: You receive a unique deposit address for your swap
3. **Token Transfer**: Send your tokens to the deposit address (manually or auto)
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
├── main.go                      # CLI entry point
├── .near-swap.yaml.example      # Sample configuration file
├── cmd/
│   ├── root.go                 # Root command
│   ├── swap.go                 # Swap command with auto-deposit
│   ├── tokens.go               # List tokens command
│   └── status.go               # Status check command
├── pkg/
│   ├── client/
│   │   └── oneclick.go         # 1Click API client wrapper
│   ├── parser/
│   │   └── command.go          # Command parser
│   ├── deposit/
│   │   ├── deposit.go          # Deposit manager
│   │   └── bitcoin.go          # Bitcoin auto-deposit
│   └── types/
│       └── swap.go             # Type definitions
├── config/
│   └── config.go               # Configuration management
└── go.mod                      # Dependencies
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
# Or create .env file with: NEAR_SWAP_JWT_TOKEN=your-token
```

### "Token not found" error

The token symbol you're trying to swap might not be supported or the name might be different. Use the `list-tokens` command to see all available tokens:

```bash
near-swap list-tokens --symbol <your-token>
```

### "refundTo is not valid" error

For cross-chain swaps, the refund address must be valid for the **source chain**:

```bash
# Correct: refund-to is a Solana address for SOL → USDC swap
near-swap swap 1 SOL to USDC \
  --from-chain sol \
  --to-chain near \
  --recipient your.near \
  --refund-to <valid-solana-address>
```

### Auto-deposit errors

**Bitcoin errors:**

**"bitcoin-cli not accessible"**:
- Ensure `bitcoin-cli` is installed and in your PATH
- Verify Bitcoin Core is running
- Check RPC credentials if using authentication

**"insufficient balance"**:
- Check your Bitcoin wallet balance: `bitcoin-cli getbalance`
- Ensure you have enough for the amount + transaction fees

**Monero errors:**

**"monero-wallet-rpc not accessible"**:
- Ensure `monero-wallet-rpc` is running
- Verify the host and port in your configuration
- Check RPC authentication credentials if enabled
- Test connectivity: `curl http://127.0.0.1:18082/json_rpc -d '{"jsonrpc":"2.0","id":"0","method":"get_version"}' -H 'Content-Type: application/json'`

**"insufficient balance"**:
- Check your Monero wallet unlocked balance (funds must be unlocked to spend)
- Ensure you have enough XMR for the amount + transaction fees
- Note: Monero has a 10-block confirmation requirement before funds are spendable

**Zcash errors:**

**"zcash-cli not accessible"**:
- Ensure `zcash-cli` is installed and in your PATH
- Verify Zcash daemon is running
- Check RPC credentials if using authentication
- Test with: `zcash-cli getblockchaininfo`

**"insufficient balance"**:
- Check your Zcash wallet balance: `zcash-cli getbalance`
- Ensure you have enough for the amount + transaction fees

**"auto-deposit not enabled"**:
- Check your `.near-swap.yaml` configuration
- Ensure `auto_deposit.enabled: true` and the respective chain is enabled
- For Bitcoin: `auto_deposit.bitcoin.enabled: true`
- For Monero: `auto_deposit.monero.enabled: true`
- For Zcash: `auto_deposit.zcash.enabled: true`

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
