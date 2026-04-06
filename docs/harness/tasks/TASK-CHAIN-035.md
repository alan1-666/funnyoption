# TASK-CHAIN-035

## Summary

Widen frozen-mode runtime truth from the first trading gate into a broader
mutable-backend guard:
once `rollup_freeze_state.frozen = true`, stop privileged API writes,
oracle/settlement background mutation, account-service mutable balance
processing, and rollup submitter broadcasting from continuing to advance the
legacy SQL/Kafka truth.

## Scope

- build directly on `TASK-CHAIN-034`
- implement:
  - one shared API-side frozen gate for privileged mutable endpoints
  - one oracle-worker frozen skip
  - one settlement-processor frozen skip
  - one account-service frozen skip
  - one submitter-side frozen idle state so frozen mode stops before onchain
    revert attempts
  - docs that call this a broader frozen mutable-truth guard, not full
    escape-hatch runtime or full production-truth switching
- do not implement:
  - Merkle-proof escape claims
  - a full mutable-backend truth switch for every service/process
  - a new prover / verifier contract lane

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-034.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-034.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-034.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-034.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-034.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-034.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md](/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/internal/api/handler/order_handler.go](/Users/zhangza/code/funnyoption/internal/api/handler/order_handler.go)
- [/Users/zhangza/code/funnyoption/internal/account/service/event_processor.go](/Users/zhangza/code/funnyoption/internal/account/service/event_processor.go)
- [/Users/zhangza/code/funnyoption/internal/oracle/service/worker.go](/Users/zhangza/code/funnyoption/internal/oracle/service/worker.go)
- [/Users/zhangza/code/funnyoption/internal/settlement/service/processor.go](/Users/zhangza/code/funnyoption/internal/settlement/service/processor.go)
- [/Users/zhangza/code/funnyoption/internal/chain/service/rollup_submitter.go](/Users/zhangza/code/funnyoption/internal/chain/service/rollup_submitter.go)

## Owned files

- `internal/api/handler/**`
- `internal/account/service/**`
- `internal/oracle/service/**`
- `internal/settlement/service/**`
- `internal/chain/service/rollup_submitter.go`
- `internal/rollup/store.go`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-035.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-035.md`

## Acceptance criteria

- privileged API writes reject while frozen:
  - create market
  - first liquidity
  - resolve market
  - claim payout
- oracle worker writes no new resolution state and publishes no market events
  while frozen
- settlement processor writes no new position/settlement state while frozen
- account service applies no new order/trade/settlement balance mutations while
  frozen
- rollup submitter returns a stable frozen idle state instead of continuing to
  broadcast batch transactions that will revert onchain
- docs stay explicit that this is still not full escape-hatch runtime or full
  production-truth switching

## Validation

- `go test ./internal/account/service ./internal/api ./internal/api/handler ./internal/matching/service ./internal/oracle/service ./internal/settlement/service ./internal/chain/service ./internal/rollup ./cmd/rollup`
- `forge test --offline --match-path contracts/test/FunnyRollupCore.t.sol`
- `git diff --check`

## Dependencies

- `TASK-CHAIN-034` completed

## Handoff

- return changed files, widened frozen runtime contract, validation commands,
  residual limitations, and the recommended next follow-up for escape-hatch
  claims or fuller production-truth switching
