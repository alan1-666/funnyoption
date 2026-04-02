# Local Lifecycle Runbook

## Purpose

Run one reproducible local lifecycle that covers:

- admin market creation
- wallet session authorization
- deposit credit and balance increase
- order placement
- trade matching
- market resolution
- settlement payout and read-surface updates

## Prerequisites

1. Start the local stack with:

```bash
/Users/zhangza/code/funnyoption/scripts/dev-up.sh
```

2. Make sure the default API responds on `http://127.0.0.1:8080/healthz`.

## Run

```bash
cd /Users/zhangza/code/funnyoption
set -a
source .env.local
set +a
go run ./cmd/local-lifecycle
```

## What the runner does

1. Creates a fresh market through `POST /api/v1/markets`
2. Creates two wallet-backed session grants through `POST /api/v1/sessions`
3. Deploys an ephemeral proof vault on an in-process simulated chain
4. Submits a real wallet-signed deposit transaction for the buyer user
5. Boots a real `DepositListener` and waits for the credited deposit row and balance delta
6. Issues explicit first-liquidity inventory for the seller user through the admin path
7. Places a resting `SELL` order and a crossing `BUY` order
8. Waits for the trade to appear through `GET /api/v1/trades`
9. Resolves the market through `POST /api/v1/markets/:market_id/resolve`
10. Waits for payout, position settlement, and resolved market state through the read APIs

## Important local caveats

- Default `.env.local` still does not include a persistent external `FUNNYOPTION_CHAIN_RPC_URL` or `FUNNYOPTION_VAULT_ADDRESS`.
- The runner works around that by creating its own in-process listener-proof chain for the deposit step, with `chain_id=1337`, `chain_name=simulated`, and `network_name=local-proof`.
- The proof vault is local-only and exists to emit the canonical `Deposited(address,uint256)` event shape, so the listener path is truthful even though the default stack is not pointed at a shared BSC testnet vault.
- The current matching model is `BUY` versus `SELL` on the same outcome. To guarantee a first trade in local smoke, the runner now issues explicit first-liquidity inventory before placing the resting `SELL`.

## Output

The command prints a JSON summary with:

- proof environment metadata
- market id and final market state
- buyer and seller user ids
- deposit evidence
- buyer and seller order states
- matched trade
- buyer position state
- payout record
- buyer balance before deposit, after deposit, and after settlement
