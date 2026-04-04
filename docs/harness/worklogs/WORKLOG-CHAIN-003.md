# WORKLOG-CHAIN-003

### 2026-04-03 20:20 Asia/Shanghai

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `HANDSHAKE-CHAIN-002.md`
  - `WORKLOG-CHAIN-002.md`
  - `migrations/003_wallet_sessions_and_deposits.sql`
- changed:
  - created a narrow chain schema-drift follow-up task for reused local `chain_deposits` tables
- validated:
  - task, handshake, and acceptance criteria are in repo files
  - this worker can run in parallel with `TASK-OFFCHAIN-010` because it should stay out of order/session API code and does not own the regression worklog
- blockers:
  - none yet; worker may still need to prove whether an old local DB is available for a repair dry run
- next:
  - launch a worker against `TASK-CHAIN-003`

### 2026-04-03 20:35 Asia/Shanghai

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `HANDSHAKE-CHAIN-003.md`
- changed:
  - paused this local schema-drift cleanup task behind `TASK-STAGING-001` and `TASK-CICD-001`
- validated:
  - active plan and handshake status now match
- blockers:
  - none
- next:
  - resume after staging E2E and CI/CD setup are no longer the top priority

### 2026-04-04 19:32 Asia/Shanghai

- read:
  - `AGENTS.md`
  - `PLAN.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-CHAIN-003.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-003.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-003.md`
  - `docs/architecture/direct-deposit-session-key.md`
  - `docs/sql/schema.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-002.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-004.md`
  - `migrations/003_wallet_sessions_and_deposits.sql`
  - `internal/chain/service/listener.go`
  - `internal/chain/service/processor.go`
  - `internal/chain/service/sql_store.go`
  - `docs/operations/local-persistent-chain.md`
- changed:
  - added `migrations/010_chain_deposits_tx_hash_width_repair.sql` as a narrow repair migration that only widens `chain_deposits.tx_hash` back to repo truth `VARCHAR(128)`
  - added `docs/operations/local-chain-deposits-schema-repair.md` with local-only preflight, dry-run, apply, verify, and rollback-safe boundary commands
  - updated `docs/sql/schema.md` to state the expected `chain_deposits` widths, the observed legacy local drift, and the repair path
  - updated `docs/operations/local-persistent-chain.md` to point deposit-debugging operators at the schema-repair runbook instead of suggesting code changes
- validated:
  - repo truth from DDL is now explicit:
    - `chain_deposits.deposit_id = VARCHAR(64)`
    - `chain_deposits.tx_hash = VARCHAR(128)`
  - the legacy reused-local drift observed by `TASK-CHAIN-002` is now documented as:
    - `chain_deposits.deposit_id = VARCHAR(64)`
    - `chain_deposits.tx_hash = VARCHAR(64)`
  - direct inspection in this workspace could not read a pre-existing reused local DB volume:
    - `.env.local` pointed to a Homebrew PostgreSQL on `127.0.0.1:5432`
    - no preserved legacy Docker volume was present before this task started
  - synthetic validation against a fresh local Docker PostgreSQL reproduced the drift and repair path safely:
    - baseline migrations `001`-`009` produced `deposit_id=64`, `tx_hash=128`
    - manual drift simulation changed `tx_hash` to `64`
    - transaction-scoped dry run widened `tx_hash` to `128` and `ROLLBACK` returned it to `64`
    - applying `migrations/010_chain_deposits_tx_hash_width_repair.sql` widened `tx_hash` back to `128` without touching sample rows
    - re-applying `migrations/010_chain_deposits_tx_hash_width_repair.sql` stayed idempotent
    - rollback-safe shrink back to `VARCHAR(64)` succeeded only after validating `MAX(length(tx_hash)) <= 64`
    - running all migrations `001`-`010` on a fresh database still ended at `deposit_id=64`, `tx_hash=128`
  - listener-driven local deposit proof boundary remains intact because this task did not change runtime chain code and the documented repair keeps accepting the normalized 64-character tx hashes already emitted by the listener
  - `go test ./internal/chain/...`
  - `git diff --check`
- blockers:
  - no blocking code or migration issue remains for the documented local repair path
  - residual evidence boundary: this workspace did not contain a preserved legacy reused local DB to inspect directly, so the drift shape still rests on `TASK-CHAIN-002`'s recorded observation plus the synthetic reproduction above
- next:
  - hand back the documented schema truth, local-only repair path, and validation results to commander
