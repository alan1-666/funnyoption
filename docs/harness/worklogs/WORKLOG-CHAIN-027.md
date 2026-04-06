# WORKLOG-CHAIN-027

### 2026-04-06 18:09 CST

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
  - `docs/harness/tasks/TASK-CHAIN-026.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-026.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-026.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
  - `internal/rollup/**`
  - `internal/chain/service/**`
  - `internal/shared/config/config.go`
  - `contracts/src/FunnyRollupCore.sol`
- changed:
  - created `TASK-CHAIN-027`, `HANDSHAKE-CHAIN-027`, and this worklog for one
    current-session tranche that turns persisted shadow submissions into a
    restart-safe live onchain submission runtime
- validated:
  - scope is still narrow enough to finish in one current-session tranche
    without reopening the proof/public-signal contract or switching production
    truth
- blockers:
  - none yet
- next:
  - implement the submission state machine, tx tracking, minimal command path,
    and optional chain-service bootstrap

### 2026-04-06 18:33 CST

- thread:
  - commander+worker merged
- changed:
  - extended `rollup_shadow_submissions` runtime state with:
    - `RECORD_SUBMITTED`
    - `ACCEPT_SUBMITTED`
    - `ACCEPTED`
    - `FAILED`
    - durable tx hashes / timestamps / last error fields
  - added `internal/chain/service/rollup_submitter.go` with:
    - `RollupSubmissionProcessor`
    - restart-safe `PollOnce(...)`
    - `RunRollupSubmissionOnce(...)`
    - chain tx send + receipt follow-up over the persisted submission lane
  - extended `cmd/rollup` with `-mode=submit-next`
  - wired optional chain-service bootstrap when `ROLLUP_CORE_ADDRESS` and
    chain operator config are present
  - added `migrations/016_rollup_shadow_submission_runtime.sql`
- validated:
  - `gofmt -w internal/rollup/types.go internal/rollup/store.go internal/shared/config/config.go internal/chain/service/rpc_pool.go internal/chain/service/server.go internal/chain/service/rollup_submitter.go internal/chain/service/rollup_submitter_test.go cmd/rollup/main.go`
  - `go test ./internal/rollup ./internal/chain/service ./cmd/rollup`
  - `forge test --match-path contracts/test/FunnyRollupCore.t.sol`
  - `git diff --check`
  - `psql "$FUNNYOPTION_POSTGRES_DSN" -f migrations/016_rollup_shadow_submission_runtime.sql`
  - verified new `rollup_shadow_submissions` runtime columns exist in local
    Postgres
- blockers:
  - local `.env.local` does not currently define `CHAIN_RPC_URL` or
    `ROLLUP_CORE_ADDRESS`, so this thread could not run a real live broadcast
    against a configured chain target
  - accepted roots still do not switch production truth
- next:
  - if we continue this lane, the next slice should replace the current
    deterministic repo-local proof lane with a broader real state-transition
    proving contract while keeping the accepted submission runtime stable

### 2026-04-06 18:47 CST

- thread:
  - commander+worker merged
- changed:
  - extended local/dev config surfaces for the live submission runtime:
    - `.env.example`
    - `configs/staging/funnyoption.env.example`
    - `configs/test/funnyoption.env.example`
  - extended `scripts/local-chain-up.sh` so local anvil bootstrap now also:
    - computes the rollup genesis root through `cmd/rollup -mode=print-genesis-root`
    - deploys `FunnyRollupCore`
    - deploys `FunnyRollupVerifier`
    - wires `FunnyRollupCore.setVerifier(...)`
    - writes `FUNNYOPTION_ROLLUP_CORE_ADDRESS`,
      `FUNNYOPTION_ROLLUP_VERIFIER_ADDRESS`,
      `FUNNYOPTION_ROLLUP_BATCH_LIMIT`, and `FUNNYOPTION_ROLLUP_POLL_INTERVAL`
      into `.run/dev/local-chain.env`
  - updated `docs/operations/local-persistent-chain.md` so local-chain runtime
    docs now describe the rollup core/verifier bootstrap too
- validated:
  - `go run ./cmd/rollup -mode=print-genesis-root`
  - `bash -n scripts/local-chain-up.sh`
  - `./scripts/local-chain-up.sh`
  - `source .env.local && source .run/dev/local-chain.env && go run ./cmd/rollup -mode=submit-next -timeout=15s`
  - verified `.run/dev/local-chain.env` contains:
    - `FUNNYOPTION_ROLLUP_CORE_ADDRESS`
    - `FUNNYOPTION_ROLLUP_VERIFIER_ADDRESS`
    - `FUNNYOPTION_ROLLUP_BATCH_LIMIT`
    - `FUNNYOPTION_ROLLUP_POLL_INTERVAL`
- blockers:
  - local submitter config is now complete for the no-pending path, but a full
    live onchain submission still requires an actual prepared shadow
    submission row
- next:
  - keep moving the proving lane forward until the repo can generate and submit
    richer state-transition batches without reopening this config/bootstrap
    surface
