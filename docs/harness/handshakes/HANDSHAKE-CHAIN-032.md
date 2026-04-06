# HANDSHAKE-CHAIN-032

## Task

- [TASK-CHAIN-032.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-032.md)

## Thread owner

- commander+worker merged thread

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/COMMANDER.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/tasks/TASK-CHAIN-031.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-031.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-031.md`
- `docs/architecture/mode-b-zk-rollup.md`
- `docs/sql/schema.md`
- `contracts/src/FunnyRollupCore.sol`
- `contracts/src/FunnyVault.sol`
- `internal/chain/service/**`
- this handshake
- `WORKLOG-CHAIN-032.md`

## Files in scope

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

## Inputs from other threads

- `TASK-CHAIN-031` proved one verifier-eligible local full-flow through
  accepted read truth
- the next hard gap is still exit guarantees:
  - forced-withdrawal request state
  - freeze gating
  - later escape hatch

## Outputs back to commander

- changed files
- forced-withdrawal / freeze contract boundary
- local mirror evidence
- validation commands
- residual limitations
- recommended next satisfaction / escape-hatch follow-up

## Handoff notes

- keep unchanged:
  - current accepted submission lane
  - current verifier/public-input contract
  - mutable backend write truth
- add only:
  - forced-withdrawal request storage and freeze gating
  - local SQL mirrors
  - one chain-service sync path
  - fresh-start cleanup for the new local mirror tables
- do not widen into:
  - escape-hatch claim proofs
  - production truth switch
  - new proof/public-signal contract work

## Blockers

- do not claim full exit guarantee until escape-hatch claims exist
- keep first-cut semantics honest if satisfaction is still narrower than the
  final architecture

## Status

- completed
