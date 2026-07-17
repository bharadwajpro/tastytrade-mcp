# tastytrade-mcp

A public, fully-configurable [MCP](https://modelcontextprotocol.io/) server exposing the **tastytrade** brokerage API as MCP tools — REST, DxLink market streaming, and account streaming.

## Features

- **Instruments & search** — look up equities, search any tradeable symbol, fetch full nested option chains
- **Market data** — synchronous bid/ask/last quotes, live streaming quotes via DxLink, and market metrics (IV rank/percentile, beta, liquidity)
- **Market sessions** — check whether the equity market is currently open
- **Accounts** — linked accounts and per-account trading status
- **Balances & positions** — account balances, net liquidating value, and open positions
- **Transactions** — filtered transaction history
- **Orders (gated)** — dry-run-first placement, replace, patch, cancel, complex-order (OTOCO/OCO) cancel, plus live-order listing and historical search
- **Margin** — margin requirements by account
- **Watchlists & alerts (gated)** — create/update/delete watchlists and quote alerts
- **Account streamer** — background WebSocket that caches real-time order/balance/position updates; surfaced via a status tool

## Safety model

- **`TT_READ_ONLY=true` ships as the default** — no writes are possible without explicitly opting in.
- **Trading** requires `TT_ENABLE_TRADING=true`, and `place_order` always runs a dry-run first — a live order is only submitted when the call passes `confirm=true`.
- **Watchlist and alert writes** each require their own flag (`TT_ENABLE_WATCHLIST_WRITES`, `TT_ENABLE_QUOTE_ALERTS`).
- The master kill-switch `TT_READ_ONLY=true` overrides every other flag, no matter what.

## OAuth setup — from zero to a refresh token

The server authenticates to tastytrade using an OAuth2 **`client_secret` + `refresh_token`**. It exchanges them for short-lived (15-minute) access tokens at runtime and refreshes automatically. You do this setup once.

1. **Create an OAuth application.** Log into [my.tastytrade.com](https://my.tastytrade.com) → **Manage → My Profile → API → OAuth Applications → + New OAuth client**. Pick the `read` and (optionally) `trade` scopes. On creation you are shown a **Client ID** and **Client Secret** — copy the secret now (it is shown only once) into `TT_CLIENT_SECRET`.
2. **Create a personal grant.** On the same page, click **Manage → Create Grant** for your application. This reveals a **refresh token** (long-lived) — put it in `TT_REFRESH_TOKEN`.
3. **Pick an environment.** Sandbox apps talk to `api.cert.tastyworks.com`; production apps talk to `api.tastyworks.com`. Start with `TT_ENVIRONMENT=sandbox`.

> The refresh token is long-lived. If it is ever lost or compromised, delete the grant on my.tastytrade.com and create a new one. Never commit `.env` — it is gitignored.

## Quick start

```sh
git clone https://github.com/your-org/tastytrade-mcp
cd tastytrade-mcp
cp .env.example .env
# edit .env: set TT_CLIENT_SECRET, TT_REFRESH_TOKEN, TT_ACCOUNT_NUMBER
# keep TT_ENVIRONMENT=sandbox and TT_READ_ONLY=true to start

# run in stdio mode (simplest):
MCP_TRANSPORT=stdio go run ./cmd/tastytrade-mcp
```

## Run modes

### stdio (single-user, local)

The server speaks the MCP protocol on stdin/stdout; diagnostics go to stderr. Ideal for Claude Desktop, which spawns the binary directly.

```sh
MCP_TRANSPORT=stdio ./tastytrade-mcp
```

### HTTP + OAuth (multi-user, remote)

The server exposes `/mcp` behind bearer auth and publishes OAuth discovery documents at `/.well-known/oauth-authorization-server`. MCP clients connect, discover the endpoints, and complete an authorization-code + PKCE flow — authenticating with a fixed `OAUTH_CLIENT_ID` + `OAUTH_CLIENT_SECRET` you generate and share with the client — to obtain a short-lived access token (and a rotating refresh token). `/mcp` accepts only valid, unexpired access tokens.

```sh
MCP_TRANSPORT=http SERVER_URL=https://mcp.example.com PORT=8080 \
  OAUTH_CLIENT_ID=<random-id> OAUTH_CLIENT_SECRET=<random-secret> ./tastytrade-mcp
```

`SERVER_URL` must be a public **`https://`** URL for discovery to work — terminate TLS at your reverse proxy and point it at the server's `PORT`. There is no dynamic client registration: any client_id/secret other than the configured pair is rejected, and if either is left blank the OAuth endpoint rejects every client (fail-closed), leaving `/mcp` unreachable. Tokens are held in memory, so a restart forces clients to re-authenticate.

Generate `OAUTH_CLIENT_ID` (32 characters) and `OAUTH_CLIENT_SECRET` (64 characters, alphanumeric):

```sh
tr -dc 'A-Za-z0-9' </dev/urandom | head -c 32; echo   # OAUTH_CLIENT_ID
tr -dc 'A-Za-z0-9' </dev/urandom | head -c 64; echo   # OAUTH_CLIENT_SECRET
```

### Deploy as a systemd service behind Caddy (Tailscale)

For a single always-on HTTP instance shared by multiple agents: bind the binary to localhost, terminate TLS on Caddy on your Tailscale IP, and run under a user systemd unit so it survives reboots and crashes.

1. **Build and configure `.env`:**

```sh
go build -o tastytrade-mcp ./cmd/tastytrade-mcp
```

```env
MCP_TRANSPORT=http
PORT=9090
LISTEN_HOST=127.0.0.1
SERVER_URL=https://tt.ts.example.com
# plus TT_* credentials and OAUTH_CLIENT_ID / OAUTH_CLIENT_SECRET
```

`LISTEN_HOST=127.0.0.1` keeps the app off the LAN/WAN; only the reverse proxy should reach it. `SERVER_URL` must be the public HTTPS hostname clients use (OAuth discovery embeds it).

2. **User systemd unit** (`~/.config/systemd/user/tastytrade-mcp.service`):

```ini
[Unit]
Description=tastytrade MCP server (HTTP + OAuth)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=%h/Projects/tastytrade-mcp
ExecStart=%h/Projects/tastytrade-mcp/tastytrade-mcp
Restart=always
RestartSec=5
TimeoutStopSec=15
Environment=HOME=%h
Environment=PATH=%h/.local/bin:/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin
StartLimitIntervalSec=60
StartLimitBurst=5

[Install]
WantedBy=default.target
```

Adjust `WorkingDirectory` / `ExecStart` to your checkout path. The process loads `.env` from the working directory via godotenv.

```sh
systemctl --user daemon-reload
systemctl --user enable --now tastytrade-mcp.service
# linger so the user unit starts at boot without an interactive login:
loginctl enable-linger "$USER"
```

Useful commands: `systemctl --user status tastytrade-mcp`, `systemctl --user restart tastytrade-mcp`, `journalctl --user -u tastytrade-mcp -f`. Rebuild the binary, then restart the unit after code changes.

3. **Caddy on the Tailscale IP** — listen only on the tailnet address (not LAN), proxy to localhost. Example site block (wildcard DNS `*.ts.example.com` → your Tailscale IPv4):

```caddy
(tailnet_tls) {
	bind 100.x.y.z          # your Tailscale IPv4 (and IPv6 if desired)
	tls {
		dns cloudflare {$CLOUDFLARE_API_TOKEN}
	}
	encode gzip
	header {
		Strict-Transport-Security "max-age=31536000; includeSubDomains"
		X-Content-Type-Options "nosniff"
		X-Frame-Options "DENY"
	}
}

tt.ts.example.com {
	import tailnet_tls
	# Streamable HTTP / SSE — disable response buffering
	reverse_proxy 127.0.0.1:9090 {
		flush_interval -1
	}
}
```

Reload Caddy after editing. Clients on the same Tailscale network then use `https://tt.ts.example.com/mcp`.

**stdio vs this HTTP deploy:** each stdio client typically spawns its own process; this HTTP setup is one long-lived server shared by many agents.

## Configuration

All configuration is via environment variables or a `.env` file. See [`.env.example`](.env.example) for the complete, annotated reference.

| Variable | Default | Description |
|---|---|---|
| `TT_CLIENT_SECRET` | *(required)* | OAuth client secret |
| `TT_REFRESH_TOKEN` | *(required)* | OAuth refresh token |
| `TT_ENVIRONMENT` | `production` | `production` or `sandbox` |
| `TT_ACCOUNT_NUMBER` | — | Default account (or a comma-list via `TT_ACCOUNT_NUMBERS`) |
| `TT_READ_ONLY` | `false`* | Master write kill-switch (`.env.example` ships it `true`) |
| `TT_ENABLE_TRADING` | `false` | Allow order placement (also needs `TT_READ_ONLY=false`) |
| `TT_ENABLE_WATCHLIST_WRITES` | `false` | Allow watchlist create/update/delete |
| `TT_ENABLE_QUOTE_ALERTS` | `false` | Allow quote-alert list/create/delete |
| `MCP_TRANSPORT` | `http` | `http` or `stdio` |
| `PORT` | `8080` | HTTP listen port |
| `LISTEN_HOST` | *(all interfaces)* | Optional bind host; use `127.0.0.1` behind a reverse proxy |
| `SERVER_URL` | — | Required for HTTP mode; the public HTTPS base URL of this server |
| `OAUTH_CLIENT_ID` | — | Fixed client id the MCP client presents to the OAuth flow (HTTP mode) |
| `OAUTH_CLIENT_SECRET` | — | Fixed client secret for the OAuth flow; with the id, gates token issuance |
| `TT_TOOL_TIMEOUT` | `30s` | Per-tool timeout |
| `TT_STALE_QUOTE_THRESHOLD` | `5s` | Age after which a streamed quote is flagged stale |

\* The code default is `false`, but the shipped `.env.example` sets `TT_READ_ONLY=true` so a fresh checkout is read-only until you opt in.

## Connecting clients

### Claude Desktop (stdio)

Add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "tastytrade": {
      "command": "/path/to/tastytrade-mcp",
      "env": {
        "MCP_TRANSPORT": "stdio",
        "TT_CLIENT_SECRET": "...",
        "TT_REFRESH_TOKEN": "..."
      }
    }
  }
}
```

### HTTP + OAuth client

Point your MCP client at `{SERVER_URL}/mcp` (for example `https://tt.ts.example.com/mcp`). It discovers the OAuth endpoints via `{SERVER_URL}/.well-known/oauth-authorization-server` and performs PKCE to obtain a bearer token automatically. Configure the client with the same `OAUTH_CLIENT_ID` / `OAUTH_CLIENT_SECRET` set on the server.

## Tool catalog

All tools are read-only by default. Write tools require the flag in the **Gate** column **plus** `TT_READ_ONLY=false`. An `account` argument is optional when a single account is configured and required when several are.

### Read-only (always available)

| Tool | Description | Key args |
|---|---|---|
| `get_equity` | Look up an equity instrument | `symbol` |
| `search_symbols` | Search tradeable symbols by fragment | `query` |
| `get_option_chain` | Nested option chain: equity OCC, or futures-option series (`/CL` = all, `/CLZ6` = that future) | `symbol` |
| `select_contract_by_delta` | Pick the provided contract nearest a target absolute delta | `contracts`, `target_abs_delta` |
| `select_contract_by_width` | Pick the provided contract nearest `anchor_strike + signed_width` | `contracts`, `anchor_strike`, `signed_width` |
| `get_market_data` | Full synchronous market quotes by instrument type | `equities`, `equity_options`, `futures`, `future_options` |
| `get_streaming_quote` | Live DxLink Quote/Trade/Summary/Profile/Greeks snapshot | `symbols`, `account` (optional; enables position-based streamer resolution) |
| `get_market_metrics` | IV rank/percentile, beta, liquidity | `symbols` |
| `is_market_open` | Whether the equity session is currently open | — |
| `list_accounts` | Accounts available to the customer | — |
| `get_trading_status` | Permissions, day-trade count, margin-call flag | `account` |
| `get_balances` | Cash, net liq, buying power | `account` |
| `list_positions` | Open positions with streamer-symbol and cost/P/L fields | `account` |
| `list_transactions` | Filtered transaction history | `account`, `symbol`, `type`, `start_date`, `end_date` |
| `get_margin_requirements` | Margin requirements report | `account` |
| `get_order` | A single order by id | `account`, `id` |
| `list_live_orders` | Currently-live orders | `account` |
| `search_orders` | Historical orders by status/symbol/date | `account`, `status`, `underlying_symbol`, dates |
| `dry_run_order` | Validate an order + preview buying-power effect | `account`, `legs`, `order_type`, `price` |
| `list_watchlists` | All user watchlists | — |
| `get_account_stream_status` | Account-streamer health + cached counts | — |

### Trading — `TT_ENABLE_TRADING`

| Tool | Description | Key args |
|---|---|---|
| `place_order` | Place an order (dry-run unless `confirm=true`) | `account`, `legs`, `confirm` |
| `replace_order` | Replace a live order (PUT) | `account`, `id`, `legs` |
| `patch_order` | Partially update a live order (PATCH) | `account`, `id`, `patch` |
| `cancel_order` | Cancel a live order | `account`, `id` |
| `cancel_complex_order` | Cancel an OTOCO/OCO order | `account`, `id` |

### Watchlist writes — `TT_ENABLE_WATCHLIST_WRITES`

| Tool | Description | Key args |
|---|---|---|
| `create_watchlist` | Create a watchlist | `name`, `group_name`, `entries` |
| `update_watchlist` | Replace a watchlist | `name`, `entries` |
| `delete_watchlist` | Delete a watchlist | `name` |

### Quote alerts — `TT_ENABLE_QUOTE_ALERTS`

| Tool | Description | Key args |
|---|---|---|
| `list_quote_alerts` | List configured quote alerts | — |
| `create_quote_alert` | Create a price alert | `symbol`, `field`, `operator`, `threshold` |
| `delete_quote_alert` | Delete a quote alert | `id` |

**Rate-limit note:** order tools are subject to tastytrade's order-rate limits. The server adds no extra throttling — rapid repeated calls may return `429` from the API.

## Build

```sh
go build -o tastytrade-mcp ./cmd/tastytrade-mcp
go vet ./...
```
