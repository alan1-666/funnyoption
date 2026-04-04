# WORKLOG-OFFCHAIN-015

### 2026-04-04 21:52 CST

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `docs/architecture/direct-deposit-session-key.md`
  - `docs/sql/schema.md`
  - `HANDSHAKE-OFFCHAIN-014.md`
  - `WORKLOG-OFFCHAIN-014.md`
- changed:
  - created the first V2 auth implementation task, handshake, and worklog
- validated:
  - the auth design is now explicit enough to start a narrow runtime slice
    without reopening the trust-model discussion
  - the next safe cut is challenge issuance plus `EIP-712` trading-key
    registration, not a broad end-to-end order payload rewrite
- blockers:
  - none yet
- next:
  - launch one worker on `TASK-OFFCHAIN-015`

### 2026-04-04 21:56 CST

- read:
  - `AGENTS.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-OFFCHAIN-015.md`
  - `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-015.md`
  - `docs/harness/worklogs/WORKLOG-OFFCHAIN-015.md`
  - `docs/architecture/direct-deposit-session-key.md`
  - `docs/sql/schema.md`
  - `web/lib/session-client.ts`
  - `web/components/trading-session-provider.tsx`
  - `internal/shared/auth/session.go`
  - `internal/api/handler/order_handler.go`
  - `internal/api/handler/sql_store.go`
  - `internal/chain/service/sql_store.go`
  - supporting DTO, route, test, config, migration, and listener files required
    to wire the first V2 runtime slice safely
- changed:
  - expanded handshake scope to include the narrow DTO / route / server / test /
    migration files this slice needs
- validated:
  - the narrowest truthful runtime cut still needs one dedicated challenge
    storage layer plus one active-key status readback surface
  - `wallet_sessions` can remain the compatibility carrier for trading keys and
    order nonces even if challenge storage is separate
- blockers:
  - none
- next:
  - implement challenge issuance, EIP-712 registration verification, truthful
    browser restore, and durable wallet-binding lookup

### 2026-04-04 22:54 CST

- changed:
  - implemented `POST /api/v1/trading-keys/challenge` and
    `POST /api/v1/trading-keys` with one-time SQL-backed challenge storage,
    `EIP-712` wallet-signature verification, and compatibility persistence into
    `wallet_sessions`
  - kept the order write path narrow by continuing to use existing
    `session_*` carrier fields while changing runtime truth and user-facing copy
    to trading-key language
  - moved browser persistence to local metadata plus IndexedDB private-key
    storage, then made restore truthful against wallet / chain / vault scope and
    server-side active-key readback
  - switched deposit / withdrawal wallet attribution lookup to
    `user_profiles.wallet_address` so custody semantics no longer depend on an
    active trading key row
- validated:
  - `gofmt -w internal/shared/auth/session.go internal/shared/auth/session_test.go internal/api/dto/order.go internal/api/handler/order_handler.go internal/api/handler/sql_store.go internal/api/handler/order_handler_test.go internal/api/routes_auth.go internal/api/server.go internal/api/router_test.go internal/chain/service/sql_store.go internal/chain/service/listener.go internal/api/middleware.go`
  - `go test ./internal/shared/auth`
  - `go test ./internal/api/...`
  - `go test ./internal/chain/service`
  - `cd web && npm run build`
  - temporary local Playwright / Node proof run validated:
    - fresh registration stores metadata locally and private key in IndexedDB
      without re-deriving from the wallet signature
    - refresh restore reuses the same active key without issuing a second
      challenge or registration
    - wallet mismatch and chain mismatch both degrade to auth-required state
      without falsely restoring the old key
    - rotate / revoke clears stale local key material and requires reauth
- blockers:
  - none
- next:
  - hand back changed files, validation commands, before / after behavior, and
    residual tradeoffs to commander

### 2026-04-05 00:10 CST

- read:
  - `internal/api/routes_auth.go`
  - `cmd/local-lifecycle/main.go`
  - `scripts/staging-concurrency-orders.mjs`
  - `internal/api/handler/sql_store.go`
- changed:
  - commander marked `TASK-OFFCHAIN-015` back to blocked after review
- validated:
  - the V2 auth runtime slice itself is coherent and its claimed tests/builds
    pass
  - but `POST /api/v1/sessions` was removed from the API router while existing
    repo proof tooling still calls it:
    - `cmd/local-lifecycle/main.go`
    - `scripts/staging-concurrency-orders.mjs`
  - the current compatibility storage still does not durably carry
    `vault_address`, so active-key rotation is effectively scoped to
    `wallet + chain`, not the fuller `wallet + chain + vault` contract
- blockers:
  - restore an explicit compatibility story for the old `/api/v1/sessions`
    callers or migrate the repo proof tooling in the same task
- next:
  - continue the same worker on `TASK-OFFCHAIN-015` to close the compatibility
    gap before marking the slice complete

### 2026-04-04 23:11 CST

- read:
  - `AGENTS.md`
  - `PLAN.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-OFFCHAIN-015.md`
  - `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-015.md`
  - `docs/harness/worklogs/WORKLOG-OFFCHAIN-015.md`
  - `internal/api/routes_auth.go`
  - `cmd/local-lifecycle/main.go`
  - `scripts/staging-concurrency-orders.mjs`
  - `docs/architecture/direct-deposit-session-key.md`
  - `docs/sql/schema.md`
  - `internal/api/handler/sql_store.go`
  - `web/lib/session-client.ts`
- changed:
  - restored `POST /api/v1/sessions` in the API router as a deprecated
    compatibility route for repo proof tooling, without changing the V2 browser
    trading-key flow
  - added a router-level regression test proving the legacy session-create path
    is reachable again through the public engine
  - documented the current boundary that durable active-key rotation / listing
    is still effectively `wallet + chain` because `wallet_sessions` has no
    durable `vault_address`
  - updated handshake status and handoff notes to reflect the compat route plus
    the remaining single-vault assumption
- validated:
  - `gofmt -w internal/api/routes_auth.go internal/api/router_test.go`
  - `go test ./internal/api/... ./cmd/local-lifecycle`
  - `node --check scripts/staging-concurrency-orders.mjs`
- blockers:
  - no remaining route-regression blocker
  - follow-up risk remains documented:
    - durable active-key scope is not yet fully `wallet + chain + vault`
- next:
  - hand back the compat-route fix, validation commands, and the remaining
    scope tradeoff to commander
