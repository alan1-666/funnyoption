# TASK-API-003

## Summary

Add replay/idempotency protection to the privileged bootstrap-order lane so operator-signed bootstrap sell orders cannot be replayed within the current proof window.

## Scope

- define one explicit anti-replay model for privileged bootstrap orders submitted through `POST /api/v1/orders`
- ensure the dedicated admin bootstrap flow includes the fields required by that anti-replay model
- reject duplicate or replayed privileged bootstrap order submissions after the first accepted attempt
- keep normal end-user session-backed order flow unchanged
- keep the dedicated admin bootstrap flow working
- do not widen into a full redesign of user-session order nonces or unrelated matching semantics

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-002.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-002.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-API-003.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-API-003.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-003.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-003.md)

## Owned files

- `internal/api/**`
- `internal/api/handler/**`
- `internal/api/dto/**`
- `admin/app/api/operator/**`
- `admin/lib/**` if narrowly required
- related docs/runbooks

## Acceptance criteria

- the privileged bootstrap-order lane has explicit replay/idempotency protection beyond the current signature time window alone
- duplicate privileged bootstrap order submissions are rejected or safely collapsed after the first accepted attempt
- the dedicated admin bootstrap flow still succeeds on the first legitimate attempt
- normal session-backed order flow still succeeds
- tests or proof cover:
  - first privileged bootstrap order succeeds
  - replayed privileged bootstrap order is rejected or treated idempotently

## Validation

- `go test ./internal/api/...`
- `cd /Users/zhangza/code/funnyoption/admin && npm run build`
- one proof for first privileged bootstrap order success
- one proof for privileged bootstrap replay rejection or idempotent collapse

## Dependencies

- `TASK-API-002` output is the baseline

## Handoff

- return the chosen anti-replay/idempotency model
- state what field or key is now authoritative for bootstrap-order uniqueness
- note any remaining deeper order-ingress hardening gaps after this change
