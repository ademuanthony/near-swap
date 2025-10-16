# Near-Swap Setup Guide

## Quick Start

### 1. Build the CLI

```bash
cd near-swap
go build -o near-swap
```

### 2. Set up Configuration

Copy the example config:

```bash
cp .near-swap.yaml.example ~/.near-swap.yaml
```

Edit `~/.near-swap.yaml` with your settings:

```yaml
# Required: Your JWT token
jwt_token: "your-actual-jwt-token-here"

# Optional: Enable auto-deposit for Bitcoin
auto_deposit:
  enabled: true
  bitcoin:
    enabled: true
    cli_path: "bitcoin-cli"
    # wallet: "mywallet"
    # fee_rate: 1
```

### 3. Test the Setup

```bash
# List available tokens
./near-swap list-tokens

# Test a quote (won't execute)
./near-swap swap 1 SOL to USDC \
  --from-chain sol \
  --to-chain near \
  --recipient test.near \
  --refund-to <your-solana-address>
```

## Auto-Deposit Setup (Bitcoin)

### Prerequisites

1. **Bitcoin Core installed and running**
   ```bash
   # Check if bitcoin-cli is accessible
   bitcoin-cli getblockchaininfo
   ```

2. **Wallet with sufficient balance**
   ```bash
   # Check balance
   bitcoin-cli getbalance
   ```

### Configuration

Add to your `.near-swap.yaml`:

```yaml
auto_deposit:
  enabled: true
  bitcoin:
    enabled: true
    cli_path: "bitcoin-cli"           # Path to bitcoin-cli
    cli_args: []                       # Optional: custom args like ["-testnet"]
    wallet: "default"                  # Optional: wallet name
    fee_rate: 1                        # Optional: fee rate in sat/vB
```

### Using Auto-Deposit

```bash
./near-swap swap 0.01 BTC to USDC \
  --from-chain btc \
  --to-chain near \
  --recipient your.near \
  --refund-to <your-btc-address> \
  --auto-deposit
```

The CLI will:
1. ✓ Generate a swap quote
2. ✓ Show deposit details
3. ✓ Verify bitcoin-cli connectivity
4. ✓ Check wallet balance
5. ✓ Ask for confirmation
6. ✓ Send the Bitcoin transaction
7. ✓ Display transaction ID

## Auto-Deposit Setup (Monero)

### Prerequisites

1. **Monero wallet-rpc installed**
   ```bash
   # Check if monero-wallet-rpc is accessible
   which monero-wallet-rpc
   ```

2. **Wallet with sufficient unlocked balance**
   - Funds must be confirmed for at least 10 blocks to be spendable

### Configuration

Add to your `.near-swap.yaml`:

```yaml
auto_deposit:
  enabled: true
  monero:
    enabled: true
    host: "127.0.0.1"              # RPC host (default: 127.0.0.1)
    port: 18082                    # RPC port (default: 18082)
    username: "user"               # Optional: RPC username
    password: "pass"               # Optional: RPC password
    account_index: 0               # Optional: account index (default: 0)
    priority: 0                    # Optional: priority 0-4 (0=default)
    unlock_time: 0                 # Optional: unlock time in blocks
```

### Starting monero-wallet-rpc

Before using auto-deposit, start monero-wallet-rpc with your wallet:

```bash
# Basic usage (no authentication)
monero-wallet-rpc \
  --rpc-bind-port 18082 \
  --wallet-file /path/to/your/wallet \
  --password yourwalletpassword \
  --daemon-address node.moneroworld.com:18089

# With RPC authentication (recommended)
monero-wallet-rpc \
  --rpc-bind-port 18082 \
  --wallet-file /path/to/your/wallet \
  --password yourwalletpassword \
  --rpc-login user:pass \
  --daemon-address node.moneroworld.com:18089

# For testnet
monero-wallet-rpc \
  --rpc-bind-port 18082 \
  --wallet-file /path/to/your/wallet \
  --password yourwalletpassword \
  --testnet \
  --daemon-address stagenet.community.rino.io:38081
```

### Using Auto-Deposit

```bash
./near-swap swap 0.5 XMR to USDC \
  --from-chain xmr \
  --to-chain near \
  --recipient your.near \
  --refund-to <your-xmr-address> \
  --auto-deposit
```

The CLI will:
1. ✓ Generate a swap quote
2. ✓ Show deposit details
3. ✓ Verify monero-wallet-rpc connectivity
4. ✓ Check wallet unlocked balance
5. ✓ Ask for confirmation
6. ✓ Send the Monero transaction
7. ✓ Display transaction hash

### Important Notes

- **Unlocked Balance**: Only unlocked funds can be spent. New funds need 10 confirmations (~20 minutes).
- **Transaction Priority**: Higher priority (1-4) means higher fees but faster confirmation.
- **RPC Security**: Use authentication (username/password) when exposing RPC to network.
- **Wallet Must Be Open**: monero-wallet-rpc must be running with your wallet loaded.

## Environment Variables

Alternative to config file:

```bash
# Required
export NEAR_SWAP_JWT_TOKEN="your-token"

# Optional
export NEAR_SWAP_BASE_URL="https://1click.chaindefuser.com"
```

Or use a `.env` file:

```bash
NEAR_SWAP_JWT_TOKEN=your-token-here
```

## Getting a JWT Token

1. Visit: https://docs.near-intents.org/near-intents/integration/distribution-channels/1click-api
2. Apply for API access
3. Receive your JWT token
4. Add it to your config or .env file

**Note**: Without a JWT token, you'll pay a 0.1% fee on all swaps.

## Supported Chains for Auto-Deposit

Currently supported:
- ✅ **Bitcoin (BTC)** - via bitcoin-cli
- ✅ **Monero (XMR)** - via monero-wallet-rpc

Coming soon:
- 🔄 Ethereum (ETH) - via web3/RPC
- 🔄 Solana (SOL) - via solana-cli
- 🔄 Near (NEAR) - via near-cli

## Testing Your Setup

### 1. Test API Connection

```bash
./near-swap list-tokens --symbol BTC
```

Expected output: List of BTC tokens on different chains

### 2. Test Quote Generation

```bash
./near-swap swap 0.001 BTC to USDC \
  --from-chain btc \
  --to-chain near \
  --recipient test.near \
  --refund-to <your-btc-address>
```

Expected output: Swap quote with deposit address

### 3. Test Bitcoin Auto-Deposit (if configured)

```bash
# Test with small amount first!
./near-swap swap 0.001 BTC to USDC \
  --from-chain btc \
  --to-chain near \
  --recipient your.near \
  --refund-to <your-btc-address> \
  --auto-deposit \
  --verbose
```

Expected output: Quote → Deposit confirmation → Transaction sent

### 4. Test Monero Auto-Deposit (if configured)

```bash
# Test with small amount first!
./near-swap swap 0.1 XMR to USDC \
  --from-chain xmr \
  --to-chain near \
  --recipient your.near \
  --refund-to <your-xmr-address> \
  --auto-deposit \
  --verbose
```

Expected output: Quote → Deposit confirmation → Transaction sent

## Common Issues

### Bitcoin-cli not found

```bash
# Check if bitcoin-cli is in PATH
which bitcoin-cli

# If not, specify full path in config
auto_deposit:
  bitcoin:
    cli_path: "/usr/local/bin/bitcoin-cli"
```

### Authentication errors

```bash
# If using authentication, add to cli_args
auto_deposit:
  bitcoin:
    cli_args: ["-rpcuser=youruser", "-rpcpassword=yourpass"]
```

### Named wallet not found

```bash
# List available wallets
bitcoin-cli listwallets

# Use correct wallet name in config
auto_deposit:
  bitcoin:
    wallet: "your-wallet-name"
```

### Monero-wallet-rpc not accessible

```bash
# Check if monero-wallet-rpc is running
curl http://127.0.0.1:18082/json_rpc \
  -d '{"jsonrpc":"2.0","id":"0","method":"get_version"}' \
  -H 'Content-Type: application/json'

# If using authentication, add credentials:
curl -u user:pass http://127.0.0.1:18082/json_rpc \
  -d '{"jsonrpc":"2.0","id":"0","method":"get_version"}' \
  -H 'Content-Type: application/json'

# Start monero-wallet-rpc if not running
monero-wallet-rpc \
  --rpc-bind-port 18082 \
  --wallet-file /path/to/wallet \
  --password yourpassword
```

### Monero insufficient balance

```bash
# The issue might be that funds are not yet unlocked
# Monero requires 10 confirmations before funds are spendable
# Wait for confirmations and try again

# You can check your balance with:
curl http://127.0.0.1:18082/json_rpc \
  -d '{"jsonrpc":"2.0","id":"0","method":"get_balance","params":{"account_index":0}}' \
  -H 'Content-Type: application/json'
# Look at "unlocked_balance" field
```

### Monero RPC authentication errors

```bash
# If you set up RPC authentication, update your config:
auto_deposit:
  monero:
    username: "your-rpc-username"
    password: "your-rpc-password"
```

## Support

- **Documentation**: Check the main README.md
- **Issues**: Open an issue on GitHub
- **NEAR Intents Docs**: https://docs.near-intents.org/
