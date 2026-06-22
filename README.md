# dex-backend

Go backend for the dex price chart viewer. Handles live price streaming via WebSocket, proxies historical data from GeckoTerminal, and detects on-chain swaps through Helius.

## Project Structure

```
dex-backend/
├── main.go                 # Entry point, wires everything together
├── api/
│   ├── ohlcv.go            # GET /ohlcv — historical candlestick data
│   ├── trades.go           # GET /trades — recent swap history
│   ├── poolinfo.go         # GET /pool-info — token metadata, supply, FDV
│   └── errors.go           # Shared error handling for GeckoTerminal responses
├── conn/
│   ├── poolManager.go      # Manages pool subscriptions and RPC polling goroutines
│   ├── poolSubscription.go # Per-pool state: connected clients, cancel function
│   ├── client.go           # Single WebSocket client: read loop, write loop, cleanup
│   ├── handler.go          # HTTP → WebSocket upgrade handler
│   ├── message.go          # Incoming/outgoing WebSocket message types
│   └── priceTick.go        # PriceTick struct
├── dex/
│   ├── pool.go             # GetPoolInfo, GetTokenPrice — on-chain pool data
│   ├── raydium.go          # Raydium V4 AMM pool layout parser
│   └── pumpfun.go          # PumpFun bonding curve + AMM pool layout parser
├── geckoterminal/
│   └── geckoterminal.go    # GeckoTerminal API client (OHLCV, trades, pool details, token info)
├── helius/
│   └── txsub.go            # Helius WebSocket subscriber for live transaction detection
├── cache/
│   └── cache.go            # In-memory key-value cache with TTL and auto-cleanup
├── price/
│   └── solprice.go         # SOL/USD price fetcher from CoinGecko (30s refresh)
├── constants/
│   ├── constants.go        # DEX program IDs (Raydium, PumpFun, Orca)
│   └── pools.go            # Known token mints (USDC, USDT, SOL)
├── types/
│   └── types.go            # PoolInfo struct, DEX enum
└── utils/
    └── utils.go            # PubkeyAt helper for reading public keys from byte arrays
```

## How It Works

### Live Price Streaming

When a user opens a chart, the frontend sends a WebSocket `subscribe` message with the pool address. The backend:

1. Fetches the pool's on-chain account data and parses it to identify the DEX and extract vault addresses
2. Starts a goroutine that polls `GetTokenAccountBalance` on both vaults every 3 seconds
3. Computes price as `quoteAmount / baseAmount`
4. If the quote token is SOL, multiplies by the cached SOL/USD rate
5. Broadcasts the price to all connected clients watching that pool

One goroutine per pool regardless of how many clients are watching. When the last client unsubscribes, the goroutine is cancelled.

### Historical Data

REST endpoints proxy to GeckoTerminal with an in-memory cache layer:

- `/ohlcv` — cached 15s per pool+timeframe combination
- `/trades` — cached 10s per pool
- `/pool-info` — cached 60s per pool

GeckoTerminal requests are rate-limited to 1 request per 2 seconds (burst of 3) to stay within the free tier.

### Transaction Detection

The Helius WebSocket subscriber connects to `wss://mainnet.helius-rpc.com` and uses `logsSubscribe` to monitor subscribed pool addresses. When a swap is detected, the transaction signature and type (buy/sell) are broadcast to connected clients.

### Pool Parsing

Each DEX has a different on-chain account layout:

- **Raydium V4** — base/quote mints at offsets 400/432, vaults at 336/368, decimals at 464/465
- **PumpFun AMM** — mints at offsets 43/75, vaults at 139/171 (after 8-byte discriminator)
- **PumpFun Bonding Curve** — detected by discriminator, derives token vault from associated token address

The DEX is identified by the account's owner program ID.

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `HELIUS_API_KEY` | Yes | Helius RPC API key |
| `PORT` | No | Server port (default: 8080) |

## Running

```bash
export HELIUS_API_KEY=your_key_here
go run .
```

## API Reference

| Endpoint | Method | Params | Description |
|---|---|---|---|
| `/ws` | WebSocket | — | Send `{"action":"subscribe","pool":"..."}` to start receiving price ticks |
| `/ohlcv` | GET | `address`, `aggregate`, `timeframe` | OHLCV candles from GeckoTerminal |
| `/trades` | GET | `address` | Recent trades from GeckoTerminal |
| `/pool-info` | GET | `address` | Token name, image, price, FDV, market cap, supply |
