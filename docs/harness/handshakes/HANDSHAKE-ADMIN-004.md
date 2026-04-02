# HANDSHAKE-ADMIN-004

## Task

- [TASK-ADMIN-004.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-ADMIN-004.md)

## Thread owner

- implementation worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/architecture/direct-deposit-session-key.md`
- `WORKLOG-ADMIN-002.md`
- `WORKLOG-OFFCHAIN-009.md`
- `WORKLOG-ADMIN-003.md`
- this handshake
- `WORKLOG-ADMIN-004.md`

## Files in scope

- `internal/api/handler/**`
- `internal/api/dto/**`
- `admin/app/api/operator/**`
- `admin/lib/operator-*.ts`
- narrowly required config/docs files

## Inputs from other threads

- the dedicated Next admin service is now the single supported operator runtime
- create, resolve, and first-liquidity are wallet-gated at the admin-service boundary today
- commander review confirmed that the shared backend endpoints still trust direct callers outside that boundary

## Outputs back to commander

- changed files
- the chosen backend auth model
- protected endpoint list
- authorized and unauthorized validation notes
- any remaining deeper auth/audit gaps

## Blockers

- do not widen into general user session auth or unrelated order-entry auth in this task

## Status

- completed

## Handoff notes

- chosen backend auth model:
  - shared operator signature verification at the core API boundary
  - the dedicated Next admin service still verifies the wallet first for UX and early denial, then forwards the signed proof in a shared JSON shape that the Go API re-verifies independently
  - privileged proof fields sent to the core API:
    - `operator.wallet_address`
    - `operator.requested_at`
    - `operator.signature`
- protected endpoint list:
  - `POST /api/v1/markets`
  - `POST /api/v1/markets/:market_id/resolve`
  - `POST /api/v1/admin/markets/:market_id/first-liquidity`
- implementation notes:
  - `CreateMarket` now ignores caller-supplied `created_by` and stamps the configured `FUNNYOPTION_DEFAULT_OPERATOR_USER_ID` after operator proof verification
  - create-market metadata is now enriched at the shared API boundary with verified `operatorWalletAddress`, `operatorRequestedAt`, and `operatorService=shared-api`
  - resolve and first-liquidity verify the operator proof in-request but do not persist a new audit row in this task
- validation notes:
  - focused tests: `go test ./internal/api/handler ./internal/api/dto`
  - admin build: `cd /Users/zhangza/code/funnyoption/admin && npm run build`
  - authorized runtime proof:
    - temporary hardened API on `http://127.0.0.1:8083` with allowlist `0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266`
    - temporary admin service on `http://127.0.0.1:3012`
    - signed `POST /api/operator/markets` through the admin service returned `201`
    - created market `1775113807022`
    - response metadata showed `operatorWalletAddress=0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266`, `operatorRequestedAt=1775113769163`, `operatorService=shared-api`
  - unauthorized runtime proof:
    - direct `POST http://127.0.0.1:8083/api/v1/markets` without operator proof returned `401 {"error":"operator proof is required for privileged actions"}`
- remaining deeper auth/audit gaps:
  - this task does not harden general user order ingress
  - first-liquidity bootstrap still relies on the existing `/api/v1/orders` path for the first sell-order submit after inventory issuance
