# WORKLOG-CHAIN-028

### 2026-04-06 20:19 CST

- thread:
  - commander+worker merged
- read:
  - `AGENTS.md`
  - `PLAN.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/COMMANDER.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-CHAIN-027.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-027.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-027.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
  - `internal/rollup/**`
  - `internal/chain/service/**`
  - `cmd/rollup/**`
  - `contracts/src/FunnyRollupCore.sol`
- changed:
  - created `TASK-CHAIN-028`, `HANDSHAKE-CHAIN-028`, and this worklog for one
    current-session tranche that hardens the live submission runtime with
    onchain reconciliation and a `submit-until-idle` command path
- validated:
  - scope stays narrow:
    - no production truth switch
    - no proof/public-signal contract changes
    - no withdrawal rewrite
- blockers:
  - none yet
- next:
  - implement `FunnyRollupCore` read/reconciliation helpers
  - teach the submitter to require visible onchain state before advancing
  - add `cmd/rollup -mode=submit-until-idle`

### 2026-04-06 20:31 CST

- thread:
  - commander+worker merged
- changed:
  - added `internal/chain/service/rollup_core_state.go` so the live submitter
    can read:
    - `latestBatchId`
    - `latestStateRoot`
    - `batchMetadata(batchId)`
    - `latestAcceptedBatchId`
    - `latestAcceptedStateRoot`
    - `acceptedBatches(batchId)`
    directly from `FunnyRollupCore`
  - hardened `RollupSubmissionProcessor` so:
    - record receipt success alone no longer advances the lane
    - accept receipt success alone no longer marks `ACCEPTED`
    - persisted submission JSON is decoded and compared against visible
      onchain state before the runtime advances
  - added `RunRollupSubmissionUntilIdle(...)` and
    `cmd/rollup -mode=submit-until-idle`
  - updated `scripts/local-chain-up.sh` so generated local env files now use
    `export KEY=...` and can be sourced directly into follow-up commands
- validated:
  - `gofmt -w internal/rollup/submission.go internal/chain/service/rpc_pool.go internal/chain/service/rollup_core_state.go internal/chain/service/rollup_submitter.go internal/chain/service/rollup_submitter_test.go cmd/rollup/main.go`
  - `go test ./internal/rollup ./internal/chain/service ./cmd/rollup`
  - `forge test --match-path contracts/test/FunnyRollupCore.t.sol`
  - `bash -n ./scripts/local-chain-up.sh`
  - `./scripts/local-chain-up.sh`
  - `set -a && source .env.local && source .run/dev/local-chain.env && set +a && go run ./cmd/rollup -mode=submit-until-idle -timeout=15s`
  - observed local runtime output:
    - one stable `NOOP` run when no pending submission exists
- blockers:
  - local runtime path is now chain-configured and reconciliation-aware, but
    this workstation still has no prepared pending submission row to exercise
    one full metadata+acceptance broadcast in the live command path
- next:
  - if we keep pushing this lane, the next slice should move from the current
    narrow fixed-vk proving contract toward a richer state-transition circuit
    while preserving the now-stable submission/reconciliation boundary
