# PLAN-2026-04-01-master

## Goal

Run FunnyOption with a harness-style operating model and close out the off-chain MVP without depending on long chat history.

## Why now

- the repo is large enough that chat-only memory is fragile
- the project now needs planning threads and execution threads to stay in sync
- off-chain MVP work is moving fast and needs a durable repo memory

## Lanes

1. Harness operating model
2. Off-chain MVP closeout
3. Chain hardening after off-chain stabilization
4. Dedicated admin service and reproducible local lifecycle validation
5. API service hardening and modular routing

## Ordered tasks

| Task ID | Status | Owner | Depends On | Output |
| --- | --- | --- | --- | --- |
| TASK-HARNESS-001 | active | commander/docs | none | harness file system and prompts |
| TASK-OFFCHAIN-001 | active | commander umbrella | none | off-chain MVP lane definition |
| TASK-OFFCHAIN-002 | completed | worker | none | stable local regression path for homepage, detail, matching, settlement, and candles |
| TASK-OFFCHAIN-004 | completed | worker | TASK-OFFCHAIN-002 | resolved-market finality enforcement for order ingress, matching restore, and resting order cleanup |
| TASK-OFFCHAIN-003 | completed | worker | TASK-OFFCHAIN-004 | query/read surface cleanup and operator visibility pass |
| TASK-OFFCHAIN-006 | completed | worker | TASK-OFFCHAIN-003 | honest SSR error surfacing for homepage, detail, and control read paths |
| TASK-OFFCHAIN-005 | completed | worker | TASK-OFFCHAIN-002 | stale-freeze audit runbook and local repair guidance for reused databases |
| TASK-OFFCHAIN-007 | completed | worker/docs | TASK-OFFCHAIN-005 | local stale-freeze cleanup SQL correctness and runbook alignment |
| TASK-OFFCHAIN-008 | completed | worker | TASK-OFFCHAIN-006 | API collection contract normalization so empty list endpoints return `[]` instead of `null` |
| TASK-ADMIN-001 | completed | worker | TASK-OFFCHAIN-008 | dedicated admin route for market operations plus a reproducible local off-chain lifecycle runner |
| TASK-CHAIN-001 | completed | worker | TASK-ADMIN-001 | claim lane hardening so malformed claim payloads cannot queue or submit zero-address on-chain calls |
| TASK-CHAIN-002 | completed | worker | TASK-CHAIN-001 | truthful wallet deposit path so lifecycle proof uses a real listener-driven credit instead of direct processor simulation |
| TASK-ADMIN-002 | completed | worker | TASK-CHAIN-002 | separate admin service with coupled FE/BE allowed, wallet-gated operator access, and explicit operator identity |
| TASK-OFFCHAIN-009 | completed | worker | TASK-ADMIN-002 | explicit first-liquidity path for fresh admin-created markets inside the dedicated admin service so lifecycle proof no longer needs hidden inventory seeding |
| TASK-ADMIN-003 | completed | worker | TASK-OFFCHAIN-009 | converge to one supported dedicated admin runtime and extend wallet-gated operator access to first-liquidity/bootstrap flows |
| TASK-ADMIN-004 | completed | worker | TASK-ADMIN-003 | harden shared core API operator endpoints so create, resolve, and first-liquidity cannot bypass the admin-service wallet gate |
| TASK-API-001 | completed | worker | TASK-ADMIN-004 | apply Gin best practices to the API service with modular route registration, middleware-based auth layering, and rate limiting on sensitive paths |
| TASK-API-002 | completed | worker | TASK-API-001 | remove the transitional bare-`user_id` trade-write path by migrating admin bootstrap order placement onto an authenticated lane and then enforcing session-or-privileged auth on `/api/v1/orders` |
| TASK-API-003 | completed | worker | TASK-API-002 | add replay/idempotency protection to privileged bootstrap orders so operator-signed bootstrap sell orders cannot be replayed within the current proof window |
| TASK-API-004 | completed | worker | TASK-API-003 | define and enforce semantic uniqueness for privileged bootstrap orders so a fresh `requested_at` alone cannot silently authorize a second otherwise-identical bootstrap sell order |
| TASK-OFFCHAIN-010 | paused | worker-validation | TASK-API-004 | rerun the local core business flow and return a pass/fail matrix plus regression evidence after the bootstrap-auth hardening sequence |
| TASK-CHAIN-003 | paused | worker | TASK-CHAIN-002 | reconcile legacy local `chain_deposits` schema drift for reused databases with a safe repair path and validation notes |
| TASK-STAGING-001 | next | worker-validation | TASK-API-004 | run the full staging E2E business flow on `https://funnyoption.xyz/` and `https://admin.funnyoption.xyz/` with evidence and a pass/fail matrix |
| TASK-CICD-001 | blocked | worker-platform | TASK-API-004 | add GitHub push-to-deploy CI/CD for the current server deployment without committing plaintext secrets |

## Risks

- agent threads drift without explicit file ownership
- task context balloons if `AGENTS.md` becomes large again
- chain work may start before off-chain behavior is stable
- reused local databases may still carry legacy `chain_deposits` column widths even though current repo migrations are wider
- the current bootstrap policy intentionally blocks same-terms second bootstrap orders until the repo introduces an explicit operator action handle
- staging E2E may still need at least one funded non-operator user wallet in addition to the funded operator key already available locally
- GitHub CI/CD requires server SSH credentials and deployment commands to be injected through GitHub Secrets, never plaintext repo files

## Decision log

- `AGENTS.md` is now a map, not a handbook
- plans, tasks, handshakes, and worklogs live under `docs/harness/`
- commander plans and routes; workers execute scoped tasks
- `TASK-OFFCHAIN-001` remains the umbrella lane, but worker threads should execute smaller tasks
- next worker should close the local regression path before broader chain hardening
- `TASK-OFFCHAIN-002` is the next worker thread and should return a pass/fail matrix plus reproducible local verification notes
- `TASK-OFFCHAIN-003` is now reserved as the immediate follow-up cleanup pass and should not start until `TASK-OFFCHAIN-002` writes back results
- `TASK-OFFCHAIN-002` surfaced a release blocker: resolved markets are not terminal because new orders can still be accepted and restored resting orders can still match after settlement
- `TASK-OFFCHAIN-004` is now the immediate next worker task and must restore resolved-market finality before read-surface cleanup or chain hardening continue
- stale pre-fix freezes in reused local DBs are a separate residual risk and can be investigated in parallel without touching the `TASK-OFFCHAIN-004` ownership set
- `TASK-OFFCHAIN-004` closed the finality blocker: resolved markets now reject new orders, active resting orders are cancelled on resolve, and cold restart does not rehydrate a tradable book for resolved markets
- `TASK-OFFCHAIN-002` is complete when combined with the `TASK-OFFCHAIN-004` follow-up validation, so `TASK-OFFCHAIN-003` is now the next primary worker lane
- `TASK-OFFCHAIN-003` landed the runtime-backed read surfaces, but review found one remaining truthfulness gap: SSR fetch helpers still collapse API failure into empty datasets, which can make a broken API look like an empty queue or empty market table
- `TASK-OFFCHAIN-005` landed a useful audit helper and runbook, but review found one cleanup-semantic bug: the local cleanup SQL marks released freezes without zeroing `remaining_amount`
- `TASK-OFFCHAIN-006` fixed the frontend truthfulness gap and closed the old follow-up on `TASK-OFFCHAIN-003`, but it also exposed a backend API contract bug: empty collection endpoints still serialize `{"items":null}`, so healthy empty reads can still look unavailable
- `TASK-OFFCHAIN-007` closed the local cleanup semantic gap, so `TASK-OFFCHAIN-005` is now fully complete
- `TASK-OFFCHAIN-008` is now the next worker task and should normalize empty collection responses before chain hardening begins
- `TASK-OFFCHAIN-008` closed the empty-collection contract gap, so the off-chain MVP closeout lane is now complete at the code level
- product scope was reprioritized after off-chain closeout: the next task is a dedicated admin/operator route plus a reproducible local lifecycle path that demonstrates market creation, wallet session authorization, deposit credit, order placement, matching, and settlement without depending on long chat memory
- `TASK-ADMIN-001` is now complete: `/admin` holds the operator surface, `cmd/local-lifecycle` and `docs/operations/local-offchain-lifecycle.md` provide a deterministic local proof, and the current residual truth is documented explicitly
- the local lifecycle proof is honest about two product gaps rather than hiding them:
  - deposit credit is simulated through the confirmed-deposit processor because `.env.local` does not ship a live vault address or listener-ready chain path
  - first opposing inventory is seeded explicitly because newly created markets still lack a native primary issuance / initial-liquidity lane
- `TASK-CHAIN-001` is now complete: malformed claim addresses are rejected at the API boundary, queued invalid claim tasks fail before signing, and the zero-address submission gap is closed for claim payloads
- current product risk is no longer claim input correctness; it is lifecycle truthfulness and operator hardening:
  - the user-visible deposit story is still only partially honest in local/testnet because the default proof bypasses the live listener path
  - fresh admin-created markets still need out-of-band seeded inventory to become tradable
  - the current `/web/admin` surface is functional, but it is only a transitional shell and should not keep growing as the long-term operator backend
- `TASK-CHAIN-002` is the next worker lane because it closes the highest-signal remaining truth gap in the user lifecycle: wallet deposit -> listener -> credited balance
- product direction changed after the initial admin proof: the long-term admin/operator surface should be a separate service, while frontend and backend may stay coupled inside that service
- `TASK-ADMIN-002` now extracts and hardens the operator surface as a dedicated admin service before more admin-only flows land there
- `TASK-OFFCHAIN-009` follows `TASK-ADMIN-002` so explicit first-liquidity lands in the dedicated admin service instead of the transitional public-web admin shell
- `TASK-CHAIN-002` is now complete: the local lifecycle proof uses a real listener-driven deposit observed on an embedded simulated chain, and the repo now has tx/deposit/balance evidence for the funding step
- `TASK-ADMIN-002` is now complete at its original boundary: a dedicated Next-based admin service exists, `scripts/dev-up.sh` starts it, and create/resolve actions are wallet-gated at the admin-service boundary
- `TASK-OFFCHAIN-009` is now complete at its original boundary: hidden lifecycle seeding is replaced by an explicit first-liquidity issuance path and the local lifecycle proof uses that path
- combined review of `TASK-ADMIN-002` and `TASK-OFFCHAIN-009` exposed one new product-level inconsistency:
  - the repo now carries two admin runtime shapes inside `admin/` (the wallet-gated Next service and an ungated Go/template runtime)
  - first-liquidity/bootstrap proof currently lives on the ungated runtime, so the dedicated admin boundary is no longer singular or uniformly wallet-gated
- `TASK-ADMIN-003` is now the next worker lane and should converge the admin surface to one supported runtime before more operator-only flows are added
- low-priority residual ops risk remains from `TASK-CHAIN-002`: older local databases may still enforce legacy `VARCHAR(64)` storage for deposit ids / tx hashes even though repo migrations are wider
- `TASK-ADMIN-003` is now complete: the Go/template runtime is deprecated, the Next admin service is the single supported operator runtime, and first-liquidity now uses the same wallet-gated lane as create and resolve
- commander review still found one deeper privileged-access gap after `TASK-ADMIN-003`:
  - shared backend endpoints such as `POST /api/v1/markets`, `POST /api/v1/markets/:market_id/resolve`, and `POST /api/v1/admin/markets/:market_id/first-liquidity` still accept direct callers without the admin-service signature check
- `TASK-ADMIN-004` is now the next worker lane and should move operator authorization deeper than the admin-service boundary so the shared API cannot be bypassed directly
- the API service itself still carries a structural maintenance gap:
  - routing is centralized in one mixed `RegisterRoutes` block
  - middleware is minimal (`Recovery`, `Logger`, ad hoc CORS)
  - there is no explicit rate-limit layer for sensitive write paths
- `TASK-API-001` is now queued after `TASK-ADMIN-004` so route modularization and broader Gin middleware cleanup can build on the deeper operator-auth boundary instead of conflicting with it
- `TASK-ADMIN-004` is now complete: shared core API operator endpoints re-verify the admin wallet proof and no longer trust direct unauthenticated callers for create, resolve, or first-liquidity
- `TASK-API-001` is now complete at its intended boundary: the API runtime uses modular route registration, explicit middleware layering, and route-group rate limiting for sensitive paths
- combined review of `TASK-ADMIN-004` and `TASK-API-001` exposed one remaining write-path gap:
  - `/api/v1/orders` still allows a transitional bare `user_id` fallback when no session fields are present
  - the dedicated admin bootstrap route still relies on that fallback for the first sell order after first-liquidity issuance
- `TASK-API-002` is now the next worker lane and should migrate bootstrap order placement onto an authenticated path, then remove the transitional bare-`user_id` order-write lane from the shared API
- `TASK-API-002` is now complete: the admin bootstrap route forwards an authenticated operator proof into `/api/v1/orders`, and the shared API no longer accepts bare `user_id` order writes without either session fields or an operator proof envelope
- commander review of `TASK-API-002` found one narrower residual hardening gap:
  - privileged bootstrap orders now use an explicit operator-proof lane, but that lane still relies on a short signature window without session-style nonce or idempotency semantics
  - a replayed bootstrap proof inside that window could still attempt to enqueue duplicate bootstrap sell orders until the underlying position freeze blocks it
- `TASK-API-003` is now the next worker lane and should add explicit replay/idempotency protection to the privileged bootstrap-order path without reopening the old bare-write fallback
- `TASK-API-003` is now complete: replay of the same signed bootstrap payload is blocked by a deterministic bootstrap `order_id`, persisted replay checks, and an in-process keyed gate before the order command is published
- commander review of `TASK-API-003` found one narrower residual product-policy gap:
  - a second operator proof with a new `requested_at` can still intentionally authorize an otherwise-identical bootstrap sell order because the current uniqueness handle is derived from the signed proof itself
  - the repo does not yet state whether that behavior is intended bootstrap policy or an accidental duplicate path
- `TASK-API-004` is now the next worker lane and should define one explicit semantic-uniqueness policy for privileged bootstrap orders, then enforce that policy consistently across the admin service and shared API
- `TASK-API-004` is now complete: same-terms privileged bootstrap sell orders resolve to a deterministic semantic `order_id` derived from `market_id`, `user_id`, `quantity`, `outcome`, and `price`, so re-signing with a fresh `requested_at` is rejected as an already-accepted bootstrap action
- the chosen bootstrap policy is intentionally narrow:
  - `requested_at` is proof freshness only, not a distinct-action handle
  - same-terms second bootstrap actions remain out of contract until a future task introduces an explicit operator action handle
- `TASK-OFFCHAIN-010` is now the next validation worker lane and should rerun the local core business flow after the API bootstrap-auth hardening sequence before larger product work resumes
- `TASK-CHAIN-003` can run in parallel with `TASK-OFFCHAIN-010` because it owns only schema-drift repair docs/migrations for reused local DBs and should stay out of order/session API code
- current priority changed because the local flow has already been tested and the app is deployed on a server:
  - user web: `https://funnyoption.xyz/`
  - admin web: `https://admin.funnyoption.xyz/`
- `TASK-OFFCHAIN-010` and `TASK-CHAIN-003` are paused while staging validation and CI/CD setup take priority
- `TASK-STAGING-001` is now the primary validation worker lane for a full deployed-environment E2E pass from admin market creation and first liquidity through user order matching and settlement
- `TASK-CICD-001` is now the platform worker lane for GitHub push-to-deploy automation; it should keep all private keys and server SSH material in GitHub Secrets or server-only env files, and must not commit `.secrets` or plaintext private keys
- `TASK-CICD-001` has landed the workflow/script/docs implementation, but first live deployment is blocked on external setup outside the repo:
  - GitHub Secrets: `STAGING_SSH_HOST`, `STAGING_SSH_USER`, `STAGING_SSH_PRIVATE_KEY`, `STAGING_DEPLOY_PATH`
  - optional GitHub Secrets: `STAGING_SSH_PORT`, `STAGING_SSH_KNOWN_HOSTS`
  - server-local env file: `deploy/staging/.env.staging`
- commander review found no repo-code blocker in the CI/CD implementation itself; the next deploy action should be to provision those secrets/env values and trigger `staging-deploy`
