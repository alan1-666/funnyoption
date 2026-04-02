# HANDSHAKE-API-004

## Task

- [TASK-API-004.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-API-004.md)

## Thread owner

- implementation worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/architecture/direct-deposit-session-key.md`
- `WORKLOG-API-003.md`
- this handshake
- `WORKLOG-API-004.md`

## Files in scope

- `internal/api/**`
- `internal/api/handler/**`
- `internal/api/dto/**`
- `admin/app/api/operator/**`
- `admin/lib/**` if narrowly required
- related docs/runbooks

## Inputs from other threads

- `/api/v1/orders` no longer accepts bare unauthenticated `user_id` writes
- privileged bootstrap orders already reject replay of the exact same signed payload via the `TASK-API-003` deterministic bootstrap `order_id`
- the remaining explicit gap is semantic uniqueness across fresh proofs:
  - a second operator proof with a new `requested_at` can still authorize an otherwise-identical bootstrap sell order
  - the repo has not yet stated whether that should be allowed policy or rejected duplicate behavior

## Outputs back to commander

- changed files
- the chosen semantic-uniqueness policy for privileged bootstrap orders
- validation notes for first bootstrap success and second-bootstrap behavior with a fresh proof
- any remaining deeper order-ingress gaps

## Blockers

- do not widen into a general session nonce redesign for all user orders
- do not reintroduce the removed bare-`user_id` fallback
- do not break the first legitimate admin bootstrap flow while hardening second-attempt behavior

## Status

- completed

## Handoff notes

- semantic uniqueness now rejects a second otherwise-identical bootstrap sell order even with a freshly signed operator proof
- the authoritative bootstrap uniqueness handle is the deterministic `order_id` derived from `market_id`, `user_id`, `quantity`, `outcome`, and `price`; `requested_at` remains a proof-freshness field, not a distinct-action handle
- intentional same-terms second bootstrap actions are now out of contract until the repo introduces an explicit operator action handle for them
- validated with `go test ./internal/api/...`, focused bootstrap/session route tests, and `cd admin && npm run build`
