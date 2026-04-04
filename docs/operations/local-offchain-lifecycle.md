# Local Off-Chain Lifecycle

## Goal

Run one deterministic local proof for:

1. admin creates a market
2. wallet-style sessions are authorized
3. a deposit-style credit lands
4. orders are placed and matched
5. the market resolves
6. settlement updates the terminal reads

## Persistent local-chain mode

If you set `FUNNYOPTION_LOCAL_CHAIN_MODE=anvil`, the repo now supports a persistent local-chain path in addition to the older in-process proof environment.

See:

- [/Users/zhangza/code/funnyoption/docs/operations/local-persistent-chain.md](/Users/zhangza/code/funnyoption/docs/operations/local-persistent-chain.md)

In that mode:

- `scripts/dev-up.sh` starts a managed `anvil` node and deploys local contracts
- `scripts/local-lifecycle.sh` sources `/Users/zhangza/code/funnyoption/.run/dev/local-chain.env`
- `cmd/local-lifecycle` uses the persistent local vault and running `chain-service` instead of spinning up its own simulated proof chain

## Important local truth

The default local `.env.local` does **not** include a configured `FUNNYOPTION_VAULT_ADDRESS` or collateral token address.

That means the default local stack still does **not** boot a persistent external `chain-service` listener against a shared RPC / vault target.

Instead, `cmd/local-lifecycle` now provisions its own deterministic listener-proof environment for the deposit step:

- session authorization is real and uses the same message/signature rules as the browser wallet flow
- deposit credit is driven by a real wallet-signed transaction against an in-process proof vault on a go-ethereum simulated chain
- the command boots a real `DepositListener` against that proof chain and waits for the normal listener -> processor -> account credit path to land
- fresh-market first liquidity is issued explicitly through the admin first-liquidity path by debiting operator collateral, minting paired `YES` / `NO` inventory, and queueing the bootstrap `SELL` order in one shot
- matching, settlement, payout creation, and query readback still happen through the normal running services

## Proof environment

- product services: local `api`, `account`, `matching`, `settlement`, `ledger`, `ws`
- deposit chain: in-process go-ethereum simulated backend created by `cmd/local-lifecycle`
- proof vault: ephemeral local contract that emits the canonical `Deposited(address,uint256)` event shape consumed by `internal/chain/service/listener.go`
- listener config:
  - `chain_id = 1337`
  - `chain_name = simulated`
  - `network_name = local-proof`
  - `confirmations = 0`
  - `start_block = vault_deploy_block + 1`

This restores listener truthfulness for local proof runs without claiming live BSC custody semantics from the default `.env.local`.

## Prerequisites

- local stack is up:

```bash
/Users/zhangza/code/funnyoption/scripts/dev-up.sh
```

- env is loaded for the command:

```bash
cd /Users/zhangza/code/funnyoption
set -a
source .env.local
set +a
```

## Run

```bash
go run ./cmd/local-lifecycle
```

Optional flags:

```bash
go run ./cmd/local-lifecycle --base-url http://127.0.0.1:8080 --deposit-amount 5000 --price 58 --quantity 40
```

## Dedicated admin service

The explicit market-bootstrap flow now lives in a dedicated local admin service instead of the transitional public-web `/admin` shell.
The single supported runtime for that service is the Next.js app under `/Users/zhangza/code/funnyoption/admin`.

If you already started local dev with `/Users/zhangza/code/funnyoption/scripts/dev-up.sh`, the admin service is already running on `http://127.0.0.1:3001`.

To run only the admin service in a second terminal:

```bash
cd /Users/zhangza/code/funnyoption/admin
npm run dev -- --hostname 127.0.0.1 --port 3001
```

By default it serves on `http://127.0.0.1:3001` and uses `FUNNYOPTION_API_BASE_URL` or `http://127.0.0.1:8080` to reach the API service.
Create, resolve, and first-liquidity now all move through the same wallet-gated operator lane inside that runtime.

## What the command does

- creates a fresh local market through `POST /api/v1/markets`
- creates two wallet-style sessions through `POST /api/v1/sessions`
- deploys an ephemeral proof vault on a local simulated chain
- submits one real wallet-signed deposit transaction for user `1001`
- boots a real `DepositListener` and waits for the resulting credit to appear through `GET /api/v1/deposits`
- issues one-shot first-liquidity for user `1002` through `POST /api/v1/admin/markets/:market_id/first-liquidity`, which returns the queued bootstrap `SELL` order id
- waits for that bootstrap `SELL` order to become visible, then places the crossing `BUY` order on the same book
- waits for a matched trade
- resolves the market to `YES`
- waits for payout creation and terminal market status
- prints a JSON summary with:
  - `proof_environment`
  - `deposit_id`
  - `deposit_tx_hash`
  - `deposit_log_index`
  - `deposit_block_number`
  - `deposit_vault_address`
  - `maker.first_liquidity_id`
  - `bootstrap_order_id`
  - `bootstrap_order_status`
  - buyer balance before deposit, after deposit, and after settlement

## Expected evidence

- a new market id
- one credited deposit id
- one deposit transaction hash
- one deposit log index / block number pair
- one bootstrap sell order id
- one buy order id
- one matched trade id
- market status `RESOLVED`
- resolved outcome `YES`
- buyer payout amount `> 0`

## Operator follow-up in UI

After the command completes:

- open `http://127.0.0.1:3001`
- connect an allowlisted operator wallet in the admin service
- confirm the new market appears in the dedicated admin-service market list
- confirm maker user `1002` now shows explicit paired `YES` / `NO` inventory rather than a hidden lifecycle-only seed
- confirm recent trades include the matched trade
- confirm the user snapshot cards or tables show:
  - buyer session
  - credited deposit
  - explicit first-liquidity issuance
  - order activity
  - payout row
