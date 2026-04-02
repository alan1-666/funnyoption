# FunnyOption Testing Deployment on BSC Testnet

This runbook describes the current recommended way to deploy FunnyOption to a
server-side testing environment while keeping the chain integration pointed at
BSC Testnet (`chain_id=97`).

## Target topology

The current repo expects these runtime pieces:

- PostgreSQL
- Redis
- Kafka
- core Go services
  - `api`
  - `ws`
  - `matching`
  - `account`
  - `ledger`
  - `settlement`
  - `chain`
- frontend services
  - `web`
  - `admin`

Recommended public endpoints:

- `https://api-staging.example.com` -> `api:8080`
- `https://ws-staging.example.com` -> `ws:8081`
- `https://app-staging.example.com` -> `web:3000`
- `https://admin-staging.example.com` -> `admin:3001`

## Required secrets and addresses

Before building or starting containers, prepare these values:

- PostgreSQL DSN
- Kafka broker list
- Redis address
- BSC Testnet RPC URL
- chain operator private key
- deployed `FunnyVault` address on BSC Testnet
- collateral token address on BSC Testnet
- operator wallet allowlist
- default operator user id

The backend runtime template lives here:

- [configs/staging/funnyoption.env.example](/Users/zhangza/code/funnyoption/configs/staging/funnyoption.env.example)

Copy it to a server-only file such as `.env.staging`, then replace all
`replace-me` placeholders.

## Build-time frontend env

`web` and `admin` embed their public chain and API settings at build time.
That means you must pass the staging URLs and BSC Testnet values with
`--build-arg` during image build.

### Web build example

```bash
docker build \
  -f deploy/docker/web.Dockerfile \
  -t funnyoption-web:staging \
  --build-arg NEXT_PUBLIC_API_BASE_URL=https://api-staging.example.com \
  --build-arg NEXT_PUBLIC_WS_BASE_URL=https://ws-staging.example.com \
  --build-arg NEXT_PUBLIC_ADMIN_BASE_URL=https://admin-staging.example.com \
  --build-arg NEXT_PUBLIC_CHAIN_ID=97 \
  --build-arg NEXT_PUBLIC_CHAIN_NAME="BSC Testnet" \
  --build-arg NEXT_PUBLIC_CHAIN_RPC_URL=https://data-seed-prebsc-1-s1.bnbchain.org:8545 \
  --build-arg NEXT_PUBLIC_CHAIN_EXPLORER_URL=https://testnet.bscscan.com \
  --build-arg NEXT_PUBLIC_VAULT_ADDRESS=0xreplaceMe \
  --build-arg NEXT_PUBLIC_COLLATERAL_TOKEN_ADDRESS=0xreplaceMe \
  --build-arg NEXT_PUBLIC_COLLATERAL_SYMBOL=USDT \
  --build-arg NEXT_PUBLIC_COLLATERAL_DECIMALS=6 \
  --build-arg NEXT_PUBLIC_COLLATERAL_ACCOUNTING_DECIMALS=2 \
  --build-arg NEXT_PUBLIC_NATIVE_CURRENCY_NAME=BNB \
  --build-arg NEXT_PUBLIC_NATIVE_CURRENCY_SYMBOL=tBNB \
  --build-arg NEXT_PUBLIC_NATIVE_CURRENCY_DECIMALS=18 \
  .
```

### Admin build example

```bash
docker build \
  -f deploy/docker/admin.Dockerfile \
  -t funnyoption-admin:staging \
  --build-arg NEXT_PUBLIC_API_BASE_URL=https://api-staging.example.com \
  --build-arg NEXT_PUBLIC_WS_BASE_URL=https://ws-staging.example.com \
  --build-arg NEXT_PUBLIC_PUBLIC_WEB_BASE_URL=https://app-staging.example.com \
  --build-arg NEXT_PUBLIC_DEFAULT_OPERATOR_USER_ID=1001 \
  --build-arg NEXT_PUBLIC_OPERATOR_WALLETS=0xreplaceMe \
  --build-arg NEXT_PUBLIC_CHAIN_ID=97 \
  --build-arg NEXT_PUBLIC_CHAIN_NAME="BSC Testnet" \
  --build-arg NEXT_PUBLIC_CHAIN_RPC_URL=https://data-seed-prebsc-1-s1.bnbchain.org:8545 \
  --build-arg NEXT_PUBLIC_CHAIN_EXPLORER_URL=https://testnet.bscscan.com \
  --build-arg NEXT_PUBLIC_COLLATERAL_SYMBOL=USDT \
  --build-arg NEXT_PUBLIC_COLLATERAL_ACCOUNTING_DECIMALS=2 \
  --build-arg NEXT_PUBLIC_NATIVE_CURRENCY_NAME=BNB \
  --build-arg NEXT_PUBLIC_NATIVE_CURRENCY_SYMBOL=tBNB \
  --build-arg NEXT_PUBLIC_NATIVE_CURRENCY_DECIMALS=18 \
  --build-arg FUNNYOPTION_DEFAULT_OPERATOR_USER_ID=1001 \
  --build-arg FUNNYOPTION_OPERATOR_WALLETS=0xreplaceMe \
  .
```

## Backend image builds

Build the Go services from the repo root:

```bash
docker build -f deploy/docker/api.Dockerfile -t funnyoption-api:staging .
docker build -f deploy/docker/ws.Dockerfile -t funnyoption-ws:staging .
docker build -f deploy/docker/matching.Dockerfile -t funnyoption-matching:staging .
docker build -f deploy/docker/account.Dockerfile -t funnyoption-account:staging .
docker build -f deploy/docker/ledger.Dockerfile -t funnyoption-ledger:staging .
docker build -f deploy/docker/settlement.Dockerfile -t funnyoption-settlement:staging .
docker build -f deploy/docker/chain.Dockerfile -t funnyoption-chain:staging .
```

## Deployment order

Recommended startup order on the server:

1. Start PostgreSQL, Redis, and Kafka.
2. Apply SQL migrations against the staging database.
3. Start `account`, `matching`, `ledger`, `settlement`, and `chain`.
4. Start `api` and `ws`.
5. Start `web` and `admin`.
6. Put the reverse proxy in front of the public services.

## Migrations

Run the SQL files in `migrations/` in order against the staging database.

The latest migrations currently include:

- taxonomy and option schema
- user profile table

Do not skip them, because the current frontend and admin flows expect them.

## Reverse proxy expectations

At minimum, the proxy should:

- terminate TLS
- forward `app-staging` to port `3000`
- forward `admin-staging` to port `3001`
- forward `api-staging` to port `8080`
- forward `ws-staging` to port `8081`
- preserve `Host` and `X-Forwarded-*` headers

If WebSocket upgrades are terminated at the proxy, remember to allow upgrade
headers for the `ws` service.

## Smoke checklist

After the stack is up, verify these in order:

1. `api` health and public market reads work.
2. `web` loads and shows BSC Testnet as the target chain.
3. `admin` wallet gate only allows `FUNNYOPTION_OPERATOR_WALLETS`.
4. operator can create a market through `admin`.
5. user can connect wallet, create session, and place an order.
6. chain listener sees deposits from the configured vault.
7. market resolution produces settlement payouts.
8. claim flow can submit a chain task without zero-address regressions.

## BSC Testnet-specific notes

- The repo default for testing is already BSC Testnet:
  - `FUNNYOPTION_CHAIN_NAME=bsc`
  - `FUNNYOPTION_NETWORK_NAME=testnet`
  - `FUNNYOPTION_CHAIN_ID=97`
  - `FUNNYOPTION_FRONTEND_CHAIN_NAME=BSC Testnet`
- `FUNNYOPTION_CHAIN_CONFIRMATIONS=6` is a reasonable testing default.
- `FUNNYOPTION_CHAIN_START_BLOCK` should be set deliberately on staging.
  - Use `0` only for first-time bring-up or disposable environments.
  - For redeploys, move it forward so `chain` does not rescan a huge range.

## Current known gaps

This repo now has service-level Dockerfiles, but it does not yet ship a
blessed `docker-compose.yml` or Kubernetes manifests. For the first staging
deployment, the fastest path is:

- use the Dockerfiles in [deploy/docker](/Users/zhangza/code/funnyoption/deploy/docker)
- use the env template in [configs/staging/funnyoption.env.example](/Users/zhangza/code/funnyoption/configs/staging/funnyoption.env.example)
- wire the services together in your server-side orchestration of choice

