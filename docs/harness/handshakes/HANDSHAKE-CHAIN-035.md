# HANDSHAKE-CHAIN-035

## Task

- [TASK-CHAIN-035.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-035.md)

## Thread owner

- commander+worker merged thread

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/COMMANDER.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/tasks/TASK-CHAIN-034.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-034.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-034.md`
- `docs/architecture/mode-b-zk-rollup.md`
- `docs/sql/schema.md`
- `internal/api/handler/order_handler.go`
- `internal/account/service/event_processor.go`
- `internal/oracle/service/worker.go`
- `internal/settlement/service/processor.go`
- `internal/chain/service/rollup_submitter.go`
- this handshake
- `WORKLOG-CHAIN-035.md`

## Files in scope

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

## Inputs from other threads

- `TASK-CHAIN-034` already made trading/runtime stop calling frozen markets
  tradable, but broader mutable-backend truth still needed one honest frozen
  guard

## Outputs back to commander

- changed files
- widened frozen runtime contract
- validation commands
- residual limitations
- recommended next frozen-mode / escape-hatch follow-up

## Handoff notes

- keep unchanged:
  - current verifier/proof lane
  - current accepted submission lane
  - current withdrawal claim lane
  - current forced-withdrawal satisfaction lane
- add only:
  - privileged API frozen write rejection
  - oracle / settlement / account frozen write skips
  - submitter frozen idle state
- do not widen into:
  - full escape-hatch proof claims
  - full production-truth switching
  - new contract/prover work

## Blockers

- frozen mode must stop the repo from continuing to advance legacy mutable
  backend truth through privileged writes or background processors

## Status

- completed
