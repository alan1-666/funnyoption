# HANDSHAKE-API-001

## Task

- [TASK-API-001.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-API-001.md)

## Thread owner

- implementation worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/architecture/direct-deposit-session-key.md`
- `docs/topics/kafka-topics.md`
- `WORKLOG-ADMIN-004.md`
- this handshake
- `WORKLOG-API-001.md`

## Files in scope

- `internal/api/server.go`
- `internal/api/handler/**`
- `internal/api/**` newly introduced middleware/router files if needed
- `internal/shared/auth/**` if narrowly required
- related API docs and runbooks

## Inputs from other threads

- privileged operator/admin auth is being hardened deeper into the shared API by `TASK-ADMIN-004`
- current API structure still centralizes many unrelated routes in one mixed registration function
- current middleware stack is minimal and does not yet include explicit rate limiting

## Outputs back to commander

- changed files
- router/module split summary
- middleware stack summary
- rate-limited endpoint list
- auth-boundary validation notes

## Blockers

- do not widen into unrelated service rewrites outside `internal/api`
- do not duplicate `TASK-ADMIN-004`; build on its chosen backend auth boundary
- full session-only enforcement on `POST /api/v1/orders` still depends on migrating the dedicated admin service's first-liquidity bootstrap caller, which is outside this task's owned files; this task keeps that transitional lane explicit, isolated, and rate-limited instead of silently preserving it in one mixed router

## Status

- completed

## Handoff notes

- router split landed in `internal/api` with separate registration for:
  - health/meta
  - public reads
  - session routes
  - trade writes
  - claim writes
  - privileged operator/admin writes
- middleware stack is now explicit:
  - global: recovery -> request logging -> CORS
  - sensitive writes: route-group rate limiting
  - privileged writes: operator-proof envelope check before handler verification
  - trade writes: explicit session-or-transitional-bootstrap boundary instead of one mixed route file
- rate-limited sensitive paths:
  - `POST /api/v1/sessions`
  - `POST /api/v1/sessions/:session_id/revoke`
  - `POST /api/v1/orders`
  - `POST /api/v1/payouts/:event_id/claim`
  - `POST /api/v1/markets`
  - `POST /api/v1/markets/:market_id/resolve`
  - `POST /api/v1/admin/markets/:market_id/first-liquidity`
- validation now includes router-level tests for:
  - public read success
  - session-backed order write success
  - operator route unauthorized without proof
  - repeated session creation denied by rate limiting
