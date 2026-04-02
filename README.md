# funnyoption

A prediction market MVP built with Go, Gin, gRPC, Kafka, WebSocket, and BSC testnet settlement.

## Service roles

- `api`: public HTTP API powered by Gin
- `matching`: Kafka-driven matching center, single writer per order book
- `account`: gRPC account service for mutable balances, freezing, and idempotent external credits / debits
- `ledger`: append-only journal and reserve reconciliation service
- `settlement`: market resolution and payout event service
- `chain`: BSC vault listener and deposit credit service
- `ws`: websocket quote fanout service

## Core architecture

- Order placement does **not** call matching synchronously over gRPC
- `api` calls `account` over gRPC for pre-trade freezing, then publishes order commands to Kafka
- `matching` consumes Kafka commands in order and emits trade / order / quote events back to Kafka
- `ws` and downstream services consume event topics for push, settlement, and bookkeeping
- `account` holds the current balance snapshot, but `ledger` remains the final evidence chain for replay and reconciliation
- `trade.matched` is now the first ledger ingress event; `ledger` converts fills into append-only cash postings
- `account` now mirrors `order.event` and `trade.matched` to maintain freeze consumption and leftover release asynchronously
- `settlement` consumes `position.changed` and `market.event`, then emits `settlement.completed`
- V1 deposit direction is now `frontend -> BSC vault contract -> chain listener -> account credit`
- wallet should only pop for session authorization, deposit, withdraw, and claim
- gRPC remains available for control-plane and internal query use, not hot-path order ingress

## Current gRPC surface

- `account`: `PreFreeze`, `ReleaseFreeze`, `GetBalance`, `CreditBalance`, `DebitBalance`
- `api` dials `FUNNYOPTION_ACCOUNT_GRPC_ADDR`, defaulting to the `account` service gRPC address

## Quick start

```bash
go mod tidy
go run ./cmd/api
go run ./cmd/ws
go run ./cmd/matching
go run ./cmd/account
go run ./cmd/ledger
go run ./cmd/settlement
go run ./cmd/chain
```

## One-click local dev

```bash
/Users/zhangza/code/funnyoption/scripts/dev-up.sh
```

- starts `account / matching / ledger / settlement / chain / api / ws / web`
- starts a managed local `anvil` chain too when `FUNNYOPTION_LOCAL_CHAIN_MODE=anvil`
- applies local PostgreSQL migrations before boot
- builds local Go service binaries into `/Users/zhangza/code/funnyoption/.run/dev/bin`
- reuses an existing local `zookeeper + kafka-broker` first, otherwise falls back to the bundled local Kafka compose
- writes `/Users/zhangza/code/funnyoption/web/.env.local` for the frontend
- stores logs in `/Users/zhangza/code/funnyoption/.logs/dev`
- stores pid files in `/Users/zhangza/code/funnyoption/.run/dev`

Stop everything:

```bash
/Users/zhangza/code/funnyoption/scripts/dev-down.sh
```

Check process status:

```bash
/Users/zhangza/code/funnyoption/scripts/dev-status.sh
```

## Persistent local chain

To switch the stack onto one persistent local chain instead of the default external BSC-testnet-style config:

1. set `FUNNYOPTION_LOCAL_CHAIN_MODE=anvil` in `/Users/zhangza/code/funnyoption/.env.local`
2. run `/Users/zhangza/code/funnyoption/scripts/dev-up.sh`

The stack will then:

- start `anvil` on `127.0.0.1:8545`
- deploy local `MockUSDT` and `FunnyVault`
- generate `/Users/zhangza/code/funnyoption/.run/dev/local-chain.env`
- generate `/Users/zhangza/code/funnyoption/.run/dev/local-chain-wallets.env`

Detailed runbook:

- [/Users/zhangza/code/funnyoption/docs/operations/local-persistent-chain.md](/Users/zhangza/code/funnyoption/docs/operations/local-persistent-chain.md)

## Web frontend

```bash
cd /Users/zhangza/code/funnyoption/web
npm install
npm run dev
```

- dev frontend now writes to `/Users/zhangza/code/funnyoption/web/.next-dev`
- production builds still use `/Users/zhangza/code/funnyoption/web/.next`
- this avoids `next build` breaking a live `next dev` session and causing missing CSS / chunk 404s

- default frontend URL: `http://127.0.0.1:3000`
- default API target: `http://127.0.0.1:8080`
- override with:
  - `NEXT_PUBLIC_API_BASE_URL`
  - `NEXT_PUBLIC_DEFAULT_USER_ID`

Current frontend routes:

- `/`: tape home, lead market, trade tape, architecture summary
- `/markets/:marketId`: market detail + order ticket
- `/portfolio`: balances, positions, direct vault controls, session deck, payout claim desk, live chain queue
- `/control`: operator queue, live chain tasks, resolution board

Current frontend interaction model:

- wallet connect via browser `window.ethereum`
- session key authorization via wallet `personal_sign`
- local session key storage for MVP browser sessions
- session-signed order submission when an active session exists
- fallback dev order path (`user_id=1001`) when no session exists
- direct `approve -> deposit` and `queueWithdrawal` contract calls from the browser
- backend session revoke wired through the frontend session deck
- chain task boards poll the queue so claim submission feedback is visible without a hard refresh

Suggested frontend chain envs:

- `NEXT_PUBLIC_CHAIN_ID=97`
- `NEXT_PUBLIC_CHAIN_NAME=BSC Testnet`
- `NEXT_PUBLIC_VAULT_ADDRESS=0x...`
- `NEXT_PUBLIC_COLLATERAL_TOKEN_ADDRESS=0x...`
- `NEXT_PUBLIC_COLLATERAL_SYMBOL=USDT`
- `NEXT_PUBLIC_COLLATERAL_DECIMALS=6`
- `NEXT_PUBLIC_CHAIN_EXPLORER_URL=https://testnet.bscscan.com`
- `NEXT_PUBLIC_CHAIN_RPC_URL=https://data-seed-prebsc-1-s1.bnbchain.org:8545`

## WebSocket

- endpoint: `GET /ws?stream=depth&book_key=<market_id>:<outcome>`
- endpoint: `GET /ws?stream=ticker&book_key=<market_id>:<outcome>`
- endpoint: `GET /ws?stream=market&market_id=<market_id>`
- `stream=market` will push both `market.event` and `settlement.completed` envelopes

## HTTP API

- `POST /api/v1/markets`: create a market directly in PostgreSQL
- `GET /api/v1/markets`: list markets with optional `status`, `created_by`, `limit`
- `GET /api/v1/markets/:market_id`: fetch market detail
- `POST /api/v1/sessions`: verify wallet signature and register a browser session key
- `GET /api/v1/sessions`: list wallet session grants with optional `user_id`, `wallet_address`, `status`, `limit`
- `POST /api/v1/sessions/:session_id/revoke`: revoke an active session grant on the backend
- `GET /api/v1/deposits`: list mirrored on-chain deposits with optional `user_id`, `wallet_address`, `status`, `limit`
- `GET /api/v1/withdrawals`: list mirrored on-chain withdrawal queue events with optional `user_id`, `wallet_address`, `status`, `limit`
- `GET /api/v1/chain-transactions`: inspect claim / chain task queue with optional `biz_type`, `ref_id`, `status`, `limit`
- `POST /api/v1/orders`: pre-freeze via `account` then enqueue `order.command`
- `GET /api/v1/orders`: list orders with optional `user_id`, `market_id`, `status`, `limit`
- `GET /api/v1/trades`: list trades with optional `user_id`, `market_id`, `outcome`, `limit`
- `GET /api/v1/balances`: list balances with `user_id` and optional `asset`, `limit`
- `GET /api/v1/positions`: list positions with `user_id` and optional `market_id`, `outcome`, `limit`
- `GET /api/v1/payouts`: list settlement payouts with `user_id` and optional `market_id`, `limit`
- `POST /api/v1/payouts/:event_id/claim`: enqueue a vault claim request for an existing payout
- `GET /api/v1/freezes`: inspect freeze records with optional `user_id`, `status`, `limit`
- `GET /api/v1/ledger/entries`: inspect ledger entries with optional `biz_type`, `ref_id`, `limit`
- `GET /api/v1/ledger/entries/:entry_id/postings`: inspect postings under a ledger entry
- `GET /api/v1/reports/liabilities`: summarize current internal liabilities by asset
- `POST /api/v1/markets/:market_id/resolve`: publish `market.event` for settlement

## Deposit ingestion foundation

- `chain-service` persists confirmed vault deposits into `chain_deposits`
- `chain-service` persists confirmed vault withdrawal queue events into `chain_withdrawals`
- `chain-service` now polls the configured BSC vault `Deposited(address,uint256)` event
- `chain-service` now also polls `WithdrawalQueued(bytes32,address,uint256,address)`
- `chain-service` now supports primary + fallback BSC RPC endpoints
- `chain-service` can pick pending `CLAIM` rows from `chain_transactions` and submit `processClaim` to the vault with the configured operator key
- active `wallet_sessions` provide wallet -> user resolution for deposit crediting
- `account.CreditBalance` and `account.DebitBalance` apply idempotent external balance mutations via `account_balance_events`
- `ledger` consumes `chain.deposit` and `chain.withdrawal` and appends custody evidence entries

## Environment bootstrap

- root env example: `/Users/zhangza/code/funnyoption/.env.example`
- local ignored remote env: `/Users/zhangza/code/funnyoption/.env.test.remote`
- test env example: `/Users/zhangza/code/funnyoption/configs/test/funnyoption.env.example`
- postgres bootstrap: `/Users/zhangza/code/funnyoption/docs/deploy/postgres.md`
- schema apply script: `/Users/zhangza/code/funnyoption/scripts/apply_migrations.sh`
- direct deposit/session-key design: `/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md`
