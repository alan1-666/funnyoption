# WORKLOG-CHAIN-018

### 2026-04-05 21:36 CST

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `docs/harness/tasks/TASK-CHAIN-017.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-017.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-017.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
- changed:
  - created the first real verifier implementation follow-up task, handshake,
    and worklog
- validated:
  - the next slice is now explicit enough to consume `VerifierArtifactBundle`
    and turn the current interface-only verifier boundary into one real digest-
    constraining verifier contract without reopening public inputs
- blockers:
  - none yet
- next:
  - launch one worker on `TASK-CHAIN-018`

### 2026-04-05 21:42 CST

- read:
  - `AGENTS.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-CHAIN-017.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-017.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-017.md`
  - `docs/harness/tasks/TASK-CHAIN-018.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-018.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-018.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
  - `internal/rollup/**`
  - `contracts/src/FunnyRollupCore.sol`
  - `contracts/src/FunnyRollupVerifier.sol`
  - `contracts/test/**`
- changed:
  - updated `internal/rollup/types.go`,
    `internal/rollup/verifier_contract.go`, and
    `internal/rollup/verifier_contract_test.go` so
    `VerifierArtifactBundle` now exports:
    - the concrete verifier implementation name
    - placeholder proof-envelope metadata
    - deterministic `verifierProof = abi.encode(proofTypeHash,
      verifierGateHash)` calldata alongside the stabilized `VerifierContext`
  - replaced the interface-only verifier file in
    `contracts/src/FunnyRollupVerifier.sol` with the first real Foundry
    verifier contract that:
    - directly consumes `FunnyRollupVerifierTypes.VerifierContext`
    - requires `batchEncodingHash == keccak256("shadow-batch-v1")`
    - recomputes/constrains `verifierGateHash` onchain
    - checks the current placeholder proof envelope without claiming to be a
      final cryptographic verifier
  - updated `contracts/test/FunnyRollupCore.t.sol` to:
    - integrate `FunnyRollupCore` with the real verifier contract on the happy
      path
    - keep the mock verifier only where call-count/context inspection still
      adds value
    - add direct `FunnyRollupVerifier` tests for context/proof rejection
  - updated:
    - `docs/architecture/mode-b-zk-rollup.md`
    - `docs/sql/schema.md`
    - `docs/harness/handshakes/HANDSHAKE-CHAIN-018.md`
    - `docs/harness/worklogs/WORKLOG-CHAIN-018.md`
    to document the first real verifier tranche truthfully
- validated:
  - `gofmt -w internal/rollup/types.go internal/rollup/verifier_contract.go internal/rollup/verifier_contract_test.go`
  - `forge fmt contracts/src/FunnyRollupVerifier.sol contracts/test/FunnyRollupCore.t.sol`
  - `GOCACHE=/tmp/funnyoption-gocache go test ./internal/rollup`
  - `forge test --match-path contracts/test/FunnyRollupCore.t.sol`
- blockers:
  - none in this tranche
- next:
  - hand back changed files, the first real verifier contract boundary,
    validation commands, residual limitations, and the recommended next
    proof/verifier follow-up
