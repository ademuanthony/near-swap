# Near-Swap CLI

A command-line interface for cross-chain token swaps using the NEAR Intents 1Click API.

## Features

- 🔄 **Cross-chain swaps**: Swap tokens across different blockchains
- 📝 **Natural language commands**: Simple syntax like "swap 1 SOL to USDC"
- 🎨 **Rich terminal output**: Colorized output with progress indicators
- 📊 **Token discovery**: List all supported tokens across chains
- 🔍 **Status tracking**: Monitor swap execution in real-time
- 🌐 **Multi-chain support**: Works with multiple blockchains via NEAR Intents
- 📈 **Trading plans**: Automated price-triggered swaps with execution history
- 🤖 **Auto-deposit**: Automatically send deposits for Bitcoin, Monero, Zcash, EVM, and Solana

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

# Optional: Configure auto-deposit
auto_deposit:
  enabled: true

  # Bitcoin
  bitcoin:
    enabled: true
    cli_path: "bitcoin-cli"

  # Solana
  solana:
    enabled: true
    rpc_url: "https://api.mainnet-beta.solana.com"
    private_key: "YOUR_BASE58_PRIVATE_KEY"
    commitment: "confirmed"

  # EVM Networks (Ethereum, BSC, Polygon, etc.)
  evm:
    enabled: true
    networks:
      ethereum:
        rpc_url: "https://eth-mainnet.g.alchemy.com/v2/YOUR-API-KEY"
        chain_id: 1
        private_key: "0xYOUR_PRIVATE_KEY"
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

### Trading Plans (Automated Strategies)

Create automated trading plans that execute swaps when specific price conditions are met. Perfect for dollar-cost averaging, limit orders, and automated trading strategies.

**Key Features:**
- ⏰ **Price-triggered execution**: Automatically execute trades when price crosses a threshold
- 📊 **Partial fulfillment**: Split large orders into multiple smaller trades
- 💾 **Persistent state**: Survives restarts and tracks full execution history
- 🔄 **Auto-deposit required**: All plans use auto-deposit for seamless execution
- 📈 **Multiple plans**: Run multiple trading strategies simultaneously
- 📅 **Daily limits**: Control how much to trade per day to spread executions over time

#### Create a Trading Plan

**IMPORTANT**: All trading plans require auto-deposit to be configured for the source blockchain. See [Auto-Deposit Feature](#auto-deposit-feature) section for setup instructions.

```bash
# Sell 10 BTC when price reaches $150k (1 BTC per trade, max 2 BTC per day)
near-swap plan create sell-btc-high \
  --from BTC --to USDC \
  --from-chain btc --to-chain near \
  --total 10 --per-trade 1 --per-day 2 \
  --when-price "above 150000" \
  --recipient your.near \
  --refund-to <your-btc-address>

# Buy ETH when price drops below $3000 (max $1000 per day)
near-swap plan create buy-eth-dip \
  --from USDC --to ETH \
  --from-chain near --to-chain eth \
  --total 5000 --per-trade 500 --per-day 1000 \
  --when-price "below 3000" \
  --recipient 0x123... \
  --description "Buy the dip strategy"
```

#### List All Plans

```bash
# List all plans
near-swap plan list

# List only active plans
near-swap plan list --status active

# JSON output
near-swap plan list --json
```

#### View Plan Details

```bash
# View detailed information about a plan
near-swap plan view sell-btc-high

# View as JSON
near-swap plan view sell-btc-high --json
```

#### Start/Stop Plans

```bash
# Activate a plan (changes status to "active")
near-swap plan start sell-btc-high

# Deactivate a plan (changes status to "paused")
near-swap plan stop sell-btc-high
```

#### Run the Daemon

After activating your plans, run the daemon to start monitoring and executing:

```bash
# Start daemon in foreground (recommended for testing)
near-swap plan daemon

# Run daemon in background (Linux/Mac)
nohup near-swap plan daemon > ~/near-swap-daemon.log 2>&1 &

# Check daemon logs
tail -f ~/near-swap-daemon.log
```

**The daemon will:**
- Automatically load all active plans and their execution history
- Resume from where it stopped (survives restarts)
- Monitor prices every 30 seconds
- **Check for plan changes every 60 seconds** (dynamically detects new/started/stopped plans)
- Execute trades when conditions are met
- Respect daily limits for each plan
- Save state after each execution
- Handle graceful shutdown on Ctrl+C

**Dynamic Plan Management:**
- The daemon automatically detects new plans created and started
- No need to restart daemon when adding new plans
- Start/stop plans in another terminal - daemon adjusts within 60 seconds
- Perfect for managing multiple strategies without downtime

#### View Execution History

```bash
# See all executions for a plan
near-swap plan history sell-btc-high

# JSON output for analysis
near-swap plan history sell-btc-high --json
```

#### Delete a Plan

```bash
# Delete a plan (must be stopped first)
near-swap plan delete sell-btc-high
```

#### Price Conditions

Trading plans support three price condition types:

- **`above <price>`**: Execute when price goes above the specified value
- **`below <price>`**: Execute when price goes below the specified value
- **`at <price>`**: Execute when price equals the value (±0.5% tolerance)

Examples:
```bash
--when-price "above 150000"  # BTC > $150k
--when-price "below 3000"    # ETH < $3k
--when-price "at 100"        # SOL ≈ $100
```

#### How Trading Plans Work

1. **Create**: Define your trading strategy with price conditions and daily limits
2. **Activate**: Mark the plan as "active" with `near-swap plan start <name>`
3. **Run Daemon**: Start the monitoring daemon with `near-swap plan daemon`
4. **Load & Resume**: Daemon loads all active plans and their execution history
5. **Monitor**: Bot checks current prices every 30 seconds using 1Click API quotes
6. **Execute**: When conditions are met and daily limit not reached, creates a swap
7. **Auto-Deposit**: Automatically sends the deposit using configured auto-deposit
8. **Daily Tracking**: Tracks executed amount per day and resets at midnight
9. **Repeat**: Continues until the total amount is fully executed
10. **Persist**: Full history saved - survives restarts

**Daily Limit System:**
- Each plan has a `--per-day` limit to control trade frequency
- The bot tracks how much has been executed each day
- Once the daily limit is reached, no more trades until the next day
- Daily counter resets at midnight (00:00) local time
- Useful for spreading large orders over multiple days

**State Persistence:**
- All plan data stored in `~/.near-swap-plans.json`
- Execution history tracked for each trade
- Daily counters and progress saved automatically
- Daemon resumes from exact state after restart
- Never loses track of your trades

#### Plan Storage

Plans are stored in `~/.near-swap-plans.json` by default. This file contains:
- All plan configurations
- Execution history
- Current progress and remaining amounts

The storage location can be customized in your `.near-swap.yaml`:
```yaml
plan_storage_path: "/custom/path/to/plans.json"
```

#### Example Strategies

**Dollar-Cost Averaging (DCA):**
```bash
# Buy BTC gradually: $100 per trade, max $500 per day
near-swap plan create dca-btc \
  --from USDC --to BTC \
  --from-chain near --to-chain btc \
  --total 10000 --per-trade 100 --per-day 500 \
  --when-price "below 50000" \
  --recipient <btc-address> \
  --refund-to your.near
```

**Take Profit (Spread over 5 days):**
```bash
# Sell ETH gradually: 0.5 per trade, max 1 ETH per day
near-swap plan create take-profit-eth \
  --from ETH --to USDC \
  --from-chain eth --to-chain near \
  --total 5 --per-trade 0.5 --per-day 1 \
  --when-price "above 4000" \
  --recipient your.near \
  --refund-to 0xYourEthAddress
```

**Buy the Dip (Conservative approach):**
```bash
# Buy SOL dip: $100 per trade, max $200 per day to avoid FOMO
near-swap plan create sol-dip \
  --from USDC --to SOL \
  --from-chain near --to-chain sol \
  --total 2000 --per-trade 100 --per-day 200 \
  --when-price "below 80" \
  --recipient <sol-address> \
  --refund-to your.near
```

**Aggressive Selling (Market downturn):**
```bash
# Sell entire position fast: 2 BTC per trade, max 10 BTC per day
near-swap plan create sell-btc-fast \
  --from BTC --to USDC \
  --from-chain btc --to-chain near \
  --total 50 --per-trade 2 --per-day 10 \
  --when-price "below 40000" \
  --recipient your.near \
  --refund-to <btc-address>
```

#### Complete Workflow Example

Here's a complete example of setting up and running trading plans:

```bash
# 1. Configure auto-deposit for Bitcoin in ~/.near-swap.yaml
# (See Auto-Deposit Feature section below)

# 2. Create your first trading plan
near-swap plan create btc-dca \
  --from BTC --to USDC \
  --from-chain btc --to-chain near \
  --total 10 --per-trade 0.5 --per-day 1 \
  --when-price "below 60000" \
  --recipient your.near \
  --refund-to <btc-address>

# 3. Create a second plan
near-swap plan create eth-profit \
  --from ETH --to USDC \
  --from-chain eth --to-chain near \
  --total 5 --per-trade 0.5 --per-day 1 \
  --when-price "above 4000" \
  --recipient your.near \
  --refund-to 0xYourEthAddress

# 4. View all plans
near-swap plan list

# 5. Activate the plans you want to run
near-swap plan start btc-dca
near-swap plan start eth-profit

# 6. Start the daemon to monitor and execute
near-swap plan daemon

# The daemon will:
# - Load both active plans
# - Show their current state and history
# - Monitor prices every 30 seconds
# - Check for plan changes every 60 seconds
# - Execute trades automatically when conditions are met
# - Save state after each execution

# 7. In another terminal, manage plans dynamically
near-swap plan view btc-dca
near-swap plan history eth-profit

# Create and start a new plan while daemon runs
near-swap plan create sol-trade --from SOL --to USDC ...
near-swap plan start sol-trade
# Daemon will detect and start monitoring within 60 seconds!

# Pause a plan temporarily
near-swap plan stop btc-dca
# Daemon will stop monitoring it within 60 seconds

# 8. When you're done, stop the daemon with Ctrl+C
# All state is saved automatically
# Restart anytime with: near-swap plan daemon
```

## Auto-Deposit Feature

The CLI supports automatically sending your deposit for supported blockchains:

### Supported Blockchains
- **Bitcoin** (BTC) - via `bitcoin-cli`
- **Monero** (XMR) - via `monero-wallet-rpc`
- **Zcash** (ZEC) - via `zcash-cli`
- **EVM Networks** (ETH, BNB, MATIC, etc.) - via JSON-RPC
  - Ethereum, BSC, Polygon, Avalanche, Arbitrum, Optimism, Base, Fantom
  - Supports both native tokens (ETH, BNB, MATIC) and ERC20 tokens (USDC, USDT, etc.)
- **Solana** (SOL) - via JSON-RPC
  - Supports native SOL and SPL tokens (USDC, USDT, etc.)
  - Automatic associated token account creation

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

### Setup Auto-Deposit for EVM Networks

The CLI supports auto-deposit for all EVM-compatible networks. You can configure multiple networks and send both native tokens (ETH, BNB, MATIC) and ERC20 tokens (USDC, USDT, DAI, etc.).

1. Configure one or more EVM networks in your `.near-swap.yaml`:

```yaml
auto_deposit:
  enabled: true
  evm:
    enabled: true
    networks:
      ethereum:
        rpc_url: "https://eth-mainnet.g.alchemy.com/v2/YOUR-API-KEY"
        chain_id: 1
        private_key: "0xYOUR_PRIVATE_KEY_HERE"
        # gas_price: 20000000000  # Optional: wei per gas unit
        # gas_limit: 100000       # Optional: max gas for transaction

      bsc:
        rpc_url: "https://bsc-dataseed.binance.org"
        chain_id: 56
        private_key: "0xYOUR_PRIVATE_KEY_HERE"

      polygon:
        rpc_url: "https://polygon-rpc.com"
        chain_id: 137
        private_key: "0xYOUR_PRIVATE_KEY_HERE"

      # You can add more networks: arbitrum, optimism, avalanche, base, fantom, etc.
```

**Important**:
- Never commit your private key to version control
- Use environment variables or secure key management for production
- The private key should be in hex format with or without the '0x' prefix

2. Use the `--auto-deposit` flag for native token swaps:

```bash
# Swap native ETH
near-swap swap 0.1 ETH to USDC \
  --from-chain eth \
  --to-chain near \
  --recipient your.near \
  --refund-to 0xYourEthAddress \
  --auto-deposit
```

3. For ERC20 token swaps, specify the token contract address in the deposit address using the format `recipient|tokenContract`:

**Note**: The CLI will automatically detect if a token is ERC20 based on the deposit address format. When swapping ERC20 tokens through the 1Click API, the deposit address may include the token contract information.

Example for USDC on Ethereum:

```bash
near-swap swap 100 USDC to SOL \
  --from-chain eth \
  --to-chain sol \
  --recipient YourSolanaAddress \
  --refund-to 0xYourEthAddress \
  --auto-deposit
```

The CLI will:
- Connect to the configured RPC endpoint
- Detect if the token is native (ETH/BNB/MATIC) or ERC20 based on the deposit address
- For ERC20: Query the token balance using the `balanceOf` function
- For native: Check your wallet's ETH/BNB/MATIC balance
- Estimate gas costs
- Confirm the deposit with you
- Sign and send the transaction
- Display the transaction hash

**Supported EVM Networks**:
- **Ethereum** (eth, ethereum) - Chain ID: 1
- **BSC** (bsc, bnb) - Chain ID: 56
- **Polygon** (polygon, matic) - Chain ID: 137
- **Avalanche** (avalanche, avax) - Chain ID: 43114
- **Arbitrum** (arbitrum) - Chain ID: 42161
- **Optimism** (optimism) - Chain ID: 10
- **Base** (base) - Chain ID: 8453
- **Fantom** (fantom) - Chain ID: 250

**ERC20 Token Support**:
The EVM depositor automatically handles ERC20 tokens. It will:
- Call the `balanceOf` function to check your token balance
- Call the `transfer` function to send tokens
- Automatically estimate gas for ERC20 transactions
- Handle tokens with 18 decimals (standard for most ERC20 tokens)

### Setup Auto-Deposit for Solana

The CLI supports auto-deposit for Solana, including both native SOL and SPL tokens (the Solana equivalent of ERC20 tokens).

1. Configure Solana in your `.near-swap.yaml`:

```yaml
auto_deposit:
  enabled: true
  solana:
    enabled: true
    rpc_url: "https://api.mainnet-beta.solana.com"
    # Or use a dedicated RPC provider for better performance:
    # rpc_url: "https://solana-mainnet.g.alchemy.com/v2/YOUR-API-KEY"
    private_key: "YOUR_BASE58_ENCODED_PRIVATE_KEY"
    commitment: "confirmed"     # Options: finalized, confirmed, processed
    # skip_preflight: false     # Optional: skip transaction simulation
```

**Important**:
- The private key should be Base58 encoded (the standard Solana format)
- You can export it from Phantom, Solflare, or use `solana-keygen` CLI
- Never commit your private key to version control
- Use a dedicated wallet for auto-deposit with limited funds

2. Use the `--auto-deposit` flag for native SOL swaps:

```bash
# Swap native SOL
near-swap swap 1 SOL to USDC \
  --from-chain sol \
  --to-chain near \
  --recipient your.near \
  --refund-to YourSolanaAddress \
  --auto-deposit
```

3. For SPL token swaps, the CLI will automatically detect the token type:

**Note**: The CLI will automatically handle SPL tokens when swapping tokens like USDC, USDT, or any other SPL token on Solana.

Example for USDC on Solana:

```bash
near-swap swap 100 USDC to ETH \
  --from-chain sol \
  --to-chain eth \
  --recipient 0xYourEthAddress \
  --refund-to YourSolanaAddress \
  --auto-deposit
```

The CLI will:
- Connect to the configured Solana RPC endpoint
- Detect if the token is native SOL or an SPL token
- For SPL tokens: Automatically find or create associated token accounts
- Check your SOL balance (for native transfers) or token balance (for SPL tokens)
- Estimate transaction fees (typically ~5000 lamports per signature)
- Confirm the deposit with you
- Sign and send the transaction
- Display the transaction signature

**Commitment Levels**:
- **finalized**: Wait for full confirmation (~32 seconds) - Most secure
- **confirmed**: Wait for majority confirmation (~400ms) - Recommended
- **processed**: Immediate (~200ms) - Fastest but less secure

**SPL Token Support**:
The Solana depositor automatically handles SPL tokens:
- Queries token decimals from the mint account
- Checks token balance using the associated token account
- Creates associated token accounts if they don't exist (at recipient)
- Uses the Token Program for transfers
- Supports all standard SPL tokens

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
│   ├── status.go               # Status check command
│   └── plan.go                 # Trading plan commands
├── pkg/
│   ├── client/
│   │   └── oneclick.go         # 1Click API client wrapper
│   ├── parser/
│   │   └── command.go          # Command parser
│   ├── deposit/
│   │   ├── deposit.go          # Deposit manager
│   │   ├── bitcoin.go          # Bitcoin auto-deposit
│   │   ├── monero.go           # Monero auto-deposit
│   │   ├── zcash.go            # Zcash auto-deposit
│   │   ├── evm.go              # EVM auto-deposit (ETH, BSC, Polygon, etc.)
│   │   └── solana.go           # Solana auto-deposit (SOL, SPL tokens)
│   ├── plan/
│   │   ├── types.go            # Trading plan data structures
│   │   ├── storage.go          # JSON-based persistence
│   │   ├── manager.go          # Plan CRUD operations
│   │   ├── pricer.go           # Price monitoring
│   │   └── executor.go         # Automated execution engine
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
- [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum) - Ethereum client library for EVM support
- [github.com/gagliardetto/solana-go](https://github.com/gagliardetto/solana-go) - Solana client library for Solana support

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
- For EVM: `auto_deposit.evm.enabled: true` and the network is configured

**EVM errors:**

**"network X not configured"**:
- Add the network configuration to your `.near-swap.yaml`
- Ensure the network name matches (ethereum, bsc, polygon, etc.)
- Check that all required fields are set: `rpc_url`, `chain_id`, `private_key`

**"failed to connect to RPC endpoint"**:
- Verify the RPC URL is correct and accessible
- Check your internet connection
- For Alchemy/Infura, verify your API key is valid
- Test the RPC endpoint: `curl -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' YOUR_RPC_URL`

**"invalid private key"**:
- Ensure the private key is in hex format
- It should be 64 characters (32 bytes) without the '0x' prefix, or 66 characters with it
- Never use a private key from a wallet with significant funds for testing

**"insufficient balance"** or **"insufficient token balance"**:
- For native tokens: Check your ETH/BNB/MATIC balance
- For ERC20 tokens: Check your token balance
- Ensure you have enough for both the transfer amount AND gas fees
- For ERC20: You need native tokens (ETH/BNB/etc.) for gas, even when sending tokens

**"failed to estimate gas"**:
- This usually indicates an issue with the transaction
- For ERC20: Ensure the token contract address is correct
- Verify you have enough token balance
- Check if the token has any transfer restrictions

**Solana errors:**

**"failed to connect to RPC endpoint"**:
- Verify the RPC URL is correct and accessible
- Check your internet connection
- For Alchemy/QuickNode, verify your API key is valid
- Test the RPC endpoint: `curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc":"2.0","id":1,"method":"getVersion"}' YOUR_RPC_URL`
- Consider using a dedicated RPC provider (Alchemy, QuickNode, Helius) for better performance

**"invalid private key"**:
- Ensure the private key is Base58 encoded (Solana standard format)
- Export from wallet: Phantom (Settings > Export Private Key) or Solflare
- From CLI: `solana-keygen pubkey /path/to/keypair.json` to verify
- Private key should be a long Base58 string (not JSON array format)

**"insufficient balance"** or **"insufficient token balance"**:
- For native SOL: Check your SOL balance
- Ensure you have enough for both the transfer AND transaction fees (~0.000005 SOL per signature)
- For SPL tokens: Check your token balance using `spl-token accounts`
- You need SOL for fees even when sending SPL tokens

**"failed to get token decimals"** or **"invalid mint account"**:
- Verify the token mint address is correct
- Ensure you're using the correct network (mainnet vs devnet)
- Check if the token exists: `spl-token display <mint-address>`

**"transaction simulation failed"**:
- This usually indicates the transaction would fail on-chain
- Check if you have enough balance for both amount and fees
- For SPL tokens: Ensure the recipient's associated token account can be created
- Try setting `skip_preflight: true` in config (not recommended for production)

**"blockhash not found"**:
- Network congestion or RPC node issues
- Retry the transaction
- Consider using a different RPC endpoint

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
