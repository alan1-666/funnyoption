# WORKLOG-CHAIN-030

### 2026-04-06 23:48 CST

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
  - `docs/harness/tasks/TASK-CHAIN-029.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-029.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-029.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
  - `internal/rollup/**`
  - `internal/api/handler/sql_store.go`
- changed:
  - created `TASK-CHAIN-030`, `HANDSHAKE-CHAIN-030`, and this worklog for one
    current-session tranche that widens accepted truth from withdrawals into
    balances / positions / settlement-payout read surfaces
- validated:
  - scope stays narrow:
    - no forced-withdraw freeze / escape hatch yet
    - no proof/public-signal contract rewrite
    - no mutable backend write-path switch yet
- next:
  - derive deterministic accepted replay snapshots from accepted batches
  - materialize accepted balance / position / payout tables
  - switch `/balances`, `/positions`, and `/payouts` to accepted truth when
    accepted batches exist

### 2026-04-07 00:17 CST

- changed:
  - added deterministic accepted replay snapshot export in
    `internal/rollup/accepted_snapshot.go`
  - extended `internal/rollup/store.go` so accepted-submission materialization
    now rebuilds:
    - `rollup_accepted_balances`
    - `rollup_accepted_positions`
    - `rollup_accepted_payouts`
  - added `migrations/018_rollup_accepted_read_truth.sql`
  - switched `SQLStore.ListBalances/ListPositions/ListPayouts` to prefer
    accepted read truth once accepted batches exist
  - extended local anvil fresh-start reset in `scripts/dev-up.sh` so the new
    accepted snapshot tables are also cleared on a fresh local chain bootstrap
- validated:
  - `GOCACHE=/tmp/funnyoption-gocache go test ./internal/rollup ./internal/api/handler ./internal/chain/service ./internal/api`
  - `forge test --offline --match-path contracts/test/FunnyRollupCore.t.sol`
  - `bash -n scripts/dev-up.sh`
  - live local API proof after seeding one accepted snapshot:
    - `GET /api/v1/balances?user_id=1001&limit=20`
      returned `POSITION:88:YES=6` and `USDT=1390`
    - `GET /api/v1/positions?user_id=1001&limit=20`
      returned `market_id=88 quantity=6 settled_quantity=6`
    - `GET /api/v1/payouts?user_id=1001&limit=20`
      returned `evt_settlement_88_1 payout_amount=600`
- blockers:
  - matching / account / settlement write truth still has not switched to
    accepted-root first
  - forced-withdraw / freeze / escape hatch runtime still does not exist
  - prover lane is still not a full state-transition circuit
- next:
  - continue from accepted read truth into:
    - forced-withdraw request/runtime foundations
    - accepted-root-first mutable settlement/accounting boundaries
    - fuller state-transition prover inputs
