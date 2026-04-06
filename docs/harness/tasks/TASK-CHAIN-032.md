# TASK-CHAIN-032

## Summary

Land the first forced-withdrawal / freeze runtime foundation:
add L1 forced-withdrawal request state plus freeze gating to the rollup
contracts, mirror that state into local SQL, and keep the runtime explicit that
escape claims and full exit guarantees are still follow-up work.

## Scope

- build directly on `TASK-CHAIN-031`
- implement:
  - `FunnyRollupCore` forced-withdrawal request storage, deadline checks, and
    global freeze gating for normal batch advancement
  - `FunnyVault` processed-claim metadata readback needed for first-cut forced
    withdrawal satisfaction checks
  - one local SQL mirror for forced-withdrawal requests and freeze state
  - one chain-service poller that mirrors current `FunnyRollupCore`
    forced-withdrawal / freeze state into SQL
  - local-chain bootstrap wiring for the new core settings
- do not implement:
  - escape-hatch collateral proofs or payout runtime
  - a mutable backend write-truth switch
  - a full state-transition prover rewrite

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-031.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-031.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-031.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-031.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-031.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-031.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md](/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyVault.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyVault.sol)
- [/Users/zhangza/code/funnyoption/internal/chain/service/server.go](/Users/zhangza/code/funnyoption/internal/chain/service/server.go)
- [/Users/zhangza/code/funnyoption/internal/chain/service/sql_store.go](/Users/zhangza/code/funnyoption/internal/chain/service/sql_store.go)

## Owned files

- `contracts/src/**`
- `contracts/test/**`
- `internal/chain/model/**`
- `internal/chain/service/**`
- `migrations/**`
- `scripts/local-chain-up.sh`
- `scripts/dev-up.sh`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-032.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-032.md`

## Acceptance criteria

- `FunnyRollupCore` stores forced-withdrawal requests with:
  - wallet
  - recipient
  - amount
  - deadline
  - status
- `FunnyRollupCore` can freeze after a missed forced-withdrawal deadline and
  blocks normal batch advancement while frozen
- local SQL mirrors current forced-withdrawal requests and freeze state from
  the rollup core
- chain service can keep that mirror current without hand edits
- docs stay explicit that escape hatch claims are not implemented yet

## Validation

- `go test ./internal/chain/service ./internal/api ./internal/api/handler ./internal/rollup`
- `forge test --offline --match-path contracts/test/FunnyRollupCore.t.sol`
- local dev:
  - `./scripts/dev-up.sh`
  - manual `cast send` or equivalent local request/freeze exercise
- `git diff --check`

## Dependencies

- `TASK-CHAIN-031` completed

## Handoff

- return changed files, forced-withdrawal/freeze contract boundary, local mirror
  evidence, residual limitations, and the recommended next follow-up for
  satisfaction automation or escape-hatch claim runtime
