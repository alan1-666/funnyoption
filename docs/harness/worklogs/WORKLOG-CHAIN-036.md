# WORKLOG-CHAIN-036

### 2026-04-07 03:05 CST

- thread:
  - commander+worker merged
- read:
  - `AGENTS.md`
  - `PLAN.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/COMMANDER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-CHAIN-035.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-035.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-035.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
  - `internal/rollup/**`
  - `internal/api/handler/sql_store.go`
  - `internal/chain/service/**`
  - `contracts/src/FunnyRollupCore.sol`
  - `contracts/src/FunnyVault.sol`
  - `contracts/src/FunnyRollupVerifier.sol`
- changed:
  - created `TASK-CHAIN-036`, `HANDSHAKE-CHAIN-036`, and this worklog for the
    merged escape-claim / accepted-truth / proving-lane closeout tranche
- validated:
  - planning/docs only
- next:
  - implement accepted escape collateral roots and frozen Merkle-proof claims
  - then widen accepted/frozen truth
  - then upgrade the proving lane to consume state-transition witness material

### 2026-04-07 18:00 CST

- changed:
  - accepted/frozen truth:
    - `internal/api/handler/sql_store.go`
    - `internal/api/handler/sql_store_accepted_runtime_test.go`
    - accepted financial read truth is now visible once accepted batches exist,
      not only after freeze
  - local full-flow:
    - `cmd/local-lifecycle/main.go`
    - `cmd/local-lifecycle/trading_key_oracle_flow.go`
    - the default trading-key oracle flow now reaches:
      - accepted batch
      - anchored escape root
      - forced withdrawal
      - freeze
      - Merkle-proof escape collateral claim
  - proving / accepted replay:
    - `internal/rollup/transition_witness.go`
    - `internal/rollup/verifier_contract.go`
    - `internal/rollup/groth16_lane.go`
    - `contracts/src/FunnyRollupVerifier.sol`
    - `contracts/src/FunnyRollupGroth16Backend.sol`
    - preferred lane now binds state-transition witness material instead of only
      outer digest equality
  - accepted leaf rematerialization:
    - `internal/rollup/store.go`
    - `internal/rollup/accepted_test.go`
    - accepted escape / withdrawal leaves now preserve existing claim runtime
      state across replay / restart instead of resetting `CLAIMED` rows back to
      `CLAIMABLE`
- validated:
  - `GOCACHE=/tmp/funnyoption-gocache go test ./internal/rollup ./internal/chain/service ./internal/api/handler ./internal/api ./cmd/local-lifecycle -timeout=20m`
  - `forge test --offline --match-path contracts/test/FunnyRollupCore.t.sol`
  - `git diff --check`
  - fresh local dev flow:
    - `./scripts/dev-down.sh`
    - `touch .run/dev/local-chain-fresh-start`
    - `FUNNYOPTION_LOCAL_CHAIN_FORCED_WITHDRAWAL_GRACE_PERIOD=2 ./scripts/dev-up.sh`
    - `GOCACHE=/tmp/funnyoption-gocache go run ./cmd/local-lifecycle --flow trading-key-oracle -timeout=5m`
  - post-flow evidence:
    - accepted batch count reached `2`
    - `/api/v1/balances` for buyer returned `available=0`
    - `/api/v1/rollup/escape-collateral` returned batch-2 claim with
      `claim_status=CLAIMED`
    - `rollup_accepted_escape_leaves` kept batch-2 buyer leaf at `CLAIMED`
      after the flow completed
- residuals:
  - unresolved-open-position emergency handling at freeze is still narrow
  - prover/backend is still repo-local first cut, not a production proving fleet
  - the repo is materially closer to `Mode B`, but not every live truth
    boundary is yet fully replaced by accepted roots

### 2026-04-07 19:30 CST (hotfix)

- thread: manual review
- bug found:
  - staging E2E (`staging-concurrency-orders.mjs`) failed because the
    `acceptedFinancialTruthVisible` gate returned `true` whenever
    `rollup_accepted_batches` had any rows, even when the rollup was
    **not frozen**
  - this caused `ListBalances`, `ListPositions`, and `ListPayouts` to
    use accepted-only queries (`listAccepted*`), which dropped all
    live-but-not-yet-accepted data—new positions from recent trades, new
    balances from recent deposits, and new payouts from recent
    settlements were invisible through the public API
  - root cause: the two distinct intents "rollup is frozen" and "accepted
    batches exist" were conflated into one boolean; the accepted-only path
    is only correct under freeze (safety lockdown), while normal
    operation needs the merged path that unions accepted and live data
- changed:
  - `internal/api/handler/sql_store.go`:
    - `ListBalances`, `ListPositions`, `ListPayouts` now call
      `rollupFrozen` directly; accepted-only queries are used only when
      `frozen = true`, otherwise the merged queries run
    - `BuildLiabilityReport` and market runtime payout stats also
      switched from `acceptedFinancialTruthVisible` to `rollupFrozen`
    - removed unused `acceptedFinancialTruthVisible` and
      `acceptedReadTruthVisible` helpers
- impact:
  - when not frozen: API shows the best-available merged view
    (accepted data takes precedence for existing keys; live data fills in
    for anything not yet accepted)
  - when frozen: API shows only accepted data (unchanged, correct for
    safety lockdown mode)
- validated:
  - `go build ./internal/api/...` passes
  - staging E2E pending rerun after deploy
