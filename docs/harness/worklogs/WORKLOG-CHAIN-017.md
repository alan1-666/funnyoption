# WORKLOG-CHAIN-017

### 2026-04-05 21:15 CST

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `docs/harness/tasks/TASK-CHAIN-016.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-016.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-016.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
- changed:
  - created the first prover/verifier artifact follow-up task, handshake, and
    worklog
- validated:
  - the next slice is now explicit enough to bind Go-side `solidity_export`
    output to one real verifier-facing contract boundary without reopening
    public inputs or metadata anchoring
- blockers:
  - none yet
- next:
  - launch one worker on `TASK-CHAIN-017`

### 2026-04-05 21:28 CST

- read:
  - `AGENTS.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-CHAIN-016.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-016.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-016.md`
  - `docs/harness/tasks/TASK-CHAIN-017.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-017.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-017.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
  - `internal/rollup/**`
  - `contracts/src/FunnyRollupCore.sol`
  - `contracts/test/**`
- changed:
  - added the first deterministic prover/verifier artifact lane in:
    - `internal/rollup/types.go`
    - `internal/rollup/verifier_contract.go`
    - `internal/rollup/verifier_contract_test.go`
  - `BuildVerifierArtifactBundle(history, batch)` now directly consumes
    `BuildVerifierStateRootAcceptanceContract(...).SolidityExport` and emits:
    - unchanged acceptance contract
    - deterministic `authProofHash`
    - deterministic `verifierGateHash`
    - verifier-facing `IFunnyRollupBatchVerifier.verifyBatch(context, proof)`
      export
  - upgraded the contract verifier boundary in:
    - `contracts/src/FunnyRollupVerifier.sol`
    - `contracts/src/FunnyRollupCore.sol`
    - `contracts/test/FunnyRollupCore.t.sol`
  - `FunnyRollupCore` now passes a full verifier context instead of one bare
    hash-only stub
  - pinned one shared Go/Solidity digest-parity fixture for
    `verifierGateHash`
  - updated tranche docs/handoff in:
    - `docs/architecture/mode-b-zk-rollup.md`
    - `docs/sql/schema.md`
    - `docs/harness/handshakes/HANDSHAKE-CHAIN-017.md`
    - `docs/harness/worklogs/WORKLOG-CHAIN-017.md`
- validated:
  - `gofmt -w internal/rollup/types.go internal/rollup/verifier_contract.go internal/rollup/verifier_contract_test.go`
  - `forge fmt contracts/src/FunnyRollupCore.sol contracts/src/FunnyRollupVerifier.sol contracts/test/FunnyRollupCore.t.sol`
  - `GOCACHE=/tmp/funnyoption-gocache go test ./internal/rollup ./internal/shared/auth`
  - `forge test --match-contract FunnyRollupCoreTest`
  - `git diff --check -- internal/rollup contracts/src contracts/test docs/architecture/mode-b-zk-rollup.md docs/sql/schema.md docs/harness/handshakes/HANDSHAKE-CHAIN-017.md docs/harness/worklogs/WORKLOG-CHAIN-017.md`
- blockers:
  - none in this tranche
- next:
  - hand back changed files, the first prover/verifier artifact contract,
    validation commands, residual limitations, and the recommended next real
    verifier / prover follow-up
