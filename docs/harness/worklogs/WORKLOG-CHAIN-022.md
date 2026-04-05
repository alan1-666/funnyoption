# WORKLOG-CHAIN-022

### 2026-04-05 23:45 CST

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `docs/harness/tasks/TASK-CHAIN-021.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-021.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-021.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
- changed:
  - created the first real Groth16 backend implementation follow-up task,
    handshake, and worklog
- validated:
  - the next slice is now explicit enough to implement one fixed-vk
    `Groth16`/`BN254` lane with non-empty `proofBytes` and parity fixtures
    without reopening the current schema contracts
- blockers:
  - none yet
- next:
  - launch one worker on `TASK-CHAIN-022`

### 2026-04-06 00:13 CST

- read:
  - `AGENTS.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-CHAIN-021.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-021.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-021.md`
  - `docs/harness/tasks/TASK-CHAIN-022.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-022.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-022.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
  - `internal/rollup/**`
  - `contracts/src/FunnyRollupVerifier.sol`
  - `contracts/src/FunnyRollupCore.sol`
  - `contracts/test/**`
  - `foundry.toml`
- changed:
  - updated `internal/rollup/types.go`,
    `internal/rollup/verifier_contract.go`, and
    `internal/rollup/verifier_contract_test.go` so the Go exporter now emits
    non-empty fixed-fixture Groth16 `proofBytes`, exports explicit Groth16
    parity fixture details, and pins the new proof-data / verifier-proof
    artifacts
  - added `contracts/src/FunnyRollupGroth16Backend.sol` and updated
    `contracts/src/FunnyRollupVerifier.sol` so Solidity now dispatches on the
    fixed Groth16 `proofTypeHash`, decodes
    `abi.encode(uint256[2] a, uint256[2][2] b, uint256[2] c)`, derives the six
    `2 x uint128` BN254 public inputs from the unchanged outer signals, and
    performs one real fixed-vk `Groth16/BN254` pairing check
  - updated `contracts/test/FunnyRollupCore.t.sol` to pin Go/Foundry parity
    for limb splitting, tuple codec, verifier verdict, and one end-to-end
    `FunnyRollupCore -> FunnyRollupVerifier -> FunnyRollupGroth16Backend`
    acceptance path
  - updated `docs/architecture/mode-b-zk-rollup.md`,
    `docs/sql/schema.md`, and `docs/harness/handshakes/HANDSHAKE-CHAIN-022.md`
    to record the fixed-vk backend boundary and keep the repo-truth caveat
    explicit
- validated:
  - `gofmt -w internal/rollup/types.go internal/rollup/verifier_contract.go internal/rollup/verifier_contract_test.go`
  - `forge fmt contracts/src/FunnyRollupGroth16Backend.sol contracts/src/FunnyRollupVerifier.sol contracts/test/FunnyRollupCore.t.sol`
  - `GOCACHE=/tmp/funnyoption-gocache go test ./internal/rollup`
  - `forge test --match-path contracts/test/FunnyRollupCore.t.sol`
- blockers:
  - none in this tranche
- next:
  - hand back the fixed Groth16 backend boundary, validation commands,
    residual limitations, and the recommended next prover/verifier follow-up
