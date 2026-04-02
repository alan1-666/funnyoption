# TASK-OFFCHAIN-009

## Summary

Replace the hidden inventory seed used by the lifecycle proof with an explicit first-liquidity path so a freshly created admin market can become tradable without out-of-band bootstrap state, and land that operator flow in the dedicated admin service rather than the transitional public-web admin shell.

## Scope

- define one explicit operator-controlled first-liquidity mechanism for fresh markets
- expose that mechanism truthfully in the dedicated admin-service market flow instead of relying on hidden seed logic in the lifecycle runner
- ensure balances, positions, orders, and settlement reads remain truthful after the bootstrap path is used
- update the lifecycle proof and docs to use the explicit first-liquidity path

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/operations/local-offchain-lifecycle.md](/Users/zhangza/code/funnyoption/docs/operations/local-offchain-lifecycle.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-001.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-001.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-ADMIN-002.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-ADMIN-002.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-009.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-009.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-009.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-009.md)

## Owned files

- `admin/**` as the preferred admin-service root
- `web/app/admin/**` only if a migration shim or redirect is required
- `web/components/market-studio.tsx`
- `web/components/admin-market-ops.tsx`
- `internal/api/dto/order.go`
- `internal/api/handler/order_handler.go`
- `internal/account/service/**`
- `internal/settlement/service/**`
- `cmd/local-lifecycle/**`
- related lifecycle docs

## Acceptance criteria

- a fresh admin-created market can become tradable through an explicit first-liquidity path documented in the dedicated admin-service flow
- the lifecycle proof no longer depends on hidden seed logic
- the admin UI makes the bootstrap state explicit
- balances, positions, orders, trades, and settlement reads stay truthful for the bootstrap path

## Validation

- relevant Go tests for any touched account/settlement/api paths
- build or start proof for the admin service runtime
- one lifecycle proof against a fresh market showing tradeability without out-of-band hidden seeding

## Dependencies

- `TASK-ADMIN-002` output is the baseline

## Handoff

- return the explicit first-liquidity design actually implemented in code
- note any remaining market-onboarding blockers after hidden bootstrap state is removed
