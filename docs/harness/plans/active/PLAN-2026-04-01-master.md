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
| TASK-OFFCHAIN-010 | completed | worker-validation | TASK-API-004 | rerun the local core business flow and return a pass/fail matrix plus regression evidence after the bootstrap-auth hardening sequence |
| TASK-CHAIN-003 | completed | worker | TASK-CHAIN-002 | reconcile legacy local `chain_deposits` schema drift for reused databases with a safe repair path and validation notes |
| TASK-STAGING-001 | completed | worker-validation | TASK-API-004 | run the full staging E2E business flow on `https://funnyoption.xyz/` and `https://admin.funnyoption.xyz/` plus a bounded concurrent order/matching script, with evidence and a pass/fail matrix |
| TASK-CHAIN-004 | completed | worker-platform + worker-chain | TASK-STAGING-001 | restore staging deposit ingestion after deploy restarts by fixing stale-start-block/pruned-RPC replay and documenting a restart-safe listener cursor strategy |
| TASK-API-005 | completed | worker | TASK-STAGING-001 | make first-liquidity duplicate handling atomic/idempotent and charge collateral in the same accounting units as settlement payouts |
| TASK-OFFCHAIN-011 | completed | worker | TASK-STAGING-001 | make `/portfolio` render balances, positions, orders, and payouts for the connected session user instead of default user `1001` |
| TASK-OFFCHAIN-012 | completed | worker | TASK-OFFCHAIN-010 | realign `cmd/local-lifecycle` and the local lifecycle docs with the one-shot first-liquidity contract so the local wrapper proof no longer submits a duplicate maker sell |
| TASK-OFFCHAIN-013 | completed | worker | TASK-OFFCHAIN-015 | optimize wallet-signed session login / restore UX after the new V2 trading-key registration path exists; do not implement against the retired session-key baseline |
| TASK-OFFCHAIN-014 | completed | worker-design | TASK-STAGING-001 | define the Stark-style wallet-linked trading-key architecture so users sign once with MetaMask, derive or register one off-chain trading key, and sign subsequent orders without repeated wallet prompts |
| TASK-OFFCHAIN-015 | completed | worker | TASK-OFFCHAIN-014 | implement the first V2 trading-key runtime slice: challenge issuance, `EIP-712` wallet authorization of a browser-local trading key, compatibility storage in `wallet_sessions`, and truthful local restore semantics |
| TASK-OFFCHAIN-016 | completed | worker | TASK-OFFCHAIN-015 | make active trading-key scope durably truthful to `wallet + chain + vault` by persisting vault scope server-side and stopping cross-vault rotation / lookup collapse |
| TASK-OFFCHAIN-018 | completed | worker | TASK-CHAIN-024 | finish the current main product lane by proactively cancelling post-`close_at` active orders on the backend and adding connected-user order/fill visibility plus duplicate-summary cleanup on the market detail page |
| TASK-CICD-001 | completed | worker-platform | TASK-API-004 | add GitHub push-to-deploy CI/CD for the current server deployment without committing plaintext secrets |
| TASK-CICD-002 | completed | worker-platform | TASK-CICD-001 | optimize staging CI/CD so only services affected by a push are validated, rebuilt, and redeployed, while docs-only pushes skip service deployment |
| TASK-CICD-003 | completed | worker-platform | TASK-CICD-002 | make selective deploy self-bootstrap-safe when the server checkout still has an older `scripts/deploy-staging.sh` that does not recognize new workflow-passed flags |
| TASK-CICD-004 | completed | worker-platform | TASK-CICD-003 | simplify staging CI/CD so GitHub Actions becomes a thin trigger that calls one fixed server-side deploy entrypoint, while the server entrypoint fetches the exact target SHA and delegates selective rebuild/restart planning to the repo deploy script |
| TASK-API-006 | paused | worker | TASK-CHAIN-024 | narrow the repo-structure cleanup to `internal/api` by splitting routes/handlers/store concerns into clearer module-owned packages without changing current runtime behavior or widening into a full repo migration |
| TASK-CHAIN-005 | completed | worker-design | TASK-STAGING-001 | define the oracle-settled crypto market contract and first implementation cut so crypto markets can auto-resolve from an external price source with auditable evidence and a safe manual override |
| TASK-CHAIN-006 | completed | worker | TASK-CHAIN-005 | implement the first oracle-settled crypto market runtime slice with one-provider metadata validation, a dedicated oracle worker, manual resolve conflict guards, and truthful resolution-record ownership for manual fallback |
| TASK-CHAIN-007 | completed | worker | TASK-CHAIN-006 | add an explicit retry-safe dispatch contract for oracle observations so `OBSERVED` rows whose publish step failed can be retried without duplicate settlement/account side effects |
| TASK-CHAIN-008 | completed | worker-design | TASK-OFFCHAIN-017, TASK-CHAIN-007 | define the target Mode B architecture as a `ZK-Rollup` exchange with proof-verified state transitions, slow / fast / forced withdrawals, and a migration boundary from the current centralized ledger |
| TASK-CHAIN-009 | completed | worker | TASK-CHAIN-008 | implement the first shadow-rollup tranche with append-only sequencer journal storage, durable batch input, and deterministic shadow-root derivation while keeping current SQL/Kafka settlement as production truth |
| TASK-CHAIN-010 | completed | worker | TASK-CHAIN-009 | extend the shadow-rollup lane into settlement-phase inputs, make the `shadow-batch-v1` witness/public-input contract explicit, and add the smallest L1 batch-metadata surface needed before prover work |
| TASK-CHAIN-011 | completed | worker | TASK-CHAIN-010 | lift API/auth nonce advances into durable shadow batch inputs, replace the `orders_root.nonce_root` zero placeholder with truthful shadow state, and lock the prover-facing public-input lane before verifier-gated acceptance |
| TASK-CHAIN-012 | completed | worker-design | TASK-CHAIN-011 | decide the first proof-lane nonce/auth contract, bind prover/verifier acceptance to the stabilized `shadow-batch-v1` surface, and define the narrow verifier-gated `FunnyRollupCore` acceptance boundary before implementation |
| TASK-CHAIN-013 | completed | worker | TASK-CHAIN-012 | add one narrow canonical V2 trading-key auth witness lane that binds `NONCE_ADVANCED` to verifier-eligible order authorization and migrate repo proof tooling away from deprecated `/api/v1/sessions` before verifier-gated batches |
| TASK-CHAIN-014 | completed | worker | TASK-CHAIN-013 | consume canonical auth witness material in the first verifier-gated auth/proof tranche, keep the public-input shape stable, and prepare `FunnyRollupCore` state-root acceptance without widening into production withdrawal rewrite |
| TASK-CHAIN-015 | completed | worker | TASK-CHAIN-014 | add the smallest Foundry-only verifier/state-root acceptance hook on `FunnyRollupCore` that consumes the stable verifier-gated batch contract and rejects non-`JOINED` auth proof rows without widening into full prover/runtime rewrite |
| TASK-CHAIN-016 | completed | worker | TASK-CHAIN-015 | stabilize the verifier-facing artifact/export boundary and require accepted batches to anchor against previously recorded batch metadata before `FunnyRollupCore` advances accepted state roots |
| TASK-CHAIN-017 | completed | worker | TASK-CHAIN-016 | consume the stable `solidity_export` boundary in the first prover/verifier artifact tranche, replace the current verifier stub with a real verifier-facing interface contract, and prove Go/Solidity verifier-gate digest parity without widening into production withdrawal rewrite |
| TASK-CHAIN-018 | completed | worker | TASK-CHAIN-017 | implement the first real verifier contract boundary that consumes `VerifierArtifactBundle`, recomputes/verifies `verifierGateHash` onchain, and preserves the stable public-input/auth-status contract without widening into production withdrawal rewrite |
| TASK-CHAIN-019 | completed | worker | TASK-CHAIN-018 | stabilize the first proof/public-signal schema on top of `VerifierArtifactBundle`, keep Go/Solidity proof artifact parity explicit, and prepare the path for later real prover output without widening into production withdrawal rewrite |
| TASK-CHAIN-020 | completed | worker | TASK-CHAIN-019 | stabilize the first inner `proofData` schema beneath the fixed outer proof/public-signal envelope so a later prover can emit deterministic verifier-consumable bytes without widening into production withdrawal rewrite |
| TASK-CHAIN-021 | completed | worker-design | TASK-CHAIN-020 | define the first real proof-bytes / proving-system contract under the fixed outer proof/public-signal envelope and `proofData-v1`, including whether a later real prover can stay on `proofData-v1` or needs an explicit `proofData-v2` before cryptographic verification |
| TASK-CHAIN-022 | completed | worker | TASK-CHAIN-021 | implement the first Foundry-only real Groth16 backend under the fixed outer proof/public-signal envelope and `proofData-v1`, including non-empty `proofBytes`, BN254 limb lifting, and Go/Foundry parity fixtures without widening into production withdrawal rewrite |
| TASK-CHAIN-023 | completed | worker | TASK-CHAIN-022 | implement the fixed-vk Groth16 prover artifact pipeline so Go emits batch-specific proof artifacts from actual outer signals instead of one shared fixture while keeping the outer envelope, `proofData-v1`, and production truth unchanged |
| TASK-CHAIN-024 | completed | worker | TASK-CHAIN-007 | harden market-expiry lifecycle semantics so `close_at` stops new trading even if a market row still says `OPEN`, oracle markets continue auto-resolving at `resolve_at`, and non-oracle markets become truthfully closed-awaiting-resolution instead of pretending time alone settles them |
| TASK-CHAIN-025 | completed | worker | TASK-CHAIN-024 | distinguish manual post-close markets from oracle post-close markets with one runtime-effective `WAITING_RESOLUTION` state, and restrict ordinary operator resolve to that adjudication window |

## Risks

- agent threads drift without explicit file ownership
- task context balloons if `AGENTS.md` becomes large again
- chain work may start before off-chain behavior is stable
- reused local databases may still carry legacy `chain_deposits` column widths even though current repo migrations are wider
- the current bootstrap policy intentionally blocks same-terms second bootstrap orders until the repo introduces an explicit operator action handle
- staging E2E may still need at least one funded non-operator user wallet in addition to the funded operator key already available locally
- GitHub CI/CD requires server SSH credentials and deployment commands to be injected through GitHub Secrets, never plaintext repo files
- staging chain deposits can stall after a deploy restart if the chain service replays from a static start block that is already pruned by the configured public RPC
- oracle-market work can accidentally fork the Solidity toolchain unless the repo stays explicit that contracts remain Foundry-based
- the V2 auth design temporarily reuses `wallet_sessions` and `user_profiles.wallet_address`, so cross-chain or cross-vault wallet binding must stay explicit in the follow-up implementation slice instead of being assumed implicitly
- removing `POST /api/v1/sessions` before the repo's lifecycle / concurrency proof tooling migrates would still break internal verification flows; the route now remains as an explicit deprecated compat path until those tools move
- oracle dispatch retry is now guarded by a latest-row checkpoint in `market_resolutions.evidence.dispatch`, but it is still not an append-only dispatch-attempt log or full outbox
- deprecated `/api/v1/sessions` compatibility rows still intentionally keep blank `vault_address`, so future auth cleanup must migrate proof tooling before retiring that legacy carrier
- the Mode B lane now explicitly prefers `ZK-Rollup` data availability for the strongest exit guarantees, which materially raises L1 calldata / state-diff cost and contract-surface complexity relative to the current BSC-vault product
- the Mode B lane now explicitly requires three withdrawal paths:
  - slow batch-confirmed withdrawal
  - fast LP-backed withdrawal
  - forced withdrawal / freeze / escape hatch
- the first implementation tranche after the Mode B design is intentionally shadow-only:
  - append-only sequencer journal
  - durable batch input
  - deterministic shadow roots
  - no premature prover / verifier / production claim rewrite
- the first shadow-rollup slice is now complete, but one residual replay gap is explicit:
  - `orders_root` still uses deterministic `ZeroNonceRoot()` until a later slice either shadows nonce truthfully or freezes that limitation in the witness contract

## Decision log

- `AGENTS.md` is now a map, not a handbook
- plans, tasks, handshakes, and worklogs live under `docs/harness/`
- commander plans and routes; workers execute scoped tasks
- Solidity contract work stays on the repo's existing Foundry layout (`foundry.toml`, `contracts/src`, `contracts/test`, `contracts/script`); do not introduce a second contract framework for the oracle lane
- `TASK-OFFCHAIN-014` is complete: V2 auth rejects signature-derived deterministic trading keys and adopts wallet-authorized locally generated trading keys plus durable wallet binding semantics
- `TASK-OFFCHAIN-015` is the first auth implementation slice and should land challenge issuance plus `EIP-712` trading-key registration before older UX-only follow-ups resume
- `TASK-CHAIN-005` is complete: oracle-settled crypto markets now have an explicit metadata / evidence / resolver contract with a dedicated oracle worker boundary
- `TASK-CHAIN-006` is the first oracle implementation slice and must include the manual resolve conflict guard plus truthful overwrite of `market_resolutions` ownership fields when an operator fallback wins after oracle error states
- `TASK-OFFCHAIN-015` is complete: the V2 trading-key runtime restores `POST /api/v1/sessions` as a deprecated proof-tool compat route while the canonical browser flow stays on challenge + `EIP-712` registration
- `TASK-CHAIN-006` is complete: the oracle worker now skips duplicate resolved-event publish when the same oracle observation is already recorded as `OBSERVED`, so repeated polling no longer re-triggers settlement/account side effects from this worker path
- `TASK-OFFCHAIN-016` is complete: canonical trading-key rows now durably scope uniqueness, rotation, and readback to `wallet + chain + vault`, including allowing one wallet to reuse the same trading public key across two different vaults on one chain
- `TASK-CHAIN-007` is complete: oracle `OBSERVED` rows now persist a dispatch
  checkpoint in `market_resolutions.evidence.dispatch`, and settlement only
  lets the first resolved event continue into cancel/payout publish
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
- `TASK-STAGING-001` is now the primary validation worker lane for a full deployed-environment E2E pass from admin market creation and first liquidity through user order matching and settlement, plus a bounded concurrent order-placement/matching script that can surface duplicate-fill, overfill, negative-balance, or stale-freeze regressions under parallel writes
- `TASK-CICD-001` is now the platform worker lane for GitHub push-to-deploy automation; it should keep all private keys and server SSH material in GitHub Secrets or server-only env files, and must not commit `.secrets` or plaintext private keys
- `TASK-CICD-001` has landed the workflow/script/docs implementation, but first live deployment is blocked on external setup outside the repo:
  - GitHub Secrets: `STAGING_SSH_HOST`, `STAGING_SSH_USER`, `STAGING_SSH_PRIVATE_KEY`, `STAGING_DEPLOY_PATH`
  - optional GitHub Secrets: `STAGING_SSH_PORT`, `STAGING_SSH_KNOWN_HOSTS`
  - server-local env file: `deploy/staging/.env.staging`
- commander review found no repo-code blocker in the CI/CD implementation itself; the next deploy action should be to provision those secrets/env values and trigger `staging-deploy`
- `TASK-CICD-001` is now complete: first staging GitHub Actions deploy has run successfully after external secrets/env were configured and the server checkout was bootstrapped
- commander review found one performance gap in the current CI/CD shape:
  - workflow validation always runs all Go tests and both frontend builds
  - remote deploy always runs `docker compose up -d --build --remove-orphans` for the whole stack
  - because Go service Dockerfiles use `COPY . .`, docs/script-only changes can still force unnecessary image rebuild work
- `TASK-CICD-002` is now the next platform worker lane and should introduce one explicit path-to-service change map plus a safe fallback policy for broad/shared changes
- commander review of `TASK-CICD-002` found one rollout blocker in the remote invocation path:
  - `.github/workflows/staging-deploy.yml` calls `bash ./scripts/deploy-staging.sh --service ...` inside the server's current checkout
  - argument parsing in `scripts/deploy-staging.sh` happens before `sync_release_ref`
  - if a push introduces new deploy-script flags while the server checkout still has an older script, the old parser can fail on those flags before it fetches the new ref
- `TASK-CICD-003` should fix that self-bootstrap order problem without giving up the selective-deploy behavior from `TASK-CICD-002`
- `TASK-CICD-003` is now complete: the workflow checks the server checkout for local tracked/staged edits, runs `git fetch --prune origin`, checks out the target ref, and only then invokes the checked-out deploy script with `--skip-git-sync` plus selected service flags
- commander review validated `bash -n scripts/deploy-staging.sh`, YAML parsing for `.github/workflows/staging-deploy.yml`, `git diff --check`, and one `--print-plan --diff-base HEAD~1` dry-run that produced `skip_deploy=1` for a workflow-only change
- `TASK-STAGING-001` now has a checked-in bounded concurrency script and a second staging evidence pass, but commander review found four release blockers:
  - chain deposit ingestion is broken on staging after the chain service restarts because `internal/chain/service/listener.go` reinitializes `nextBlock` from stale `FUNNYOPTION_CHAIN_START_BLOCK=99452107`, while the current public RPC returns `History has been pruned for this block`; the latest blocked test deposit tx was `0x4129a4db5f66760ca8374a1dbe3df94652552df9768500ff0d49ec9654733a6c` at block `99674293`
  - `admin/app/api/operator/markets/[marketId]/first-liquidity/route.ts` still performs first-liquidity issuance before the duplicate bootstrap order write, so a second same-terms call can return `409` after mutating maker inventory/balance
  - `internal/api/handler/order_handler.go` still debits `req.Quantity` and returns `collateral_debit=req.Quantity` for one YES/NO pair, which under-collateralizes payouts that settle at `100` accounting units per winning share
  - `web/app/portfolio/page.tsx` still server-fetches collection reads without the connected session user, and `web/lib/api.ts` defaults those reads to `user_id=1001`
- `TASK-CHAIN-004`, `TASK-API-005`, and `TASK-OFFCHAIN-011` are the next parallel worker lanes; rerun `TASK-STAGING-001` only after those fixes land and the chain deposit listener is healthy again
- `TASK-CHAIN-004` is now complete: the listener cursor implementation works, `go test ./internal/chain/service/...` passes, one fresh post-restart staging deposit reached `/api/v1/deposits` plus `/api/v1/balances`, `/opt/funnyoption-staging` is a clean detached checkout at `ea71dc8`, and the Actions dirty-check guard passes there
- one non-blocking docs follow-up remains after `TASK-CHAIN-004`: the DSN-based recovery snippet in `docs/deploy/staging-bsc-testnet.md` still uses `source deploy/staging/.env.staging`, and the current server env file emits `Testnet: command not found` for an unquoted value with spaces even though the subsequent `psql "$FUNNYOPTION_POSTGRES_DSN"` probe succeeds; prefer a shell-safe one-variable loader before relying on that fallback path
- `TASK-STAGING-001` is now complete across the combined staging evidence:
  - chain-side validation is healthy on staging: fresh generated users received deposits, the bounded script submitted `8/8` orders successfully, matched `4` trades, resolved the market, and found no duplicate-fill / overfill / negative-balance / stale-freeze anomalies
  - deployment verification on committed/staged `HEAD=125f9cd` confirmed the previously failing API/web checks are now green: duplicate bootstrap `409` is side-effect free, first-liquidity collateral debits `100 * quantity`, and `/portfolio` no-session plus connected-session reads are truthful and scoped to the active `session.userId`
  - with staging E2E and CI/CD both validated, the paused local follow-up lanes `TASK-OFFCHAIN-010` and `TASK-CHAIN-003` can resume in parallel
- `TASK-OFFCHAIN-010` is now complete as a validation lane:
  - local runtime parity with staging is confirmed for listener-driven deposit credit, duplicate bootstrap behavior, session-backed order placement, resolution, and terminal read surfaces
  - the only failure left is the local proof wrapper itself: `cmd/local-lifecycle` still places a second explicit maker `SELL` after `/api/v1/admin/markets/:market_id/first-liquidity` already queued the bootstrap order
- `TASK-CHAIN-003` is now complete:
  - repo truth and legacy drift for `chain_deposits` widths are documented explicitly
  - `migrations/010_chain_deposits_tx_hash_width_repair.sql` plus `docs/operations/local-chain-deposits-schema-repair.md` provide the narrow repair path
  - worker validated synthetic drift reproduction, rollback-safe dry run, real apply, idempotent re-apply, and `go test ./internal/chain/...`
- `TASK-OFFCHAIN-012` is the next narrow local follow-up:
  - it should remove the stale second-sell step from `cmd/local-lifecycle` and align the local lifecycle docs with the one-shot first-liquidity contract already validated in staging and direct local API/runtime checks
- `TASK-OFFCHAIN-012` is now complete:
  - `cmd/local-lifecycle` no longer submits the stale second maker `SELL`
  - `./scripts/local-lifecycle.sh` is green again
  - local lifecycle docs now describe first-liquidity as one-shot inventory issuance plus queued bootstrap `SELL`, followed by the crossing `BUY`
  - a small non-blocking local-state caveat remains documented: persistent `anvil` plus reused local postgres can reuse deterministic deposit evidence across runs unless the local DB is reset
- current deployment behavior works, but the operator experience is still heavier than needed because `.github/workflows/staging-deploy.yml` owns too much orchestration logic instead of acting as a thin authenticated trigger into one stable server-side entrypoint
- `TASK-CICD-004` is now the next optional platform lane:
  - GitHub Actions should shrink to SSH trigger plus audit trail
  - the server should own one fixed deploy entrypoint with locking, clean-checkout guard, exact-SHA checkout, and the call into repo `scripts/deploy-staging.sh`
  - selective rebuild/restart behavior and docs-only no-op deploys should stay intact
- commander review of `TASK-CICD-004` found one correctness gap before closure:
  - `deploy/staging/server-deploy-entrypoint.sh` resolves `${target_ref}^{commit}` and plain `${branch_ref}^{commit}` before the freshly fetched `origin/<ref>`
  - for symbolic deploy refs like `main`, that can silently choose a stale local branch left behind in `/opt/funnyoption-staging` instead of the current remote branch tip
  - the follow-up should keep exact-SHA deploys unchanged but make symbolic refs prefer remote-tracking refs after fetch
- `TASK-CICD-004` is now complete:
  - raw commit SHAs still deploy exactly as supplied
  - symbolic branch refs like `main` and `refs/heads/main` now prefer the freshly fetched remote-tracking ref before any same-named local-branch fallback
  - thin-trigger workflow, host lock, dirty-checkout guard, and selective/docs-only deploy behavior all remain intact
- next product-development priorities are now:
  - `TASK-OFFCHAIN-014` for Stark-style trading-key auth design
  - `TASK-CHAIN-005` for a design-first oracle auto-resolution contract on crypto markets
- the user has now changed the auth target beyond a UX tweak:
  - desired first-login flow is MetaMask connect + one off-chain signature, then a browser-local non-EVM trading key for subsequent order signing
  - desired deposits remain direct on-chain vault deposits with operator event listening and off-chain account credit
  - that conflicts with the current V1 doc, which explicitly says not to derive a trading private key from the wallet signature and not to implement a StarkEx-style auth flow
- `TASK-OFFCHAIN-014` is therefore the next auth lane and should stay design-first:
  - define whether the product truly derives a deterministic Stark/private trading key from the wallet signature, or instead wallet-authorizes a locally generated Stark key
  - define the exact signing message/domain, key registration flow, nonce/replay model, browser storage, revocation, and migration path from the current ed25519-style session model
- `TASK-OFFCHAIN-013` is complete: frontend restore now reconciles exact local key truth before any reauthorization prompt, expired / revoked / rotated / missing-local-key states fail honestly, and new browser registration still stays on the canonical trading-key routes
- deployed staging wallet/portfolio polish is now complete:
  - refresh with one unambiguous local trading key restores without reopening
    the wallet-provider chooser on mount; user-triggered trading actions still
    force a real wallet reconnect before signing
  - collateral balance reads now fall back to a larger `balances` page when
    `USDT` is paged out by `POSITION:*` rows, so position-heavy users no longer
    look empty-funded in the main wallet summary
  - wallet-address copy now has a visible success affordance, and the QR
    dialog opens higher with a more natural viewport center
- `TASK-OFFCHAIN-017` is complete; remaining tradeoffs stay narrow:
  - multiple local trading keys for the same `chain + vault` still require an
    explicit wallet reconnect instead of guessing
  - the frontend balance fallback currently tops out at `limit=200`; backend
    asset prioritization would still be cleaner if one user eventually exceeds
    that many `POSITION:*` rows
- `TASK-CHAIN-005` should stay design-first before runtime implementation:
  - define the metadata contract for oracle-settled crypto markets
  - define where oracle fetch / evidence persistence / auto-resolution lives
  - preserve the current admin manual resolve path as the fallback and override lane
- `TASK-CHAIN-008` is now the architecture lane for the user's target Mode B
  exchange:
  - design first, no premature prover or verifier implementation
  - `ZK-Rollup` DA only in the target contract
  - withdrawals must be modeled explicitly as `slow`, `fast`, and `forced`
  - the lane must state which current FunnyOption truths can remain
    operator-run and which must be replaced before the product can honestly
    claim proof-verified settlement
- `TASK-CHAIN-008` is now complete:
  - canonical design doc: `docs/architecture/mode-b-zk-rollup.md`
  - current FunnyOption is explicitly documented as not yet Mode B
  - the recommended first implementation tranche is `shadow journal + durable batch input + deterministic shadow roots`
- `TASK-CHAIN-009` is now the next implementation lane:
  - make the replay contract and shadow-root artifacts real before any proof-system work
  - keep current SQL/Kafka settlement as production truth during this slice
- `TASK-CHAIN-009` is now complete:
  - append-only shadow journal storage exists
  - durable batch input exists
  - deterministic shadow roots exist for the trading phase
  - production settlement truth remains unchanged
- `TASK-CHAIN-010` is now the next follow-up lane:
  - extend shadow replay into market-resolution / settlement-payout inputs
  - make `shadow-batch-v1` witness/public-input explicit
  - define the smallest L1 batch-metadata contract surface before prover work
- `TASK-API-005` is now complete at code/test level: duplicate same-terms bootstrap requests are rejected in the one-shot core first-liquidity handler before maker mutation, first-liquidity collateral debit uses `assets.WinningPayoutAmount(req.Quantity)`, the admin route no longer submits a second bootstrap `/api/v1/orders` call, and commander re-ran `go test ./internal/api/...` plus `admin && npm run build`
- one runtime validation gap remains for `TASK-API-005`: the worker could not run a full local lifecycle replay because the dev stack was down, so the next `TASK-STAGING-001` rerun should explicitly recheck duplicate bootstrap side effects and maker collateral debit on the deployed environment
- `TASK-OFFCHAIN-011` is now complete at code/test level: `/portfolio` SSR no longer fetches private collections with default user `1001`, `PortfolioShell` waits for `session.userId` then refreshes balances/positions/orders/payouts/profile for that user, disconnected/not-authorized states are explicit, and commander re-ran `web && npm run build`
- one runtime validation gap remains for `TASK-OFFCHAIN-011`: the worker's browser proof used a local mock API plus injected localStorage session, so the next `TASK-STAGING-001` rerun should explicitly recheck `/portfolio` against a real generated staging session wallet
