# TASK-CHAIN-007

## Summary

Add an explicit dispatch / retry contract for oracle observations so a market
that is successfully written as `OBSERVED` but fails to publish the resolved
event can be retried safely without duplicate settlement side effects.

## Scope

- close the residual reliability gap left by `TASK-CHAIN-006`
- handle the narrow failure mode:
  - oracle worker writes `market_resolutions` as `OBSERVED`
  - publish of the corresponding resolved `market.event` fails
  - later poll or restart should be able to dispatch safely
- choose the narrowest safe implementation:
  - explicit dispatch marker in the current store
  - or equivalent retry-safe contract
- keep the slice narrow:
  - no multi-provider arbitration
  - no new on-chain helper
  - no second Solidity toolchain
  - do not regress the existing duplicate-emit guard

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/oracle-settled-crypto-markets.md](/Users/zhangza/code/funnyoption/docs/architecture/oracle-settled-crypto-markets.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-006.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-006.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-006.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-006.md)
- [/Users/zhangza/code/funnyoption/backend/internal/oracle/service/worker.go](/Users/zhangza/code/funnyoption/backend/internal/oracle/service/worker.go)
- [/Users/zhangza/code/funnyoption/backend/internal/oracle/service/sql_store.go](/Users/zhangza/code/funnyoption/backend/internal/oracle/service/sql_store.go)
- [/Users/zhangza/code/funnyoption/backend/internal/settlement/service/processor.go](/Users/zhangza/code/funnyoption/backend/internal/settlement/service/processor.go)
- [/Users/zhangza/code/funnyoption/backend/internal/account/service/event_processor.go](/Users/zhangza/code/funnyoption/backend/internal/account/service/event_processor.go)

## Owned files

- `internal/oracle/service/**`
- `cmd/oracle/**` only if retry wiring needs a narrow entrypoint update
- `internal/settlement/**` only if a tiny contract note or idempotency guard is
  strictly required
- `docs/architecture/oracle-settled-crypto-markets.md` only if runtime truth
  would otherwise be unclear
- `docs/sql/schema.md` only if the chosen retry contract introduces a narrow
  schema field or table
- `migrations/**` only if required by the chosen marker
- `docs/harness/handshakes/HANDSHAKE-CHAIN-007.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-007.md`

## Acceptance criteria

- if publish fails after writing `OBSERVED`, the system has one explicit,
  restart-safe way to retry dispatch
- repeated poll / retry does not duplicate settlement payouts,
  `settled_quantity`, or account debit-credit side effects
- the current duplicate-emit guard remains intact for already-dispatched
  observations
- validation includes:
  - targeted Go tests for the retry contract
  - one simulated publish-failure proof
  - one restart or repeated-poll proof

## Validation

- targeted Go tests for oracle worker / store retry behavior
- one simulated publish-failure proof
- one repeated-poll or restart proof

## Dependencies

- `TASK-CHAIN-006` runtime baseline is complete

## Handoff

- return changed files, validation commands, and before/after dispatch behavior
- call out any remaining audit-depth tradeoff such as latest-row vs attempt log
