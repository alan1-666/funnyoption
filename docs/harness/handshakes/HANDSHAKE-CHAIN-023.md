# HANDSHAKE-CHAIN-023

## Task

- [TASK-CHAIN-023.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-023.md)

## Thread owner

- chain/rollup worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/tasks/TASK-CHAIN-022.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-022.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-022.md`
- `docs/architecture/mode-b-zk-rollup.md`
- `docs/sql/schema.md`
- `internal/rollup/**`
- `contracts/src/FunnyRollupGroth16Backend.sol`
- `contracts/src/FunnyRollupVerifier.sol`
- `contracts/test/**`
- `foundry.toml`
- this handshake
- `WORKLOG-CHAIN-023.md`

## Files in scope

- `internal/rollup/**`
- `contracts/src/**`
- `contracts/test/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-023.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-023.md`
- narrow repo-local proving helper paths only if required

## Inputs from other threads

- `TASK-CHAIN-022` landed:
  - one fixed-vk BN254 Groth16 backend contract
  - non-empty fixture `proofBytes`
  - fixed `proofTypeHash` dispatch
  - limb-splitting / proof-codec / verifier-verdict parity fixtures
- commander accepted `TASK-CHAIN-022` as completed
- commander wants the next slice to replace the one shared fixture proof with
  batch-specific proof artifacts generated from actual outer signals, without
  reopening the outer envelope or `proofData-v1`

## Outputs back to commander

- changed files
- batch-specific proof artifact pipeline
- validation commands
- residual limitations
- recommended next prover/verifier follow-up

## Handoff notes

- keep unchanged:
  - `VerifierContext`
  - `verifierGateHash`
  - outer proof/public-signal envelope
  - `proofData-v1`
  - fixed Groth16 `proofTypeHash`
  - `shadow-batch-v1` public-input shape
- replace only:
  - the one shared fixture proof artifact
  - with deterministic batch-specific proof artifacts derived from actual outer signals
- repo truth stays unchanged:
  - SQL/Kafka settlement is still production truth
  - direct-vault claim is still production truth
  - this tranche is not a claim that FunnyOption is already `Mode B`

## Blockers

- do not claim the product is already Mode B
- do not widen into production withdrawal-claim rewrite or forced-withdrawal runtime
- if contracts are touched, stay on Foundry only

## Status

- completed
