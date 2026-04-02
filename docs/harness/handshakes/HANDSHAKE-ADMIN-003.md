# HANDSHAKE-ADMIN-003

## Task

- [TASK-ADMIN-003.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-ADMIN-003.md)

## Thread owner

- implementation worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/architecture/direct-deposit-session-key.md`
- `docs/operations/local-offchain-lifecycle.md`
- `WORKLOG-CHAIN-002.md`
- `WORKLOG-ADMIN-002.md`
- `WORKLOG-OFFCHAIN-009.md`
- this handshake
- `WORKLOG-ADMIN-003.md`

## Files in scope

- `admin/**`
- `web/app/admin/**`
- `scripts/dev-up.sh`
- `docs/operations/local-offchain-lifecycle.md`
- related admin runtime docs

## Inputs from other threads

- `TASK-ADMIN-002` produced a wallet-gated Next-based admin runtime for create/resolve
- `TASK-OFFCHAIN-009` produced an explicit first-liquidity path, but the proof currently relies on a second Go/template admin runtime shape
- commander review says the product should not keep both runtime shapes as first-class operator entrypoints

## Outputs back to commander

- changed files
- the single supported admin runtime shape
- authorized and unauthorized validation notes for first-liquidity/bootstrap
- any remaining deeper backend-auth gaps

## Blockers

- do not widen into schema-drift cleanup or generalized core-API auth unless strictly required

## Status

- completed

## Handoff notes

- single supported runtime shape:
  - `admin/` Next.js service started by `scripts/dev-up.sh` or `cd /Users/zhangza/code/funnyoption/admin && npm run dev -- --hostname 127.0.0.1 --port 3001`
- deprecated runtime:
  - the old Go/template admin runtime is no longer a supported operator surface
  - `admin/main.go` is now only a deprecation shim and `admin/templates/index.html` was removed
- first-liquidity/bootstrap convergence:
  - added a wallet-gated admin route at `admin/app/api/operator/markets/[marketId]/first-liquidity/route.ts`
  - the Next admin UI now issues paired inventory and queues the first sell order from the same signed allowlist lane used by create and resolve
- validation notes:
  - authorized wallet `0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266` created market `1775106639302` through the admin runtime and then issued first-liquidity `liq_1775106644428_f467fd12c1d0` plus sell order `ord_1775106644668_7aaafd4f5849`
  - core reads confirmed user `1002` received paired `YES` / `NO` inventory rows and one resting `SELL YES` order on that market
  - unauthorized wallet `0x14791697260e4c9a71f18484c9f997b308e59325` was denied on the same first-liquidity route with `403 wallet is not authorized for operator actions`
- remaining deeper backend-auth gaps:
  - the wallet gate still lives at the admin-service boundary
  - direct callers can still hit shared backend endpoints such as `POST /api/v1/markets`, `POST /api/v1/markets/:market_id/resolve`, `POST /api/v1/admin/markets/:market_id/first-liquidity`, and `POST /api/v1/orders` without the admin-service signature check
