# WORKLOG-CHAIN-020

### 2026-04-05 22:46 CST

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `docs/harness/tasks/TASK-CHAIN-019.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-019.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-019.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
- changed:
  - created the inner `proofData` schema follow-up task, handshake, and
    worklog
- validated:
  - the next slice is now explicit enough to replace the current placeholder
    `proofData = abi.encode(proofTypeHash)` lane without reopening
    `VerifierContext`, `verifierGateHash`, the outer proof/public-signal
    envelope, or `shadow-batch-v1` public-input shape
- blockers:
  - none yet
- next:
  - launch one worker on `TASK-CHAIN-020`

### 2026-04-05 23:25 CST

- read:
  - `AGENTS.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-CHAIN-019.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-019.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-019.md`
  - `docs/harness/tasks/TASK-CHAIN-020.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-020.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-020.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
  - `internal/rollup/**`
  - `contracts/src/FunnyRollupVerifier.sol`
  - `contracts/src/FunnyRollupCore.sol`
  - `contracts/test/**`
- changed:
  - updated `internal/rollup/types.go`,
    `internal/rollup/verifier_contract.go`, and
    `internal/rollup/verifier_contract_test.go` so
    `VerifierArtifactBundle` now exports one explicit inner `proofData-v1`
    schema under the unchanged outer proof/public-signal envelope:
    - `proofDataSchemaHash = keccak256("funny-rollup-proof-data-v1")`
    - `proofData = abi.encode(proofDataSchemaHash, proofTypeHash,
      batchEncodingHash, authProofHash, verifierGateHash, proofBytes)`
    - current placeholder lane keeps
      `proofTypeHash = keccak256("funny-rollup-proof-placeholder-v1")`
      and `proofBytes = bytes("")`
    - decoded `proofData` fields, `proofData` bytes, and final
      `verifierProof` bytes are now all exported deterministically from Go
  - updated `contracts/src/FunnyRollupVerifier.sol` so the current verifier:
    - keeps the outer proof/public-signal envelope unchanged
    - decodes inner `proofData-v1`
    - constrains inner/outer/context parity for
      `batchEncodingHash` / `authProofHash` / `verifierGateHash`
    - only accepts the current empty-`proofBytes` placeholder lane instead of
      the old bare `abi.encode(proofTypeHash)` payload
  - updated `contracts/test/FunnyRollupCore.t.sol` to pin Go/Foundry parity
    for `proofDataSchemaHash`, inner `proofData`, final `verifierProof`, and
    to reject non-empty inner `proofBytes`
  - updated:
    - `docs/architecture/mode-b-zk-rollup.md`
    - `docs/sql/schema.md`
    - `docs/harness/handshakes/HANDSHAKE-CHAIN-020.md`
    - `docs/harness/worklogs/WORKLOG-CHAIN-020.md`
    to document the inner proof-data tranche truthfully
- validated:
  - `gofmt -w internal/rollup/types.go internal/rollup/verifier_contract.go internal/rollup/verifier_contract_test.go`
  - `forge fmt contracts/src/FunnyRollupVerifier.sol contracts/test/FunnyRollupCore.t.sol`
  - `GOCACHE=/tmp/funnyoption-gocache go test ./internal/rollup`
  - `forge test --match-path contracts/test/FunnyRollupCore.t.sol`
- blockers:
  - none in this tranche
- next:
  - hand back changed files, the inner `proofData` schema contract,
    validation commands, residual limitations, and the recommended next
    prover/verifier follow-up
