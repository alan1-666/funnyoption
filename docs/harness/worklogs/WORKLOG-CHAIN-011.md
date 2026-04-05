# WORKLOG-CHAIN-011

### 2026-04-05 17:25 CST

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `docs/harness/tasks/TASK-CHAIN-010.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-010.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-010.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
- changed:
  - created the nonce/public-input follow-up task, handshake, and worklog
- validated:
  - the next slice is now explicit enough to close the last material
    replay-protection gap before prover/verifier work starts
- blockers:
  - none yet
- next:
  - launch one worker on `TASK-CHAIN-011`

### 2026-04-05 17:33 CST

- read:
  - `AGENTS.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-CHAIN-010.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-010.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-010.md`
  - `docs/harness/tasks/TASK-CHAIN-011.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-011.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-011.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
  - `internal/rollup/**`
  - `internal/api/**`
  - `internal/shared/auth/**`
  - `contracts/src/FunnyRollupCore.sol`
- changed:
  - extended the shadow batch witness lane with truthful API/auth nonce inputs:
    - `internal/rollup/types.go`
    - `internal/rollup/replay.go`
    - `internal/rollup/witness.go`
    - `internal/rollup/replay_test.go`
  - wired API nonce advancement into the durable shadow journal:
    - `internal/api/handler/sql_store.go`
    - `internal/api/handler/rollup_shadow.go`
    - `internal/api/handler/rollup_shadow_test.go`
    - `internal/api/server.go`
  - clarified the shadow-only nonce/public-input contract:
    - `docs/architecture/mode-b-zk-rollup.md`
    - `docs/sql/schema.md`
    - `docs/harness/handshakes/HANDSHAKE-CHAIN-011.md`
    - `docs/harness/worklogs/WORKLOG-CHAIN-011.md`
- validated:
  - `gofmt -w internal/rollup/types.go internal/rollup/replay.go internal/rollup/witness.go internal/rollup/replay_test.go internal/api/handler/sql_store.go internal/api/handler/rollup_shadow.go internal/api/handler/rollup_shadow_test.go internal/api/server.go`
  - `go test ./internal/rollup`
  - `go test ./internal/api/handler`
  - `go test ./internal/shared/auth`
  - `go test -run TestReplayStoredBatchesSettlementDeterministic -v ./internal/rollup`
  - `git diff --check`
  - deterministic replay proof with truthful nonce subtree:
    - `BalancesRoot = 664363d42e464f547172914a37e7efe016d51bd1311646192c0bad9df996f85e`
    - `OrdersRoot = 4a35c4ac0d423b779cdf59c3d9651e09185cf745bdcc99d27fb13aa3c9d3202b`
    - `PositionsFundingRoot = ce98928eb740baa5c8c54e227f52ca43c1075df803a199a8dd5d0ba5348c0ce0`
    - `WithdrawalsRoot = 4d8d05be4ce388de39e86fd781a0625c0f833a07b3a815a5c99e5917c8302c96`
    - `StateRoot = 05e17686f48c7f0f5a3dd20190fcbafe863d6abf30bdbbadeab23a418f786990`
    - rerunning the same replay test produced the same root set from durable
      `shadow-batch-v1` input only, with no SQL snapshot or Kafka offset read
- blockers:
  - no delivery blocker for this tranche
  - remaining non-Mode-B limits stay explicit:
    - nonce truth now mirrors the current API/auth monotonic floor, but it
      still allows nonce gaps and still depends on operator-side signature
      checks
    - `insurance_root` remains a deterministic zero placeholder
    - `withdrawals_root` still mirrors direct-vault queue state, not canonical
      claim-nullifier truth
    - no prover, verifier, verifier-gated batch acceptance, or production
      withdrawal-claim rewrite landed here
- next:
  - recommended next prover/verifier tranche:
    - bind prover witness logic directly to the stabilized `NONCE_ADVANCED` +
      `shadow-batch-v1` contract without reopening public-input shape
    - decide whether the current monotonic-floor nonce semantics are sufficient
      for the first proof lane or whether a stricter gapless/auth-gadget
      contract is required
    - add verifier-gated batch acceptance against `FunnyRollupCore` only after
      that nonce/signature boundary is fixed
