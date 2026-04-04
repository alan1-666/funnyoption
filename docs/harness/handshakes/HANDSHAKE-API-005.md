# HANDSHAKE-API-005

## Task

- [TASK-API-005.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-API-005.md)

## Thread owner

- API/admin worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `WORKLOG-API-004.md`
- `WORKLOG-STAGING-001.md`
- `docs/architecture/order-flow.md`
- this handshake
- `WORKLOG-API-005.md`

## Files in scope

- `internal/api/handler/**`
- `internal/api/dto/**`
- `admin/app/api/operator/markets/[marketId]/first-liquidity/route.ts`
- narrow `admin/lib/**` updates if needed
- `docs/harness/handshakes/HANDSHAKE-API-005.md`
- `docs/harness/worklogs/WORKLOG-API-005.md`

## Inputs from other threads

- `TASK-API-004` policy baseline:
  - same-terms second bootstrap requests must still be rejected even with a fresh `requested_at`
- staging evidence:
  - second same-terms bootstrap returned `409` but maker YES position changed `1 -> 2`
  - maker USDT available changed `1106 -> 1105` for the first issuance and `1105 -> 1104` after the duplicate rejection
  - one YES/NO pair should consume `100` accounting units per quantity because winning settlement pays `100`

## Outputs back to commander

- patch summary and changed files
- tests and local replay commands
- before/after evidence for maker position, maker USDT balance, and duplicate response shape
- any compatibility impact on admin bootstrap callers
- implementation note:
  - `POST /api/v1/admin/markets/:market_id/first-liquidity` will issue paired inventory and queue the bootstrap sell order inside one API handler under the bootstrap semantic replay lock
  - the admin route will stop sending a second `/api/v1/orders` request for the same bootstrap action and will forward the order fields returned by the first-liquidity API

## Blockers

- do not weaken operator-proof verification or reintroduce bare `user_id` order writes
- do not change portfolio or chain-listener files owned by `TASK-OFFCHAIN-011` / `TASK-CHAIN-004`
- no remaining code blocker in this task

## Handoff notes

- same-terms duplicate first-liquidity requests now fail inside the core first-liquidity handler before maker collateral/inventory mutation
- successful first-liquidity requests now debit `100 * quantity` collateral units and return `order_id` / `order_status` for the bootstrap sell order
- `admin/app/api/operator/markets/[marketId]/first-liquidity/route.ts` now relies on the one-shot core first-liquidity API and no longer submits a second `/api/v1/orders` call
- full local lifecycle replay was not executed because `./scripts/dev-status.sh` shows the dev stack is down; regression evidence and compatibility notes are captured in `WORKLOG-API-005.md`
- deploy closeout:
  - committed on `main` as `125f9cd4af344680e78529c5a98358b39427e703` (`Deploy reviewed API-005 and OFFCHAIN-011 fixset`)
  - GitHub Actions `staging-deploy` run `23977457019` completed `success`; both `validate` and `deploy-staging` jobs passed
  - staging server checkout `/opt/funnyoption-staging` now reports `HEAD=125f9cd`, `git status --short` clean, and `GET https://funnyoption.xyz/healthz` returns `{"env":"staging","service":"api","status":"ok"}`

## Status

- completed
