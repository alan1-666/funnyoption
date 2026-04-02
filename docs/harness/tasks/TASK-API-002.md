# TASK-API-002

## Summary

Remove the transitional bare-`user_id` trade-write path by migrating admin bootstrap order placement onto an authenticated lane and then enforcing session-or-privileged auth on `/api/v1/orders`.

## Scope

- identify the current bootstrap caller that still posts directly to `/api/v1/orders` without a session payload
- choose one authenticated replacement path for that bootstrap order submit:
  - a privileged operator-authenticated order lane, or
  - a real session-backed order submit for the bootstrap actor
- once the bootstrap caller is migrated, remove the shared API's transitional bare-`user_id` fallback on `POST /api/v1/orders`
- keep normal end-user session-backed order flow working
- keep explicit first-liquidity bootstrap working through the dedicated admin service
- do not widen into a general redesign of order semantics, matching, or settlement

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-004.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-004.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-001.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-001.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-API-002.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-API-002.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-002.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-002.md)

## Owned files

- `internal/api/**`
- `internal/api/handler/**`
- `internal/api/dto/**`
- `admin/app/api/operator/**`
- `admin/lib/**` if narrowly required
- related docs/runbooks

## Acceptance criteria

- `/api/v1/orders` no longer accepts direct bare-`user_id` bootstrap-style writes without the intended authenticated envelope
- the dedicated admin bootstrap path still succeeds through the chosen authenticated order lane
- normal session-backed order submission still succeeds
- at least one unauthorized direct `POST /api/v1/orders` proof shows rejection when session or the chosen privileged proof is absent
- docs/worklog explain the chosen replacement path and any remaining order-ingress limitations

## Validation

- `go test ./internal/api/...`
- `cd /Users/zhangza/code/funnyoption/admin && npm run build`
- one proof that admin bootstrap still places the first sell order successfully
- one proof that bare `POST /api/v1/orders` without the required auth envelope is rejected

## Dependencies

- `TASK-API-001` output is the baseline

## Handoff

- return the chosen authenticated bootstrap-order path
- state how `/api/v1/orders` is now protected
- note any remaining deeper order-ingress gaps after the fallback is removed
