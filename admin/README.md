# FunnyOption Admin Service

This is the dedicated operator runtime extracted out of the public `web` shell.

## Local model

- the service lives under `/admin`
- it runs as its own Next.js app on port `3001` by default
- it reuses the already-installed `web/node_modules` tree through `NODE_PATH`, so local dev does not need a second package install
- market creation, first-liquidity/bootstrap, and resolution now go through admin-owned API routes under `/api/operator/**`
- this Next.js runtime is the single supported operator entrypoint; the older Go/template runtime is no longer a supported admin surface

## Auth model

- operators connect an EIP-1193 wallet
- the admin UI checks the connected wallet against `FUNNYOPTION_OPERATOR_WALLETS`
- each create/resolve/first-liquidity action is signed with `personal_sign` before the admin backend proxies the action to the core API
- the admin backend forwards the signed operator proof to the core API in a shared JSON shape:
  - `operator.wallet_address`
  - `operator.requested_at`
  - `operator.signature`
- the core API independently rebuilds the signed action message, re-verifies the wallet signature, and enforces the same allowlist on:
  - `POST /api/v1/markets`
  - `POST /api/v1/markets/:market_id/resolve`
  - `POST /api/v1/admin/markets/:market_id/first-liquidity`
- the UI surfaces the connected wallet identity and explicit allow/deny state

## Startup

Run the shared local stack:

```bash
/Users/zhangza/code/funnyoption/scripts/dev-up.sh
```

That starts:

- public web on `http://127.0.0.1:3000`
- admin service on `http://127.0.0.1:3001`

You can also run the admin service alone:

```bash
cd /Users/zhangza/code/funnyoption/admin
npm run dev -- --hostname 127.0.0.1 --port 3001
```

## Required env

- `FUNNYOPTION_OPERATOR_WALLETS`
  - comma-separated wallet allowlist for operator actions
- `FUNNYOPTION_DEFAULT_OPERATOR_USER_ID`
  - numeric user id written into `created_by` when admin creates a market

## Current limitation

The wallet gate now reaches the shared core API for market create/resolve/first-liquidity. This task intentionally does not widen into general user order auth, so `POST /api/v1/orders` still uses the older trust model and remains a separate follow-up if direct operator order ingress must be hardened later.
