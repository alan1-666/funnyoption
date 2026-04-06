# HANDSHAKE-CHAIN-034

## Task

- [TASK-CHAIN-034.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-034.md)

## Thread owner

- commander+worker merged thread

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/COMMANDER.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/tasks/TASK-CHAIN-033.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-033.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-033.md`
- `docs/architecture/mode-b-zk-rollup.md`
- `docs/sql/schema.md`
- `internal/api/handler/order_handler.go`
- `internal/matching/service/sql_store.go`
- `internal/matching/service/consumer.go`
- this handshake
- `WORKLOG-CHAIN-034.md`

## Files in scope

- `internal/api/handler/**`
- `internal/matching/service/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-034.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-034.md`

## Inputs from other threads

- `TASK-CHAIN-033` made forced-withdrawal satisfaction truthful, but frozen mode
  still needed one honest trading/write boundary

## Outputs back to commander

- changed files
- frozen-mode runtime contract
- validation commands
- residual limitations
- recommended next frozen-mode / escape-hatch follow-up

## Handoff notes

- keep unchanged:
  - current verifier/proof lane
  - current accepted submission lane
  - current withdrawal claim lane
- add only:
  - API-side frozen order rejection
  - matching-side frozen tradability / restore guard
- do not widen into:
  - escape-hatch proof claims
  - a global mutable-backend truth switch
  - new contract/prover work

## Blockers

- frozen mode must stop saying trading is possible even if market lifecycle is
  otherwise open

## Status

- completed
