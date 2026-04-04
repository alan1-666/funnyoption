# WORKLOG-OFFCHAIN-013

### 2026-04-04 21:10 CST

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `docs/architecture/direct-deposit-session-key.md`
  - `web/lib/session-client.ts`
  - `web/components/trading-session-provider.tsx`
  - `internal/shared/auth/session.go`
  - `internal/api/handler/order_handler.go`
  - `internal/api/handler/sql_store.go`
- changed:
  - created a new wallet session UX optimization task, handshake, and worklog
- validated:
  - current baseline already uses wallet-signed session grants plus local
    session-key storage, so this lane can stay focused on restore/login UX
    rather than changing the trust model
- blockers:
  - none yet
- next:
  - launch one worker on `TASK-OFFCHAIN-013`

### 2026-04-04 21:14 CST

- read:
  - `docs/architecture/direct-deposit-session-key.md`
  - user-stated target auth flow for Stark-style one-signature login
- changed:
  - commander moved this task from queued to blocked
- validated:
  - the requested auth direction is no longer a narrow UX improvement:
    - current baseline = wallet-authorized browser-generated session key
    - requested target = MetaMask one-time signature plus browser-local
      Stark-style trading key for later order signing
  - that architecture shift needs its own design lane before this implementation
    task can safely start
- blockers:
  - `TASK-OFFCHAIN-014` must first close the new trading-key auth contract
- next:
  - resume only after the auth architecture is explicit

### 2026-04-04 23:38 CST

- read:
  - `AGENTS.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-OFFCHAIN-013.md`
  - `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-013.md`
  - `docs/harness/worklogs/WORKLOG-OFFCHAIN-013.md`
  - `docs/architecture/direct-deposit-session-key.md`
  - `docs/sql/schema.md`
  - `docs/harness/worklogs/WORKLOG-OFFCHAIN-015.md`
  - `web/lib/session-client.ts`
  - `web/components/trading-session-provider.tsx`
  - `web/components/session-console.tsx`
  - `web/components/site-header.tsx`
  - `internal/shared/auth/session.go`
  - `internal/api/routes_auth.go`
  - `internal/api/handler/order_handler.go`
  - `internal/api/handler/sql_store.go`
- changed:
  - tightened V2 browser restore truth in `web/lib/session-client.ts`:
    - same wallet + chain + vault restore now requires the exact stored key to
      still exist remotely and not be expired
    - expired / revoked / rotated / missing IndexedDB private key now fail with
      explicit status instead of falling through as a fake restore
    - restore now reads wallet history via existing `GET /api/v1/sessions`
      only; new browser registration still stays on trading-key challenge +
      registration routes
  - updated `web/components/trading-session-provider.tsx` so connect /
    prepare-trading / create-session all reuse the same restore pass before any
    wallet signature prompt, while explicit rotate still forces a new auth
  - updated `web/components/session-console.tsx` and
    `web/components/site-header.tsx` to surface restore-in-progress and
    reauthorization-needed states honestly instead of defaulting to “ready”
  - removed the current-key revoke double-write in `session-console` so the
    active key now revokes through the provider path only once
  - updated this handshake to mark the task resumed on top of the landed V2
    baseline and to restate that single-vault-per-environment is still an
    active dependency
- validated:
  - `cd web && npm run build`
  - `node --input-type=module <<'EOF' ... EOF`
    - scripted proof matched the new restore-state decisions for:
      - valid same wallet / chain / vault restore
      - wallet mismatch
      - expired key
      - revoked key
      - rotated key
      - missing IndexedDB private key
- blockers:
  - none for this task slice
- next:
  - commander follow-up can decide whether a later schema/task should add
    durable `vault_address` to the server-side trading-key carrier so restore
    no longer depends on the current single-vault-per-environment assumption
