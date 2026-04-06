# TASK-CHAIN-030

## Summary

Extend the accepted rollup lane from one truthful slow-withdraw claim path into
accepted balances / positions / payout read truth:
materialize accepted replay snapshots into durable accepted-state tables and
switch read surfaces to those accepted mirrors once accepted onchain state is
visible.

## Scope

- build directly on `TASK-CHAIN-029`
- keep the current repo truth explicit:
  - SQL/Kafka settlement still drives mutable backend writes
  - direct-vault deposits still exist as the ingestion source
  - do not claim FunnyOption is already full `Mode B`
- implement:
  - one deterministic accepted replay snapshot built from ordered accepted
    batches
  - durable accepted balance / position / settlement-payout tables
  - one idempotent rebuild path that refreshes those tables after accepted
    submissions
  - one read-surface switch for `/balances`, `/positions`, and `/payouts` that
    prefers accepted truth when accepted batches exist
- do not implement:
  - forced-withdrawal / freeze / escape hatch runtime
  - a new proof/public-signal contract
  - a second Solidity toolchain
  - a full mutable backend truth switch for matching / account / settlement

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-029.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-029.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-029.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-029.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-029.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-029.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md](/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/internal/rollup](/Users/zhangza/code/funnyoption/internal/rollup)
- [/Users/zhangza/code/funnyoption/internal/api/handler/sql_store.go](/Users/zhangza/code/funnyoption/internal/api/handler/sql_store.go)
- [/Users/zhangza/code/funnyoption/internal/api/dto/order.go](/Users/zhangza/code/funnyoption/internal/api/dto/order.go)

## Owned files

- `internal/rollup/**`
- `internal/api/handler/**`
- `internal/api/dto/**`
- `migrations/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-030.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-030.md`

## Acceptance criteria

- accepted batches can be replayed into one deterministic snapshot of:
  - balances
  - positions
  - settlement payouts
- durable accepted-state tables exist for those snapshots
- accepted-submission materialization refreshes those tables idempotently
- `/api/v1/balances`, `/api/v1/positions`, and `/api/v1/payouts` return
  accepted truth once accepted batches exist
- docs stay explicit that mutable backend write truth still has not fully
  switched

## Validation

- targeted Go tests for accepted replay snapshot derivation
- targeted Go tests for accepted read-surface queries
- `go test ./internal/rollup ./internal/api/handler`
- existing `go test ./internal/chain/service ./internal/api`
- `forge test --offline --match-path contracts/test/FunnyRollupCore.t.sol`
- local API verification against a real accepted lane
- `git diff --check`

## Dependencies

- `TASK-CHAIN-029` completed

## Handoff

- return changed files, accepted-state materialization behavior, validation
  commands, local accepted read-surface evidence, residual limitations, and the
  recommended next follow-up for forced withdrawal / fuller truth switch
