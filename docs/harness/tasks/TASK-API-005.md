# TASK-API-005

## Summary

Fix first-liquidity correctness so duplicate bootstrap requests are atomic/idempotent and one YES/NO pair consumes collateral in the same accounting units that settlement later pays out.

## Scope

- preserve the `TASK-API-004` policy that same-terms second bootstrap requests are rejected even with a fresh `requested_at`
- close the current side-effect bug where the second same-terms bootstrap call returns `409` but still issues an extra first-liquidity inventory pair and debits maker collateral
- fix first-liquidity collateral debit units so one issued YES/NO pair locks/debits `100 * quantity` accounting units, not raw `quantity`
- align API responses and tests with the final unit convention and duplicate-handling behavior
- keep the operator-auth boundary intact and do not reopen the old bare `user_id` order-write fallback

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-004.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-004.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-STAGING-001.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-STAGING-001.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md](/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-API-005.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-API-005.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-005.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-005.md)

## Owned files

- `internal/api/handler/**`
- `internal/api/dto/**`
- `admin/app/api/operator/markets/[marketId]/first-liquidity/route.ts`
- `admin/lib/**` if the bootstrap request contract needs a narrow adjustment
- `docs/harness/handshakes/HANDSHAKE-API-005.md`
- `docs/harness/worklogs/WORKLOG-API-005.md`

## Acceptance criteria

- repeating the same-terms bootstrap call with a fresh operator proof returns the intended duplicate rejection without changing maker inventory or maker USDT balance
- one first-liquidity issuance of `quantity=N` debits `100 * N` accounting units from maker collateral and response fields reflect that unit convention
- existing privileged operator auth checks and bare-`user_id` rejection remain covered
- tests cover both the duplicate side-effect regression and the collateral unit regression

## Validation

- targeted API/admin tests for the first-liquidity route and order handler
- one local or script-based lifecycle replay that checks maker balance/position before and after a duplicate bootstrap attempt

## Dependencies

- `TASK-API-004` sets the duplicate bootstrap policy baseline
- `TASK-STAGING-001` supplies concrete failing evidence and expected invariants

## Handoff

- write changed files, test commands, before/after balance+position evidence, and any residual behavior tradeoff to `WORKLOG-API-005.md`
