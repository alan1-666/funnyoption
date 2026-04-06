# TASK-CHAIN-033

## Summary

Land the first forced-withdrawal satisfaction runtime:
match pending forced-withdrawal requests against already-processed canonical
withdrawal claims, submit `satisfyForcedWithdrawal(...)` onchain with durable
local tx tracking, and expose one truthful read surface for forced-withdrawal
queue / freeze state without claiming full escape-hatch runtime.

## Scope

- build directly on `TASK-CHAIN-032`
- implement:
  - a chain-service satisfier that scans local forced-withdrawal mirrors plus
    accepted withdrawal claims and submits `FunnyRollupCore.satisfyForcedWithdrawal`
    when there is one unambiguous canonical match
  - durable local tx tracking for forced-withdrawal satisfaction attempts
  - one API read surface for forced-withdrawal requests and current freeze state
  - one local live validation that proves `REQUESTED -> SATISFIED` through the
    canonical claim lane
- do not implement:
  - full escape-hatch Merkle-claim runtime
  - a frozen production truth switch across every service
  - a new proof/public-input contract

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-032.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-032.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-032.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-032.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-032.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-032.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md](/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyVault.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyVault.sol)
- [/Users/zhangza/code/funnyoption/internal/chain/service/**](/Users/zhangza/code/funnyoption/internal/chain/service)
- [/Users/zhangza/code/funnyoption/internal/api/handler/**](/Users/zhangza/code/funnyoption/internal/api/handler)

## Owned files

- `internal/chain/model/**`
- `internal/chain/service/**`
- `internal/api/dto/**`
- `internal/api/handler/**`
- `internal/api/routes_reads.go`
- `migrations/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-033.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-033.md`

## Acceptance criteria

- pending forced-withdrawal requests can become `SATISFIED` without hand-edited
  SQL when one canonical claimed withdrawal matches
- satisfaction attempts have durable local tx tracking
- API can read:
  - forced-withdrawal requests
  - current freeze state
- docs stay explicit that full escape-hatch claims and frozen production-truth
  switching are still follow-up work

## Validation

- `go test ./internal/chain/service ./internal/api ./internal/api/handler ./internal/rollup`
- `forge test --offline --match-path contracts/test/FunnyRollupCore.t.sol`
- local dev:
  - `./scripts/dev-down.sh || true`
  - `./scripts/dev-up.sh`
  - create one canonical claimed withdrawal
  - create one matching `requestForcedWithdrawal(...)`
  - verify `REQUESTED -> SATISFIED`
- `git diff --check`

## Dependencies

- `TASK-CHAIN-032` completed

## Handoff

- return changed files, satisfaction runtime contract, read-surface evidence,
  live local validation, residual limitations, and the recommended next
  follow-up for frozen-mode truth switching or escape-hatch claim runtime
