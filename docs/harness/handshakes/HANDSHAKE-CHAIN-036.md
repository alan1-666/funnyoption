# HANDSHAKE-CHAIN-036

## Task

- [TASK-CHAIN-036.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-036.md)

## Thread owner

- commander+worker merged thread

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/COMMANDER.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/tasks/TASK-CHAIN-035.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-035.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-035.md`
- `docs/architecture/mode-b-zk-rollup.md`
- `docs/sql/schema.md`
- `internal/rollup/**`
- `internal/api/handler/**`
- `internal/chain/service/**`
- `contracts/src/FunnyRollupCore.sol`
- `contracts/src/FunnyVault.sol`
- `contracts/src/FunnyRollupVerifier.sol`
- this handshake
- `WORKLOG-CHAIN-036.md`

## Files in scope

- `internal/rollup/**`
- `internal/api/dto/**`
- `internal/api/handler/**`
- `internal/api/routes_reads.go`
- `internal/chain/model/**`
- `internal/chain/service/**`
- `contracts/src/**`
- `contracts/test/**`
- `cmd/rollup/**`
- `migrations/**`
- `scripts/local-chain-up.sh`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-036.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-036.md`

## Inputs from other threads

- `TASK-CHAIN-035` widened the frozen stop signal across mutable backend lanes,
  which means escape-hatch claims can now become the next truthful exit path
  without reopening those guards.
- accepted read truth already exists for balances / positions / payouts /
  withdrawals, but escape claims and the prover lane still lag behind.

## Outputs back to commander

- changed files
- escape-claim contract/runtime
- accepted/frozen truth switch details
- proving-lane changes
- validation commands
- residual limitations

## Handoff notes

- keep stable where possible:
  - outer proof/public-signal envelope
  - current trading-key auth witness boundary
  - Foundry-only contract toolchain
- acceptable to widen if required:
  - accepted-batch stored fields
  - proofTypeHash dispatch
  - proofData payload internals
  - read surfaces for accepted/frozen truth
- do not leave escape claims as docs-only

## Blockers

- no active blocker
- residuals remain:
  - unresolved-open-position emergency handling at freeze is still narrow
  - prover/backend is still repo-local first cut, not a production proving fleet
  - the repo is closer to `Mode B`, but not every live truth boundary is fully replaced by accepted roots

## Status

- completed
