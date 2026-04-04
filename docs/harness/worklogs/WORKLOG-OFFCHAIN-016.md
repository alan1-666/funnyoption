# WORKLOG-OFFCHAIN-016

### 2026-04-05 00:31 CST

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `HANDSHAKE-OFFCHAIN-015.md`
  - `WORKLOG-OFFCHAIN-015.md`
  - `HANDSHAKE-OFFCHAIN-013.md`
  - `WORKLOG-OFFCHAIN-013.md`
  - `docs/architecture/direct-deposit-session-key.md`
  - `docs/sql/schema.md`
- changed:
  - created a follow-up task to remove the remaining single-vault-per-env auth
    assumption by making server-side active-key scope durably
    `wallet + chain + vault`
- validated:
  - `TASK-OFFCHAIN-015` and `TASK-OFFCHAIN-013` both closed cleanly, so this
    follow-up can stay narrowly focused on durable scope truth
- blockers:
  - none yet
- next:
  - assign one worker to land the narrow schema/store/readback change

### 2026-04-05 00:49 CST

- read:
  - `AGENTS.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-OFFCHAIN-016.md`
  - `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-016.md`
  - `docs/harness/worklogs/WORKLOG-OFFCHAIN-016.md`
  - `docs/architecture/direct-deposit-session-key.md`
  - `docs/sql/schema.md`
  - `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-015.md`
  - `docs/harness/worklogs/WORKLOG-OFFCHAIN-015.md`
  - `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-013.md`
  - `docs/harness/worklogs/WORKLOG-OFFCHAIN-013.md`
  - `internal/api/handler/sql_store.go`
  - `internal/api/handler/order_handler.go`
  - `internal/api/dto/order.go`
  - `web/lib/session-client.ts`
- changed:
  - added `migrations/012_wallet_sessions_vault_scope.sql` so
    `wallet_sessions` now durably stores `vault_address`
  - updated the SQL store and session DTO/readback contract so registration,
    rotation, `GET /api/v1/sessions`, and restore lookup all scope by
    `wallet + chain + vault`
  - kept `/api/v1/sessions` compatibility tooling in place, but made its
    residual blank-`vault_address` semantics explicit in docs and handshake
  - added one narrow handler regression test plus one PostgreSQL-backed SQL
    store integration test for cross-vault registration / lookup behavior
- validated:
  - `gofmt -w internal/api/dto/order.go internal/api/handler/sql_store.go internal/api/handler/order_handler_test.go internal/api/handler/sql_store_scope_test.go`
  - `GOCACHE=/tmp/funnyoption-gocache go test ./internal/api/handler -run TestListSessionsPassesVaultFilter`
  - `GOCACHE=/tmp/funnyoption-gocache go test ./internal/api/...`
  - `zsh -lc 'source .env.local; GOCACHE=/tmp/funnyoption-gocache go test ./internal/api/handler -run TestSQLStoreRegisterTradingKeyScopesByVault'`
  - `cd web && npm run build`
  - `zsh -lc 'source .env.local; psql "$FUNNYOPTION_POSTGRES_DSN" -v ON_ERROR_STOP=1 -c "BEGIN" -f migrations/012_wallet_sessions_vault_scope.sql -c "ROLLBACK"'`
    - note: the migration file is self-transactional, so this validation became
      a clean local apply rather than a true rollback-only dry-run
- blockers:
  - no code blocker remains for the vault-scoped trading-key slice
  - residual compatibility debt remains explicit:
    - legacy `/api/v1/sessions` create rows still carry blank `vault_address`
      because the old wallet-signed grant never included vault scope
- next:
  - hand back changed files, validation commands, before / after scope
    semantics, and the remaining compatibility tradeoff to commander

### 2026-04-05 01:07 CST

- read:
  - `migrations/003_wallet_sessions_and_deposits.sql`
  - `migrations/012_wallet_sessions_vault_scope.sql`
  - `internal/api/handler/sql_store.go`
  - `internal/api/handler/sql_store_scope_test.go`
  - `docs/sql/schema.md`
  - `docs/architecture/direct-deposit-session-key.md`
  - `HANDSHAKE-OFFCHAIN-016.md`
  - `WORKLOG-OFFCHAIN-016.md`
- changed:
  - added `migrations/013_wallet_sessions_vault_key_uniqueness.sql` to replace
    the legacy `UNIQUE (wallet_address, session_public_key)` rule with durable
    `UNIQUE (wallet_address, chain_id, vault_address, session_public_key)`
  - updated `CreateSession` in `internal/api/handler/sql_store.go` so
    deprecated `/api/v1/sessions` compatibility rows keep explicitly writing
    blank `vault_address`
  - strengthened `internal/api/handler/sql_store_scope_test.go` to prove
    `same wallet + same chain + same public key + two vaults` now persists as
    two active canonical rows, while vault-local rotation still leaves the
    other vault untouched
  - updated schema and architecture docs plus this handshake to describe the
    landed durable uniqueness contract and the remaining blank-vault
    compatibility tradeoff truthfully
- validated:
  - `gofmt -w internal/api/handler/sql_store.go internal/api/handler/sql_store_scope_test.go`
  - `GOCACHE=/tmp/funnyoption-gocache go test ./internal/api/handler -run TestListSessionsPassesVaultFilter -count=1`
  - `zsh -lc 'set -a; source .env.local; set +a; GOCACHE=/tmp/funnyoption-gocache go test ./internal/api/handler -run TestSQLStoreRegisterTradingKeyScopesByVaultEvenWithSamePublicKey -count=1'`
  - `zsh -lc 'set -a; source .env.local; set +a; psql "$FUNNYOPTION_POSTGRES_DSN" -v ON_ERROR_STOP=1 -f migrations/013_wallet_sessions_vault_key_uniqueness.sql'`
  - raw SQL proof on a temp table:
    - old `UNIQUE (wallet_address, session_public_key)` rejects the second
      insert with `duplicate key value violates unique constraint`
    - new
      `UNIQUE (wallet_address, chain_id, vault_address, session_public_key)`
      accepts both inserts and returns `2|2` for `row_count|distinct_vaults`
- blockers:
  - no remaining code blocker for `TASK-OFFCHAIN-016`
  - residual compatibility tradeoff remains explicit:
    - deprecated `/api/v1/sessions` create rows still keep blank
      `vault_address`
- next:
  - hand back changed files, validation commands, before / after proof, and
    the remaining compatibility tradeoff to commander
