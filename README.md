# dex-backend

Go backend for the dex price chart viewer. Streams live prices via WebSocket, proxies historical data from GeckoTerminal, and detects on-chain swaps through Helius.

## Project Structure

```
dex-backend/
├── main.go                     # Entry point, wires all components together
├── api/
│   ├── ohlcv.go                # GET /ohlcv — historical candlestick data
│   ├── trades.go               # GET /trades — recent swap history
│   ├── poolinfo.go             # GET /pool-info — token metadata, supply, FDV
│   └── errors.go               # Error handling with stale cache fallback
├── conn/
│   ├── poolManager.go          # Pool subscriptions, RPC polling goroutines, fan-out
│   ├── poolSubscription.go     # Per-pool state: connected clients, cancel function
│   ├── client.go               # WebSocket client: read/write loops, cleanup
│   ├── handler.go              # HTTP → WebSocket upgrade (gorilla/websocket)
│   ├── message.go              # Incoming/outgoing message types (price + trade)
│   └── priceTick.go            # PriceTick struct
├── dex/
│   ├── pool.go                 # GetPoolInfo, GetTokenPrice — on-chain pool data
│   ├── raydium.go              # Raydium V4 AMM pool layout parser
│   └── pumpfun.go              # PumpFun bonding curve + AMM pool layout parser
├── geckoterminal/
│   └── geckoterminal.go        # GeckoTerminal API client with rate limiting
├── helius/
│   └── txsub.go                # Helius WebSocket subscriber (logsSubscribe)
├── cache/
│   └── cache.go                # In-memory TTL cache with stale fallback
├── price/
│   └── solprice.go             # SOL/USD price fetcher (Binance, 15s refresh)
├── constants/
│   ├── constants.go            # DEX program IDs (Raydium, PumpFun, Orca)
│   └── pools.go                # Known token mints (USDC, USDT, SOL)
├── types/
│   └── types.go                # PoolInfo struct, DEX enum
├── utils/
│   └── utils.go                # Byte array helpers
└── deploy/
    ├── setup.sh                # VPS first-time setup (Go, Node, Nginx)
    ├── deploy.sh               # Build + deploy backend and frontend
    ├── nginx.conf              # Nginx config with SSL
    ├── nginx-initial.conf      # Nginx config without SSL (for first certbot run)
    ├── dex-backend.service     # systemd service file
    ├── duckdns-update.sh       # DuckDNS dynamic DNS updater
    └── env.example             # Environment variable template
```

## How It Works

### Live Price Streaming

When a user opens a chart, the frontend sends a WebSocket `subscribe` message. The backend:

1. Fetches the pool's on-chain account data and parses the DEX layout
2. Starts a goroutine that polls vault balances every 1 second
3. Computes price as `quoteAmount / baseAmount`
4. Converts to USD using the cached SOL/USD rate from Binance (if quote token is SOL)
5. Broadcasts to all connected clients watching that pool

One goroutine per pool regardless of client count. When the last client leaves, the goroutine stops.

### Historical Data

REST endpoints proxy to GeckoTerminal with caching and stale fallback:

- `/ohlcv` — 15s TTL, falls back to stale data on rate limit
- `/trades` — 10s TTL, same fallback behavior
- `/pool-info` — 60s TTL

GeckoTerminal requests are rate-limited to 1 per 2 seconds (burst 3).

### Transaction Detection

Helius WebSocket uses `logsSubscribe` to monitor pool addresses. Swap transactions are detected and broadcast as `"trade"` messages to connected clients.

### Pool Parsing

Each DEX has a different on-chain layout:

- **Raydium V4** — mints at 400/432, vaults at 336/368, decimals at 464/465
- **PumpFun AMM** — mints at 43/75, vaults at 139/171 (after 8-byte discriminator)
- **PumpFun Bonding Curve** — derives token vault from associated token address

DEX is identified by the account owner program ID.

## Limitations

- **Supported DEXes** — only Raydium V4 and PumpFun (AMM + bonding curve). Orca, Meteora, and other DEXes are not implemented
- **Helius free tier** — 30 req/s limit. At 1s polling with 2 RPC calls per tick, ~15 concurrent pools max before hitting limits
- **GeckoTerminal free tier** — 30 req/min. Rate limiter prevents 429s but new pools may load slowly when many unique pools are requested
- **SOL-quoted pools only** — price conversion assumes SOL as quote token. USDC/USDT-quoted pools work but skip the USD conversion step, which may cause mismatches with GeckoTerminal's USD-denominated historical data
- **No persistent storage** — all cache is in-memory. Server restart loses cached data
- **Single server** — no horizontal scaling. WebSocket fan-out is in-process only
- **Transaction parsing** — `logsSubscribe` gives transaction signature and basic buy/sell detection from logs, but not exact swap amounts. Detailed trade data still comes from GeckoTerminal polling
- **Binance dependency** — SOL/USD price comes from Binance API. If Binance is down or blocked, live prices for SOL-quoted pools won't be sent

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

## Deployment

```bash
# First time
sudo bash deploy/setup.sh

# Deploy
sudo bash deploy/deploy.sh
```

## API Reference

| Endpoint | Method | Params | Description |
|---|---|---|---|
| `/ws` | WebSocket | — | Send `{"action":"subscribe","pool":"..."}` for live price + trade messages |
| `/ohlcv` | GET | `address`, `aggregate`, `timeframe` | OHLCV candles from GeckoTerminal |
| `/trades` | GET | `address` | Recent trades from GeckoTerminal |
| `/pool-info` | GET | `address` | Token name, image, price, FDV, market cap, supply |
