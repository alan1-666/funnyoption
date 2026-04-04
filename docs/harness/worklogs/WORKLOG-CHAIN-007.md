# WORKLOG-CHAIN-007

### 2026-04-05 00:31 CST

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `HANDSHAKE-CHAIN-006.md`
  - `WORKLOG-CHAIN-006.md`
  - `docs/architecture/oracle-settled-crypto-markets.md`
- changed:
  - created a follow-up reliability task for the remaining
    `OBSERVED-but-publish-failed` oracle dispatch gap
- validated:
  - `TASK-CHAIN-006` closed the duplicate-emit review blocker, so the remaining
    oracle work can now stay narrowly focused on dispatch retry semantics
- blockers:
  - none yet
- next:
  - assign one worker to land a retry-safe dispatch marker or equivalent narrow
    contract

### 2026-04-05 00:44 CST

- read:
  - `AGENTS.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-CHAIN-007.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-007.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-007.md`
  - `docs/architecture/oracle-settled-crypto-markets.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-006.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-006.md`
  - `internal/oracle/service/worker.go`
  - `internal/oracle/service/sql_store.go`
  - `internal/settlement/service/processor.go`
  - `internal/account/service/event_processor.go`
  - `internal/settlement/service/sql_store.go`
  - `internal/oracle/service/metadata.go`
  - `internal/oracle/service/worker_test.go`
  - `internal/settlement/service/processor_test.go`
- changed:
  - added an explicit oracle dispatch checkpoint in
    `market_resolutions.evidence.dispatch`
  - oracle worker now distinguishes:
    - `OBSERVED + dispatch=PENDING` -> retry publish
    - `OBSERVED + dispatch=DISPATCHED` -> keep duplicate-emit guard and skip
  - initial successful oracle resolve now writes `OBSERVED`, publishes
    `market.event`, then marks the same observation as `dispatch=DISPATCHED`
  - failed publish after `OBSERVED` now leaves the row retryable as
    `OBSERVED + dispatch=PENDING`
  - settlement now short-circuits duplicate resolved `market.event` handling
    unless the current consumer is the first one to flip the market into
    `RESOLVED`
  - updated the oracle architecture doc and this thread handshake to reflect
    the new runtime contract
- validated:
  - `gofmt -w internal/oracle/service/metadata.go internal/oracle/service/worker.go internal/oracle/service/worker_test.go internal/settlement/service/store.go internal/settlement/service/processor.go internal/settlement/service/sql_store.go internal/settlement/service/processor_test.go`
  - `go test ./internal/oracle/service ./internal/settlement/service`
  - `go test ./cmd/oracle ./internal/account/service`
  - targeted proofs now exist for:
    - publish failure after `OBSERVED` -> row stays retryable with
      `dispatch=PENDING`
    - next poll retries publish without re-observing the provider
    - once `dispatch=DISPATCHED`, later polls skip duplicate `market.event`
    - duplicate resolved `market.event` only lets the first settlement consumer
      continue into payout publish
- blockers:
  - none
- next:
  - commander can review whether the latest-row-only `evidence.dispatch`
    checkpoint is sufficient audit depth, or whether a later task should add an
    append-only dispatch-attempt log
