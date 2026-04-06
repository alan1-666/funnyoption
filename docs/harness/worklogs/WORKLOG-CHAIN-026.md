# WORKLOG-CHAIN-026

### 2026-04-06 17:03 CST

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
  - `docs/harness/tasks/TASK-CHAIN-023.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-023.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-023.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
  - `internal/rollup/**`
  - `contracts/src/FunnyRollupCore.sol`
  - `contracts/src/FunnyRollupVerifier.sol`
- changed:
  - created `TASK-CHAIN-026`, `HANDSHAKE-CHAIN-026`, and this worklog for one
    current-session tranche that bridges stored shadow batches into persisted
    onchain submission payloads
- validated:
  - scope is narrow enough to finish in one session without reopening the proof
    envelope or claiming a production truth switch
- blockers:
  - none yet
- next:
  - implement the deterministic shadow submission bundle, persistence lane, and
    minimal repo command

### 2026-04-06 17:28 CST

- thread:
  - commander+worker merged
- changed:
  - added `internal/rollup/submission.go` and
    `internal/rollup/submission_test.go` with one deterministic
    `BuildShadowBatchSubmissionBundle(history, batch)` contract that combines:
    - `ShadowBatchContract`
    - `VerifierArtifactBundle`
    - stable `recordBatchMetadata(...)` calldata
    - stable `acceptVerifiedBatch(...)` calldata
    - explicit `READY` / `BLOCKED_AUTH` submission readiness
  - extended `internal/rollup/store.go` so the rollup store can now
    `PrepareNextSubmission(...)`, reuse the earliest batch without a submission
    row, or materialize the next batch first if needed, then persist one
    deterministic `rollup_shadow_submissions` row
  - added `cmd/rollup/main.go` as a minimal repo command that prepares the next
    submission bundle and prints it as JSON
  - added `migrations/015_rollup_shadow_submission_lane.sql` for the durable
    submission lane
  - updated `docs/architecture/mode-b-zk-rollup.md` and `docs/sql/schema.md`
    so the new offchain-to-onchain acceptance bridge is part of the repo's
    explicit architecture contract
- validated:
  - `gofmt -w internal/rollup/types.go internal/rollup/submission.go internal/rollup/submission_test.go internal/rollup/store.go cmd/rollup/main.go`
  - `go test ./internal/rollup ./cmd/rollup`
  - `forge test --match-path contracts/test/FunnyRollupCore.t.sol`
  - `git diff --check`
  - `psql` apply check for `migrations/015_rollup_shadow_submission_lane.sql`
- blockers:
  - no live tx broadcasting exists yet; this tranche only prepares/persists the
    onchain acceptance payload
  - the local migration validation applied the new submission table to the
    current local Postgres because the migration file owns its own
    `BEGIN/COMMIT`
- next:
  - if we continue this lane, the next slice should add a narrow runtime that
    consumes persisted submission bundles and submits
    `recordBatchMetadata(...)` / `acceptVerifiedBatch(...)` onchain
