# HANDSHAKE-CHAIN-033

## Task

- [TASK-CHAIN-033.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-033.md)

## Thread owner

- commander+worker merged thread

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/COMMANDER.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/tasks/TASK-CHAIN-032.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-032.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-032.md`
- `docs/architecture/mode-b-zk-rollup.md`
- `docs/sql/schema.md`
- `contracts/src/FunnyRollupCore.sol`
- `contracts/src/FunnyVault.sol`
- `internal/chain/service/**`
- `internal/api/handler/**`
- this handshake
- `WORKLOG-CHAIN-033.md`

## Files in scope

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

## Inputs from other threads

- `TASK-CHAIN-032` landed the first onchain forced-withdrawal queue / freeze
  foundation plus local SQL mirrors
- the next honest gap is runtime satisfaction and read visibility, not yet full
  escape-hatch claims

## Outputs back to commander

- changed files
- satisfaction runtime contract
- API read-surface evidence
- validation commands
- residual limitations
- recommended next frozen-mode / escape-hatch follow-up

## Handoff notes

- keep unchanged:
  - outer verifier/proof contract
  - current accepted submission lane
  - current mutable backend write truth
- add only:
  - forced-withdrawal satisfaction runtime
  - durable tx tracking for that runtime
  - truthful read surfaces for forced-withdraw queue / freeze state
- do not widen into:
  - full escape-hatch proof claims
  - global frozen production-truth switching across every service
  - new proving-system contract work

## Blockers

- only auto-satisfy when the canonical claim match is unambiguous
- keep ambiguous / unmatched requests explicit instead of guessing

## Status

- completed
