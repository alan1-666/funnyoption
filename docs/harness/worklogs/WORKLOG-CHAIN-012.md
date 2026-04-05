# WORKLOG-CHAIN-012

### 2026-04-05 17:40 CST

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `docs/harness/tasks/TASK-CHAIN-011.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-011.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-011.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
- changed:
  - created the proof-lane nonce/verifier design follow-up task, handshake,
    and worklog
- validated:
  - the next slice is now explicit enough to resolve the last major nonce/auth
    contract question before prover/verifier implementation starts
- blockers:
  - none yet
- next:
  - launch one design worker on `TASK-CHAIN-012`

### 2026-04-05 17:50 CST

- read:
  - `AGENTS.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-CHAIN-011.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-011.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-011.md`
  - `docs/harness/tasks/TASK-CHAIN-012.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-012.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-012.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
  - `docs/architecture/direct-deposit-session-key.md`
  - `internal/rollup/**`
  - `internal/api/**`
  - `internal/shared/auth/**`
  - `contracts/src/FunnyRollupCore.sol`
- changed:
  - fixed the tranche-1 proof-lane nonce/auth decision and verifier-gated
    `FunnyRollupCore` boundary in:
    - `docs/architecture/mode-b-zk-rollup.md`
    - `docs/sql/schema.md`
    - `docs/harness/handshakes/HANDSHAKE-CHAIN-012.md`
  - recorded this worker handoff in:
    - `docs/harness/worklogs/WORKLOG-CHAIN-012.md`
- validated:
  - docs consistency review against current runtime:
    - `AdvanceSessionNonce` still enforces monotonic-floor `last_order_nonce <
      nonce`
    - `NONCE_ADVANCED` is emitted before downstream freeze / publish completes,
      so nonce advancement truth is not a gapless order-count contract
    - canonical V2 trading-key auth remains the intended baseline; deprecated
      `/api/v1/sessions` stays compatibility-only
  - `git diff --check`
- blockers:
  - no delivery blocker for this design tranche
  - explicit residual limits remain:
    - current product is still not `Mode B`
    - no prover, verifier, or production withdrawal-claim rewrite landed
    - first prover tranche still needs one narrow auth witness lane before
      verifier-gated batch acceptance can become sound
- next:
  - recommended implementation follow-up:
    - add durable or prover-consumable auth witness material binding
      `NONCE_ADVANCED` to canonical V2 trading-key order authorization without
      reopening public inputs
    - migrate repo proof tooling away from deprecated `/api/v1/sessions`
      before treating batches as verifier-eligible
