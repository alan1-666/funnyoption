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

## Current deployed domains

The current server deployment is reachable at:

- User web: [https://funnyoption.xyz/](https://funnyoption.xyz/)
- Admin web: [https://admin.funnyoption.xyz/](https://admin.funnyoption.xyz/)

Record environment-specific API / WebSocket upstream details in server-side
deployment config or GitHub Actions secrets, not in plaintext docs.

## Push-to-deploy CI/CD

GitHub push-to-deploy is wired through:

- [.github/workflows/staging-deploy.yml](/Users/zhangza/code/funnyoption/.github/workflows/staging-deploy.yml)
- [scripts/deploy-staging.sh](/Users/zhangza/code/funnyoption/scripts/deploy-staging.sh)

### Workflow behavior

- `push` to `main` triggers a staging deployment.
- `workflow_dispatch` can redeploy a specific commit SHA or git ref through
  the `deploy_ref` input.
- the `validate` job runs:
  - `go test ./...`
  - `npm ci && npm run build` for `web`
  - `npm run build` for `admin`
  - `bash -n scripts/*.sh`
- the `deploy-staging` job SSHes into the staging server, checks out the target
  ref in the server-side repo clone, runs migrations, rebuilds the compose
  stack, and performs HTTP smoke checks.

### Required GitHub Secrets

Configure these in the `staging` environment or repository secrets:

- `STAGING_SSH_HOST`: staging server hostname or IP
- `STAGING_SSH_USER`: SSH login user
- `STAGING_SSH_PRIVATE_KEY`: private key for the deploy user
- `STAGING_DEPLOY_PATH`: absolute path to the server-side repo clone
- `STAGING_SSH_PORT`: optional SSH port, defaults to `22`
- `STAGING_SSH_KNOWN_HOSTS`: optional pinned `known_hosts` entry

Do not store `FUNNYOPTION_CHAIN_OPERATOR_PRIVATE_KEY`, database passwords, or
other runtime secrets in GitHub unless the server bootstrap process explicitly
needs them there. The deploy workflow expects those values to stay in the
server-only env file described below.

### One-time server setup

Prepare the server once before enabling the workflow:

1. Clone this repository to the path stored in `STAGING_DEPLOY_PATH`.
2. Create `deploy/staging/.env.staging` inside that clone from
   [configs/staging/funnyoption.env.example](/Users/zhangza/code/funnyoption/configs/staging/funnyoption.env.example)
   and fill all secret-bearing values on the server only.
3. Ensure the deploy user can run `git fetch`, `docker compose`, and `curl`.
4. Ensure the server has outbound HTTPS access to container registries, npm
   package downloads, and `fonts.googleapis.com` during image builds.
5. Install the reverse proxy config from
   [deploy/staging/funnyoption.xyz.conf](/Users/zhangza/code/funnyoption/deploy/staging/funnyoption.xyz.conf)
   and keep TLS termination outside the repo.

### Manual deploy and rollback

From the server-side repo clone:

```bash
FUNNYOPTION_DEPLOY_REF=origin/main ./scripts/deploy-staging.sh
```

Rollback to a known-good commit:

```bash
./scripts/deploy-staging.sh --ref <previous-good-sha>
```

If the new release is already checked out and you only want to re-run compose
with the local tree as-is:

```bash
./scripts/deploy-staging.sh --skip-git-sync
```

If deployment fails before these commands can be exercised, the usual blocker
is one of these missing server-specific values:

- `STAGING_SSH_HOST`
- `STAGING_SSH_USER`
- `STAGING_DEPLOY_PATH`
- the server-local `deploy/staging/.env.staging`

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

## Current staging assets

This repo now ships a staging compose stack and a deploy helper:

- service Dockerfiles in [deploy/docker](/Users/zhangza/code/funnyoption/deploy/docker)
- staging compose file in [deploy/staging/docker-compose.staging.yml](/Users/zhangza/code/funnyoption/deploy/staging/docker-compose.staging.yml)
- reverse proxy template in [deploy/staging/funnyoption.xyz.conf](/Users/zhangza/code/funnyoption/deploy/staging/funnyoption.xyz.conf)
- server env template in [configs/staging/funnyoption.env.example](/Users/zhangza/code/funnyoption/configs/staging/funnyoption.env.example)
