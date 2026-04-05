# WORKLOG-CHAIN-010

### 2026-04-05 17:05 CST

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `docs/harness/tasks/TASK-CHAIN-009.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-009.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-009.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
- changed:
  - created the second shadow-rollup tranche task, handshake, and worklog
- validated:
  - the next slice is now explicit enough to extend replay from trading phase
    into settlement phase without prematurely widening into prover or full L1
    implementation
- blockers:
  - none yet
- next:
  - launch one worker on `TASK-CHAIN-010`

### 2026-04-05 17:14 CST

- read:
  - `AGENTS.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-CHAIN-009.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-009.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-009.md`
  - `docs/harness/tasks/TASK-CHAIN-010.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-010.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-010.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
  - `internal/rollup/**`
  - `internal/settlement/service/**`
  - `internal/matching/service/**`
  - `internal/shared/kafka/messages.go`
  - `internal/api/handler/order_handler.go`
  - `internal/chain/service/processor.go`
  - `contracts/src/FunnyVault.sol`
  - `foundry.toml`
- changed:
  - extended settlement-phase shadow capture:
    - `internal/settlement/service/sql_store.go`
    - `internal/settlement/service/processor.go`
    - `internal/settlement/service/server.go`
    - `internal/settlement/service/store.go`
    - `internal/settlement/service/rollup_shadow.go`
  - extended rollup replay + batch contract:
    - `internal/rollup/types.go`
    - `internal/rollup/replay.go`
    - `internal/rollup/witness.go`
    - `internal/rollup/replay_test.go`
  - added minimal Foundry-only L1 batch metadata placeholder:
    - `contracts/src/FunnyRollupCore.sol`
    - `contracts/test/DSTest.sol`
    - `contracts/test/FunnyRollupCore.t.sol`
    - `foundry.toml`
  - updated docs / thread artifacts:
    - `docs/architecture/mode-b-zk-rollup.md`
    - `docs/sql/schema.md`
    - `docs/harness/handshakes/HANDSHAKE-CHAIN-010.md`
    - `docs/harness/worklogs/WORKLOG-CHAIN-010.md`
- validated:
  - `gofmt -w internal/rollup/*.go internal/settlement/service/*.go`
  - `go test ./internal/rollup ./internal/settlement/service`
  - `go test -run TestReplayStoredBatchesSettlementDeterministic -v ./internal/rollup`
  - `forge test --match-contract FunnyRollupCoreTest`
  - deterministic replay proof including settlement-phase inputs:
    - `BalancesRoot = 69979c2d18a7145642d6a03d572dd4443d46a3c3b7479f1f73e41440a3737f97`
    - `OrdersRoot = 1854c9b450264fa6410c58d2f66c3b7f32425fc528d88fac9f5624d2839f93ce`
    - `PositionsFundingRoot = e41b4aa4db3d89f132e2199b6fb7d2df7ca974034003fa6f4a313d54deb60fe9`
    - `WithdrawalsRoot = 4d8d05be4ce388de39e86fd781a0625c0f833a07b3a815a5c99e5917c8302c96`
    - `StateRoot = 6ccfa7ba6cca1f94177e85ca10ddb581e3aa8b200a21f9a7c8399add9f6b7f4a`
    - rerunning the same test produced the same root set from durable
      `shadow-batch-v1` inputs without consulting live SQL snapshots or Kafka
      offsets
- blockers:
  - no delivery blocker for this tranche
  - residual shadow-only limits remain around:
    - `orders_root.nonce_root` is still `ZeroNonceRoot()` because the
      canonical API/auth nonce input is not yet carried into the shadow batch
    - `insurance_root` is still a deterministic zero placeholder
    - `withdrawals_root` still mirrors direct-vault queued withdrawals rather
      than future canonical claim-nullifier truth
    - no prover, verifier, L1 finality, or production withdrawal-claim rewrite
- next:
  - recommended next prover/L1 tranche:
    - lift API/auth nonce advances into durable batch inputs so
      `orders_root.nonce_root` becomes truthful shadow instead of a placeholder
    - bind prover public inputs directly to the now-fixed
      `shadow-batch-v1` contract and the `FunnyRollupCore.recordBatchMetadata`
      surface
    - add verifier-gated batch acceptance and later canonical withdrawal
      nullifier handling only after the nonce/public-input lane is wired
