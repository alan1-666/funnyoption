# TASK-ADMIN-004

## Summary

Harden shared core API operator endpoints so privileged market actions cannot bypass the dedicated admin service's wallet-gated operator checks.

## Scope

- define one backend authorization model for privileged operator actions that survives direct calls to the shared API
- protect at least these endpoints from unauthenticated direct use:
  - `POST /api/v1/markets`
  - `POST /api/v1/markets/:market_id/resolve`
  - `POST /api/v1/admin/markets/:market_id/first-liquidity`
- keep the dedicated Next admin service working through the protected path
- keep user order entry and normal public read surfaces out of scope unless a narrow dependency requires touching them
- record how operator identity is conveyed to the shared API and what, if anything, is persisted for audit/debugging

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-002.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-002.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-009.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-009.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-003.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-003.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-ADMIN-004.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-ADMIN-004.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-004.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-004.md)

## Owned files

- `internal/api/handler/**`
- `internal/api/dto/**`
- `admin/app/api/operator/**`
- `admin/lib/operator-*.ts`
- narrowly required config/docs files

## Acceptance criteria

- direct calls to privileged market/admin API endpoints are denied unless they satisfy the chosen operator-auth model
- the dedicated admin service still succeeds on create, resolve, and first-liquidity through the protected backend path
- docs/worklog state the chosen auth model, its trust boundary, and any remaining limitations
- at least one unauthorized direct-call proof and one authorized admin-service proof are recorded

## Validation

- focused Go tests for touched `internal/api/handler` paths
- `cd /Users/zhangza/code/funnyoption/admin && npm run build`
- one proof that the admin service can still create or bootstrap a market
- one proof that a direct unauthenticated or improperly authenticated API call is rejected

## Dependencies

- `TASK-ADMIN-003` output is the baseline

## Handoff

- return the chosen backend auth model
- list which privileged endpoints are now protected
- note any remaining deeper auth/audit gaps after backend hardening
