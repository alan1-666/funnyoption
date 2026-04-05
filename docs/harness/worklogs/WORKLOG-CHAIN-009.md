# WORKLOG-CHAIN-009

### 2026-04-05 16:35 CST

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `docs/harness/tasks/TASK-CHAIN-008.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-008.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-008.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/architecture/order-flow.md`
  - `docs/sql/schema.md`
- changed:
  - created the first shadow-rollup implementation task, handshake, and worklog
- validated:
  - the architecture lane is now explicit enough to start one narrow
    implementation tranche without prematurely widening into prover or full L1
    contract work
- blockers:
  - none yet
- next:
  - launch one worker on `TASK-CHAIN-009`

### 2026-04-05 16:50 CST

- read:
  - `AGENTS.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-CHAIN-008.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-008.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-008.md`
  - `docs/harness/tasks/TASK-CHAIN-009.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-009.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-009.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/architecture/order-flow.md`
  - `docs/sql/schema.md`
  - `internal/api/handler/order_handler.go`
  - `internal/matching/service/consumer.go`
  - `internal/matching/service/sql_store.go`
  - `internal/account/service/event_processor.go`
  - `internal/account/service/balance_book.go`
  - `internal/account/service/sql_store.go`
  - `internal/settlement/service/processor.go`
  - `internal/settlement/service/sql_store.go`
  - `internal/chain/service/processor.go`
  - `internal/chain/service/sql_store.go`
  - `internal/shared/kafka/messages.go`
  - `migrations/001_init.sql`
- changed:
  - added shadow-rollup runtime/storage package:
    - `internal/rollup/types.go`
    - `internal/rollup/hash.go`
    - `internal/rollup/store.go`
    - `internal/rollup/replay.go`
    - `internal/rollup/replay_test.go`
  - added dedicated shadow-rollup persistence migration:
    - `migrations/014_rollup_shadow_lane.sql`
  - wired canonical shadow-input capture from matching:
    - `internal/matching/service/sql_store.go`
    - `internal/matching/service/rollup_shadow.go`
    - `internal/matching/service/server.go`
  - wired canonical shadow-input capture from chain deposit/withdrawal mirrors:
    - `internal/chain/service/processor.go`
    - `internal/chain/service/server.go`
  - updated docs to mark the landed boundary as shadow-only and explicit:
    - `docs/sql/schema.md`
    - `docs/architecture/mode-b-zk-rollup.md`
    - `docs/harness/handshakes/HANDSHAKE-CHAIN-009.md`
- validated:
  - `go test ./internal/rollup ./internal/matching/service ./internal/chain/service`
  - `go test -run TestReplayStoredBatchesDeterministic -v ./internal/rollup`
  - `git diff --check`
  - `source .env.local && psql \"$FUNNYOPTION_POSTGRES_DSN\" -v ON_ERROR_STOP=1 -f migrations/014_rollup_shadow_lane.sql`
  - `source .env.local && psql \"$FUNNYOPTION_POSTGRES_DSN\" -Atc \"SELECT table_name FROM information_schema.tables WHERE table_schema='public' AND table_name IN ('rollup_shadow_journal_entries','rollup_shadow_batches') ORDER BY table_name;\"`
  - deterministic replay proof from durable batch input:
    - `BalancesRoot = 1c4d41ec94ce0d136b9bfe7145ab69def481a0228551458a62f6abbbac5dfd8d`
    - `OrdersRoot = 1854c9b450264fa6410c58d2f66c3b7f32425fc528d88fac9f5624d2839f93ce`
    - `PositionsFundingRoot = 08febc79853246cb355f681c0d624f77421a1b77377458969f795cf0bff7a375`
    - `WithdrawalsRoot = e16ad012fcace2280ee6b306087f6214c639c054cc040f46a560a08629f8f755`
    - `StateRoot = bb61fc3c97b6d967916a034afbf362c57b101e78c79156f9ee542ae55873cdb6`
    - rerunning the same replay test produced the same root set without
      consulting live SQL snapshots or Kafka offsets
- blockers:
  - no production blocker for this tranche
  - residual shadow-only gaps remain around:
    - replay-protection state is still implicit because `orders_root` currently
      uses deterministic `ZeroNonceRoot()` instead of a truthful shadow nonce
      namespace
    - market-resolution-triggered order cancellation / settlement payout inputs
    - funding / insurance namespaces
    - prover / verifier / L1 state-update path
    - canonical slow-withdraw claim / nullifier / forced-withdraw runtime
- next:
  - recommended next tranche:
    - materialize market-resolution + settlement-payout shadow inputs
    - define one explicit witness/public-input contract for the existing
      `shadow-batch-v1`
    - add `FunnyRollupCore` batch metadata / state-root event surface on L1
    - only then start prover coordinator work against the already-fixed replay
      contract
