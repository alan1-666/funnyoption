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
- [deploy/staging/server-deploy-entrypoint.sh](/Users/zhangza/code/funnyoption/deploy/staging/server-deploy-entrypoint.sh)
- [scripts/deploy-staging.sh](/Users/zhangza/code/funnyoption/scripts/deploy-staging.sh)

### Workflow behavior

- `push` to `main` triggers a staging deployment.
- `workflow_dispatch` can redeploy a specific commit SHA or git ref through
  the `deploy_ref` input, and can force a full rebuild with
  `deploy_scope=full`.
- GitHub Actions is now a thin trigger:
  - it resolves the exact target SHA/ref
  - it configures SSH with pinned host verification
  - it invokes `/usr/local/bin/funnyoption-staging-deploy --repo /opt/funnyoption-staging --ref <target>`
  - it appends `--all-services` only for the manual full-deploy override
- the fixed server entrypoint is the orchestration source of truth:
  - it acquires `/var/lock/funnyoption-staging-deploy.lock` with `flock`
  - it rejects tracked/staged checkout drift before deploy
  - it records the current deployed repo `HEAD` as `diff_base`
  - it fetches `origin` and checks out the exact requested target commit in detached `HEAD`
  - it delegates to the checked-out repo script with `--skip-git-sync` and either `--diff-base <previous_head>` or `--all-services`
- ref resolution on the host entrypoint is intentionally asymmetric:
  - raw commit SHAs still resolve exactly as supplied
  - symbolic branch refs like `main` or `refs/heads/main` first resolve against the freshly fetched remote-tracking ref such as `origin/main`
  - only after that does the entrypoint fall back to a same-named local ref when that is explicitly reasonable
- selective service planning, docs-only no-op deploys, migration decisions, and
  post-deploy HTTP smoke checks still live in
  [scripts/deploy-staging.sh](/Users/zhangza/code/funnyoption/scripts/deploy-staging.sh).
- GitHub Actions no longer embeds the selective plan or validation matrix; the
  live deploy path now lives on the server entrypoint plus the repo deploy
  script.

### Path-to-service map

| Changed path | Rebuild / restart services | Validation |
| --- | --- | --- |
| `cmd/api/**`, `internal/api/**` | `api` | Go tests |
| `cmd/ws/**`, `internal/ws/**` | `ws` | Go tests |
| `cmd/matching/**`, `internal/matching/**` | `matching` | Go tests |
| `cmd/account/**`, `internal/account/model/**`, `internal/account/repository/**`, `internal/account/service/**` | `account` | Go tests |
| `internal/account/client/**`, `internal/gen/accountv1/**` | `account`, `api`, `chain` | Go tests |
| `cmd/ledger/**`, `internal/ledger/**` | `ledger` | Go tests |
| `cmd/settlement/**`, `internal/settlement/**` | `settlement` | Go tests |
| `cmd/chain/**`, `internal/chain/**` | `chain` | Go tests |
| `web/**` except `web/package.json` and `web/package-lock.json` | `web` | `web` build |
| `web/package.json`, `web/package-lock.json` | `web`, `admin` | `web` and `admin` builds |
| `admin/**` | `admin` | `admin` build |
| `migrations/**` | `account`, `matching`, `ledger`, `settlement`, `chain`, `api`, `ws` | run `migrate` profile |

### Fallback policy

- `go.mod`, `go.sum`, `internal/shared/**`, and `proto/**` trigger all backend
  services.
- `deploy/docker/**`, `deploy/staging/**`, and `scripts/deploy-staging.sh`
  trigger all app services plus the `migrate` profile.
- unclassified `*.go`, `cmd/**`, or `internal/**` changes fall back to all
  backend services instead of risking a missed rebuild.
- if the requested diff base commit is unavailable, the plan falls back to a
  full service deploy.
- docs-only pushes, including `docs/harness/**`, produce `skip_deploy=1` and do
  not rebuild or restart services.

### Required GitHub Secrets

Configure these in the `staging` environment or repository secrets:

- `STAGING_SSH_HOST`: staging server hostname or IP
- `STAGING_SSH_USER`: SSH login user
- `STAGING_SSH_PRIVATE_KEY`: private key for the deploy user
- `STAGING_SSH_PORT`: optional SSH port, defaults to `22`
- `STAGING_SSH_KNOWN_HOSTS`: optional pinned `known_hosts` entry

Do not store `FUNNYOPTION_CHAIN_OPERATOR_PRIVATE_KEY`, database passwords, or
other runtime secrets in GitHub unless the server bootstrap process explicitly
needs them there. The deploy workflow expects those values to stay in the
server-only env file described below.

### One-time server setup

Prepare the server once before enabling the workflow:

1. Clone this repository to `/opt/funnyoption-staging`.
2. Create `deploy/staging/.env.staging` inside that clone from
   [configs/staging/funnyoption.env.example](/Users/zhangza/code/funnyoption/configs/staging/funnyoption.env.example)
   and fill all secret-bearing values on the server only.
3. Install or refresh the fixed host-side entrypoint and lock file:

   ```bash
   sudo install -m 0755 /opt/funnyoption-staging/deploy/staging/server-deploy-entrypoint.sh /usr/local/bin/funnyoption-staging-deploy
   sudo install -o <deploy-user> -g <deploy-group> -m 0664 /dev/null /var/lock/funnyoption-staging-deploy.lock
   ```

4. Ensure the deploy user can run `git fetch`, `docker compose`, `curl`,
   `flock`, and `/usr/local/bin/funnyoption-staging-deploy`.
5. Ensure the deploy user can write `/var/lock/funnyoption-staging-deploy.lock`.
6. Ensure the server has outbound HTTPS access to container registries, npm
   package downloads, and `fonts.googleapis.com` during image builds.
7. Install the reverse proxy config from
   [deploy/staging/funnyoption.xyz.conf](/Users/zhangza/code/funnyoption/deploy/staging/funnyoption.xyz.conf)
   and keep TLS termination outside the repo.

Normal app deploys reuse the installed `/usr/local/bin/funnyoption-staging-deploy`
path. If a future change edits
[deploy/staging/server-deploy-entrypoint.sh](/Users/zhangza/code/funnyoption/deploy/staging/server-deploy-entrypoint.sh)
itself, rerun the `sudo install ... /usr/local/bin/funnyoption-staging-deploy`
command after checking out the intended repo version on the server.

### Manual deploy and rollback

Manual deploy of one exact target:

```bash
/usr/local/bin/funnyoption-staging-deploy --repo /opt/funnyoption-staging --ref <commit-sha-or-ref>
```

When `--ref` is a branch-like symbolic ref such as `main` or
`refs/heads/main`, the entrypoint deploys the freshly fetched remote branch tip
first. Raw commit SHAs keep their exact-SHA behavior unchanged.

Redeploy the currently checked-out commit as a full stack refresh:

```bash
/usr/local/bin/funnyoption-staging-deploy \
  --repo /opt/funnyoption-staging \
  --ref "$(git -C /opt/funnyoption-staging rev-parse HEAD)" \
  --all-services
```

Rollback to a known-good commit:

```bash
/usr/local/bin/funnyoption-staging-deploy --repo /opt/funnyoption-staging --ref <previous-good-sha>
```

Force a full rollback rebuild if you do not want diff-based selection:

```bash
/usr/local/bin/funnyoption-staging-deploy --repo /opt/funnyoption-staging --ref <previous-good-sha> --all-services
```

Preview the selective plan from the current checkout without touching compose:

```bash
cd /opt/funnyoption-staging
./scripts/deploy-staging.sh --skip-git-sync --print-plan --diff-base HEAD^
```

If you need a narrow, manual service-only intervention outside the standard
entrypoint flow, use the checked-out repo script directly:

```bash
cd /opt/funnyoption-staging
./scripts/deploy-staging.sh --skip-git-sync --service api --skip-migrations
```

If deployment fails before these commands can be exercised, the usual blocker
is one of these missing server-specific values:

- `STAGING_SSH_HOST`
- `STAGING_SSH_USER`
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
  - For redeploys, keep it as a lower-bound bootstrap value only. The chain
    listener now persists its restart cursor in `chain_listener_cursors` and
    resumes from `max(FUNNYOPTION_CHAIN_START_BLOCK, persisted_next_block)`.
    Lowering `FUNNYOPTION_CHAIN_START_BLOCK` alone no longer rewinds the scan
    cursor once a checkpoint row exists.

### Chain listener restart and recovery

Steady-state restart behavior:

- `chain` logs `vault scan cursor initialized` with the configured lower-bound
  start block, the persisted checkpoint, and the effective `next_block`.
- after each confirmed block-range scan, `chain` updates
  `chain_listener_cursors.next_block` so another restart can resume without
  replaying from a stale static block.
- if the RPC returns `History has been pruned for this block`, `chain` logs
  `skip pruned vault history`, fast-forwards `next_block` to `safeHead + 1`,
  persists that cursor, and continues with new blocks.

One-time recovery when staging was already wedged on an old pruned start block:

1. Read the current cursor row and the chain logs:

   ```bash
   docker compose \
     --env-file deploy/staging/.env.staging \
     -f deploy/staging/docker-compose.staging.yml \
     exec postgres \
     sh -lc 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT chain_name, network_name, vault_address, next_block, updated_at FROM chain_listener_cursors ORDER BY updated_at DESC;"'

   docker compose \
     --env-file deploy/staging/.env.staging \
     -f deploy/staging/docker-compose.staging.yml \
     logs --since 30m chain
   ```

   The `postgres` service does not load `deploy/staging/.env.staging`, so do
   not use `psql "$FUNNYOPTION_POSTGRES_DSN"` inside that container unless you
   explicitly inject the DSN from the host shell first:

   ```bash
   set -a
   source deploy/staging/.env.staging
   set +a

   docker compose \
     --env-file deploy/staging/.env.staging \
     -f deploy/staging/docker-compose.staging.yml \
     exec -e FUNNYOPTION_POSTGRES_DSN postgres \
     sh -lc 'psql "$FUNNYOPTION_POSTGRES_DSN" -c "SELECT 1;"'
   ```

2. If no cursor row exists yet and the static start block is already pruned,
   restart `chain` after deploying the fix and confirm the fast-forward log.
   If you intentionally skip an old range, record the exact `[from_block,
   to_block]` interval from `skip pruned vault history` in the handoff. The
   tradeoff is that deposit / withdrawal events that exist only inside that
   skipped pruned range cannot be replayed from the current public RPC and
   require an archival RPC or manual backfill.

3. If you need an explicit manual fast-forward before restart, update only the
   target vault row and document the skipped range:

   ```bash
   docker compose \
     --env-file deploy/staging/.env.staging \
     -f deploy/staging/docker-compose.staging.yml \
     exec postgres \
     psql -U funnyoption -d funnyoption \
     -c "INSERT INTO chain_listener_cursors (chain_name, network_name, vault_address, next_block, updated_at)
         VALUES ('bsc', 'testnet', lower('<vault-address>'), <next-block>, EXTRACT(EPOCH FROM NOW())::BIGINT)
         ON CONFLICT (chain_name, network_name, vault_address) DO UPDATE
         SET next_block = GREATEST(chain_listener_cursors.next_block, EXCLUDED.next_block),
             updated_at = EXCLUDED.updated_at;"
   ```

## Current staging assets

This repo now ships a staging compose stack and a deploy helper:

- service Dockerfiles in [deploy/docker](/Users/zhangza/code/funnyoption/deploy/docker)
- staging compose file in [deploy/staging/docker-compose.staging.yml](/Users/zhangza/code/funnyoption/deploy/staging/docker-compose.staging.yml)
- reverse proxy template in [deploy/staging/funnyoption.xyz.conf](/Users/zhangza/code/funnyoption/deploy/staging/funnyoption.xyz.conf)
- server env template in [configs/staging/funnyoption.env.example](/Users/zhangza/code/funnyoption/configs/staging/funnyoption.env.example)
