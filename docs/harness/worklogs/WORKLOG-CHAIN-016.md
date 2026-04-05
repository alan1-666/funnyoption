# WORKLOG-CHAIN-016

### 2026-04-05 20:58 CST

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `docs/harness/tasks/TASK-CHAIN-015.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-015.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-015.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
- changed:
  - created the metadata-anchored verifier/export follow-up task,
    handshake, and worklog
- validated:
  - the next slice is now explicit enough to stabilize the Go -> Solidity
    verifier artifact boundary and to close the current metadata-anchoring gap
    in `acceptVerifiedBatch(...)` without reopening public inputs
- blockers:
  - none yet
- next:
  - launch one worker on `TASK-CHAIN-016`

### 2026-04-05 21:08 CST

- read:
  - `AGENTS.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-CHAIN-015.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-015.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-015.md`
  - `docs/harness/tasks/TASK-CHAIN-016.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-016.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-016.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
  - `internal/rollup/**`
  - `contracts/src/FunnyRollupCore.sol`
  - `contracts/test/**`
- changed:
  - stabilized the acceptance artifact/export boundary in:
    - `internal/rollup/types.go`
    - `internal/rollup/verifier_contract.go`
    - `internal/rollup/verifier_contract_test.go`
  - `BuildVerifierStateRootAcceptanceContract(history, batch)` now emits one
    `solidity_export` payload with frozen contract/function identity, argument
    order, struct field names/types, enum ordinals, and normalized `bytes32`
    calldata material
  - tightened onchain acceptance anchoring in:
    - `contracts/src/FunnyRollupCore.sol`
    - `contracts/test/FunnyRollupCore.t.sol`
  - `FunnyRollupCore.acceptVerifiedBatch(...)` now requires prior matching
    `recordBatchMetadata(...)` state for the same `batch_id`
  - updated architecture/schema/handoff docs in:
    - `docs/architecture/mode-b-zk-rollup.md`
    - `docs/sql/schema.md`
    - `docs/harness/handshakes/HANDSHAKE-CHAIN-016.md`
    - `docs/harness/worklogs/WORKLOG-CHAIN-016.md`
- validated:
  - `gofmt -w internal/rollup/types.go internal/rollup/verifier_contract.go internal/rollup/verifier_contract_test.go`
  - `forge fmt contracts/src/FunnyRollupCore.sol contracts/test/FunnyRollupCore.t.sol`
  - `GOCACHE=/tmp/funnyoption-gocache go test ./internal/rollup ./internal/shared/auth`
  - `forge test --match-contract FunnyRollupCoreTest`
- blockers:
  - none in this tranche
- next:
  - hand back the stabilized verifier/export contract, validation commands,
    residual limits, and the recommended next prover/verifier follow-up
