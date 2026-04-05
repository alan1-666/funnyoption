# WORKLOG-CHAIN-021

### 2026-04-05 23:30 CST

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `docs/harness/tasks/TASK-CHAIN-020.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-020.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-020.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
- changed:
  - created the real proof-bytes / proving-system contract design follow-up
    task, handshake, and worklog
- validated:
  - the next slice is now explicit enough to decide whether real prover output
    can stay inside `proofData-v1` or requires an explicit `proofData-v2`
    before any worker widens into prover or cryptographic verifier work
- blockers:
  - none yet
- next:
  - launch one worker on `TASK-CHAIN-021`

### 2026-04-06 00:35 CST

- read:
  - `AGENTS.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-CHAIN-020.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-020.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-020.md`
  - `docs/harness/tasks/TASK-CHAIN-021.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-021.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-021.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
  - `internal/rollup/**`
  - `contracts/src/FunnyRollupVerifier.sol`
  - `contracts/src/FunnyRollupCore.sol`
  - `contracts/test/**`
- changed:
  - updated `docs/architecture/mode-b-zk-rollup.md` and
    `docs/sql/schema.md` to fix the first real proof-bytes /
    proving-system contract:
    - first real proof lane is fixed-vk `Groth16` on `BN254`
    - first real `proofTypeHash =
      keccak256("funny-rollup-proof-groth16-bn254-2x128-shadow-state-root-gate-v1")`
    - `proofTypeHash` now explicitly identifies the full verifier-facing proof
      contract, not only the proving-family label
    - real prover output stays inside `proofData-v1.proofBytes` as
      `abi.encode(uint256[2] a, uint256[2][2] b, uint256[2] c)`
    - outer public signals remain unchanged and are lifted to `BN254` field
      inputs via fixed `hi/lo uint128` limbs per `bytes32`
    - `proofData-v2` is only needed if verifier-relevant metadata must travel
      separately from `proofTypeHash + proofBytes`
  - updated `internal/rollup/types.go`,
    `internal/rollup/verifier_contract.go`, and
    `contracts/src/FunnyRollupVerifier.sol` with narrow non-behavioral
    placeholders / constants so the next prover/verifier worker does not need
    to guess the first real proof-type string or its semantics
  - updated `docs/harness/handshakes/HANDSHAKE-CHAIN-021.md` and this worklog
    to record the design decision, rejected options, and the next
    implementation tranche
- validated:
  - `cast keccak "funny-rollup-proof-groth16-bn254-2x128-shadow-state-root-gate-v1"`
- blockers:
  - none in this design tranche
- next:
  - hand back the changed files, chosen proving-system / proof-bytes contract,
    rejected options, migration consequences, and the recommended next real
    prover/verifier implementation tranche

### 2026-04-06 00:43 CST

- read:
  - touched diffs in
    `docs/architecture/mode-b-zk-rollup.md`,
    `docs/sql/schema.md`,
    `docs/harness/handshakes/HANDSHAKE-CHAIN-021.md`,
    `internal/rollup/types.go`,
    `internal/rollup/verifier_contract.go`, and
    `contracts/src/FunnyRollupVerifier.sol`
- changed:
  - no additional design changes; only formatting and validation after the
    first `TASK-CHAIN-021` decision landed
- validated:
  - `gofmt -w internal/rollup/types.go internal/rollup/verifier_contract.go`
  - `forge fmt contracts/src/FunnyRollupVerifier.sol`
  - `GOCACHE=/tmp/funnyoption-gocache go test ./internal/rollup`
  - `forge test --match-path contracts/test/FunnyRollupCore.t.sol`
  - `git diff --check -- docs/architecture/mode-b-zk-rollup.md docs/sql/schema.md docs/harness/handshakes/HANDSHAKE-CHAIN-021.md docs/harness/worklogs/WORKLOG-CHAIN-021.md internal/rollup/types.go internal/rollup/verifier_contract.go contracts/src/FunnyRollupVerifier.sol`
- blockers:
  - none
- next:
  - return the finalized handoff summary to commander/user
