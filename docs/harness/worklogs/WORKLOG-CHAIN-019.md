# WORKLOG-CHAIN-019

### 2026-04-05 22:16 CST

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `docs/harness/tasks/TASK-CHAIN-018.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-018.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-018.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
- changed:
  - created the proof/public-signal schema follow-up task, handshake, and
    worklog
- validated:
  - the next slice is now explicit enough to replace the current placeholder
    proof envelope without reopening `VerifierContext`, `verifierGateHash`, or
    `shadow-batch-v1` public-input shape
- blockers:
  - none yet
- next:
  - launch one worker on `TASK-CHAIN-019`

### 2026-04-05 22:42 CST

- read:
  - `AGENTS.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-CHAIN-018.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-018.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-018.md`
  - `docs/harness/tasks/TASK-CHAIN-019.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-019.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-019.md`
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
    `VerifierArtifactBundle` now exports one explicit proof/public-signal
    schema with:
    - proof schema version/hash
    - public-signal schema version/hash
    - deterministic `verifierPublicSignals`
    - placeholder `proofData = abi.encode(proofTypeHash)`
    - final `verifierProof = abi.encode(proofSchemaHash,
      publicSignalsSchemaHash, verifierPublicSignals, proofData)`
  - updated `contracts/src/FunnyRollupVerifier.sol` so the current verifier:
    - still consumes the unchanged `VerifierContext`
    - recomputes `verifierGateHash` onchain
    - decodes the new proof/public-signal schema
    - constrains `batchEncodingHash`, `authProofHash`, and
      `verifierGateHash` against the supplied context
    - accepts only the current placeholder inner `proofData` payload
  - updated `contracts/test/FunnyRollupCore.t.sol` to pin Go/Solidity parity
    for proof schema hashes, public-signal hashes, `proofData`, and final
    proof bytes, plus reject mismatched public signals
  - updated:
    - `docs/architecture/mode-b-zk-rollup.md`
    - `docs/sql/schema.md`
    - `docs/harness/handshakes/HANDSHAKE-CHAIN-019.md`
    - `docs/harness/worklogs/WORKLOG-CHAIN-019.md`
    to document the new schema truthfully
- validated:
  - `gofmt -w internal/rollup/types.go internal/rollup/verifier_contract.go internal/rollup/verifier_contract_test.go`
  - `forge fmt contracts/src/FunnyRollupVerifier.sol contracts/test/FunnyRollupCore.t.sol`
  - `GOCACHE=/tmp/funnyoption-gocache go test ./internal/rollup`
  - `forge test --match-path contracts/test/FunnyRollupCore.t.sol`
- blockers:
  - none in this tranche
- next:
  - hand back changed files, the proof/public-signal schema contract,
    validation commands, residual limitations, and the recommended next
    prover/verifier follow-up
