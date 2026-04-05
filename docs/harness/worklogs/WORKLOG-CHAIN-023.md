# WORKLOG-CHAIN-023

### 2026-04-06 00:16 CST

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `docs/harness/tasks/TASK-CHAIN-022.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-022.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-022.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
- changed:
  - created the batch-specific fixed-vk Groth16 proof artifact follow-up task,
    handshake, and worklog
- validated:
  - the next slice is now explicit enough to replace the one shared fixture
    proof with deterministic batch-specific proof artifacts without reopening
    the outer envelope or `proofData-v1`
- blockers:
  - none yet
- next:
  - launch one worker on `TASK-CHAIN-023`

### 2026-04-06 00:45 CST

- read:
  - `AGENTS.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-CHAIN-022.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-022.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-022.md`
  - `docs/harness/tasks/TASK-CHAIN-023.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-023.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-023.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
  - `internal/rollup/**`
  - `contracts/src/FunnyRollupGroth16Backend.sol`
  - `contracts/src/FunnyRollupVerifier.sol`
  - `contracts/test/**`
  - `foundry.toml`
- changed:
  - added `internal/rollup/groth16_lane.go` with a deterministic fixed-vk
    `Groth16/BN254` proving lane that derives six `2 x uint128` public-input
    limbs from the actual outer
    `{batchEncodingHash, authProofHash, verifierGateHash}` tuple and emits
    batch-specific `proofBytes`
  - added `internal/rollup/cmd/fixedvk-artifacts/main.go` so the repo can
    export the deterministic backend contract source plus reproducible
    batch-specific proof artifacts for a supplied outer-signal tuple
  - updated `internal/rollup/verifier_contract.go` and
    `internal/rollup/verifier_contract_test.go` so the Go-side artifact bundle
    now embeds batch-specific inner `proofBytes` instead of reusing one shared
    fixture proof, while keeping the outer proof/public-signal envelope,
    `proofData-v1`, fixed `proofTypeHash`, and `shadow-batch-v1` public-input
    shape unchanged
  - refreshed `contracts/src/FunnyRollupGroth16Backend.sol` constants from the
    same deterministic fixed-vk lane and updated
    `contracts/test/FunnyRollupCore.t.sol` to pin Go/Foundry parity for two
    batch-specific artifacts, including limb splitting, `proofBytes`,
    `proofData`, `verifierProof`, and verifier verdicts
  - updated `docs/architecture/mode-b-zk-rollup.md`,
    `docs/sql/schema.md`, and `docs/harness/handshakes/HANDSHAKE-CHAIN-023.md`
    to record the new batch-specific artifact boundary without changing the
    repo's production-truth claims
- validated:
  - `gofmt -w internal/rollup/groth16_lane.go internal/rollup/verifier_contract.go internal/rollup/verifier_contract_test.go internal/rollup/cmd/fixedvk-artifacts/main.go`
  - `forge fmt contracts/test/FunnyRollupCore.t.sol contracts/src/FunnyRollupGroth16Backend.sol contracts/src/FunnyRollupVerifier.sol`
  - `GOCACHE=/tmp/funnyoption-gocache go test ./internal/rollup`
  - `forge test --match-path contracts/test/FunnyRollupCore.t.sol`
- blockers:
  - none in this tranche
- next:
  - hand back the changed files, batch-specific proof artifact pipeline,
    validation commands, residual limitations, and the recommended next
    prover/verifier follow-up
