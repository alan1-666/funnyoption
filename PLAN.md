# FunnyOption Master Plan

This file is the current top-level map for commander threads.
Detailed execution lives in `docs/harness/plans/active/`.

## Current source-of-truth files

- Active orchestration plan: [`docs/harness/plans/active/PLAN-2026-04-01-master.md`](/Users/zhangza/code/funnyoption/docs/harness/plans/active/PLAN-2026-04-01-master.md)
- Harness rollout task: [`docs/harness/tasks/TASK-HARNESS-001.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-HARNESS-001.md)
- Off-chain umbrella task: [`docs/harness/tasks/TASK-OFFCHAIN-001.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-001.md)
- Staging E2E task: [`docs/harness/tasks/TASK-STAGING-001.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-STAGING-001.md)
- Staging chain-listener unblock task: [`docs/harness/tasks/TASK-CHAIN-004.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-004.md)
- First-liquidity correctness task: [`docs/harness/tasks/TASK-API-005.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-API-005.md)
- Portfolio connected-user read task: [`docs/harness/tasks/TASK-OFFCHAIN-011.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-011.md)
- Local lifecycle wrapper alignment task: [`docs/harness/tasks/TASK-OFFCHAIN-012.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-012.md)
- Wallet session UX optimization task: [`docs/harness/tasks/TASK-OFFCHAIN-013.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-013.md)
- Stark-style trading key auth design task: [`docs/harness/tasks/TASK-OFFCHAIN-014.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-014.md)
- Trading-key registration first-slice implementation task: [`docs/harness/tasks/TASK-OFFCHAIN-015.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-015.md)
- Vault-scoped trading-key durability follow-up task: [`docs/harness/tasks/TASK-OFFCHAIN-016.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-016.md)
- Market detail order visibility and lifecycle closeout task: [`docs/harness/tasks/TASK-OFFCHAIN-018.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-018.md)
- GitHub CI/CD optimization task: [`docs/harness/tasks/TASK-CICD-003.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CICD-003.md)
- Thin-trigger CI/CD simplification task: [`docs/harness/tasks/TASK-CICD-004.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CICD-004.md)
- API module-boundary cleanup task: [`docs/harness/tasks/TASK-API-006.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-API-006.md)
- Oracle-settled crypto market design task: [`docs/harness/tasks/TASK-CHAIN-005.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-005.md)
- Oracle-settled crypto market first-slice implementation task: [`docs/harness/tasks/TASK-CHAIN-006.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-006.md)
- Oracle dispatch retry follow-up task: [`docs/harness/tasks/TASK-CHAIN-007.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-007.md)
- Mode B rollup architecture design task: [`docs/harness/tasks/TASK-CHAIN-008.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-008.md)
- Mode B shadow-rollup first-slice task: [`docs/harness/tasks/TASK-CHAIN-009.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-009.md)
- Mode B shadow-rollup settlement-phase follow-up task: [`docs/harness/tasks/TASK-CHAIN-010.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-010.md)
- Mode B shadow-rollup nonce/public-input follow-up task: [`docs/harness/tasks/TASK-CHAIN-011.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-011.md)
- Mode B proof-lane nonce/verifier design task: [`docs/harness/tasks/TASK-CHAIN-012.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-012.md)
- Mode B canonical auth-witness tranche task: [`docs/harness/tasks/TASK-CHAIN-013.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-013.md)
- Mode B verifier-gated auth/proof tranche task: [`docs/harness/tasks/TASK-CHAIN-014.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-014.md)
- Mode B minimal verifier/state-root acceptance tranche task: [`docs/harness/tasks/TASK-CHAIN-015.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-015.md)
- Mode B verifier artifact / metadata-anchored acceptance follow-up task: [`docs/harness/tasks/TASK-CHAIN-016.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-016.md)
- Mode B first prover/verifier artifact tranche task: [`docs/harness/tasks/TASK-CHAIN-017.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-017.md)
- Mode B first verifier implementation tranche task: [`docs/harness/tasks/TASK-CHAIN-018.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-018.md)
- Mode B proof/public-signal schema tranche task: [`docs/harness/tasks/TASK-CHAIN-019.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-019.md)
- Mode B inner proof-data schema tranche task: [`docs/harness/tasks/TASK-CHAIN-020.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-020.md)
- Mode B real proof-bytes / proving-system contract design task: [`docs/harness/tasks/TASK-CHAIN-021.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-021.md)
- Mode B first real Groth16 backend tranche task: [`docs/harness/tasks/TASK-CHAIN-022.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-022.md)
- Mode B fixed-vk Groth16 prover artifact tranche task: [`docs/harness/tasks/TASK-CHAIN-023.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-023.md)
- Market expiry and resolution lifecycle hardening task: [`docs/harness/tasks/TASK-CHAIN-024.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-024.md)
- Manual-vs-oracle post-close lifecycle task: [`docs/harness/tasks/TASK-CHAIN-025.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-025.md)

## Strategic lanes

1. Off-chain MVP closeout
   - stabilize local dev
   - finish off-chain trade lifecycle
   - finish query/read surfaces
   - tighten websocket market surfaces

2. Harness operating model
   - slim `AGENTS.md`
   - formalize plans, tasks, handshakes, worklogs
   - separate commander and worker threads

3. Chain integration hardening
   - vault flows and claims
   - operator task queue reliability
   - chain state feedback into product UI

4. Dedicated admin service
   - extract operator tooling out of the public web shell
   - allow frontend and backend to stay coupled inside the admin service
   - harden wallet-gated operator actions and admin runtime

5. API service hardening
   - apply Gin-oriented middleware and routing best practices
   - add rate limiting and explicit auth boundaries
   - split route registration by module instead of one mixed handler file

6. Mode B rollup architecture exploration
   - evaluate the target proof-verified exchange architecture
   - define the state / DA / withdrawal / contract boundary before any prover or L1 implementation
   - keep current centralized product stable while the design lane stays explicit

## Commander constraints

- Commander threads plan and route work; they do not implement by default.
- Worker threads execute against one task file at a time.
- Every active worker should have:
  - one task file
  - one handshake file
  - one worklog file

## Status snapshot

- Harness framework: active
- Off-chain MVP: code-complete, with truthful local deposit proof and explicit first-liquidity now in place
- Admin/operator backend: dedicated admin service is converged and core privileged market mutations are now protected at the shared API boundary
- API service hardening: modular routing, middleware layering, rate limiting, bare-`user_id` fallback removal, same-proof bootstrap replay protection, and bootstrap semantic uniqueness are now in place
- Next worker focus:
  - `TASK-OFFCHAIN-014` is complete: V2 auth now explicitly rejects signature-derived deterministic trading keys and adopts wallet-authorized browser-local trading keys
  - `TASK-CHAIN-005` is complete: oracle-settled crypto markets now have one explicit metadata / evidence / resolver contract plus a Foundry-only contract boundary if any future on-chain helper is needed
  - `TASK-OFFCHAIN-015` is complete: the first V2 trading-key runtime slice now issues SQL-backed challenges, verifies `EIP-712` wallet authorization, keeps truthful browser restore, and restores `POST /api/v1/sessions` only as a deprecated proof-tool compatibility route
  - `TASK-CHAIN-006` is complete: the first oracle runtime slice now validates metadata, writes oracle observations, preserves manual fallback ownership truthfulness, and no longer republishes the same resolved `market.event` for an already-recorded `OBSERVED` oracle outcome
  - `TASK-OFFCHAIN-013` is complete: the browser restore UX now reconciles before reauthorization, surfaces restore-in-progress / reauthorization-needed states honestly, and keeps new browser registration on the canonical trading-key routes
  - `TASK-OFFCHAIN-016` is complete: `wallet_sessions` now durably scopes canonical trading-key rows by `wallet + chain + vault`, including uniqueness that allows reusing the same trading public key across two vaults on one wallet without cross-vault rotation or readback ambiguity
  - `TASK-CHAIN-007` is complete: oracle `OBSERVED` rows now carry a dispatch checkpoint so `publish failed after OBSERVED` can retry safely without replaying settlement/account side effects
  - `TASK-OFFCHAIN-017` is complete: refresh with one unambiguous local
    trading key now restores quietly without probing the provider on mount,
    collateral balance reads retry when `USDT` is paged out by `POSITION:*`
    rows, wallet-address copy shows an obvious success state, and the personal
    page QR dialog opens higher with a more natural visual center
  - `TASK-CHAIN-008` is complete: the target Mode B architecture is now fixed
    as a `ZK-Rollup` lane with explicit state roots, slow / fast / forced
    withdrawals, exit guarantees, and a staged migration story from the current
    BSC-vault centralized-ledger design
  - `TASK-CHAIN-009` is complete: the shadow-rollup lane now has append-only
    journal storage, durable batch input, and deterministic shadow roots for
    the trading phase, while production truth still stays on the current
    SQL/Kafka path
  - `TASK-CHAIN-010` is complete: the shadow lane now captures
    market-resolution and settlement-payout inputs, fixes `shadow-batch-v1`
    as an explicit witness/public-input contract, and lands one Foundry-only
    `FunnyRollupCore` placeholder for batch metadata without widening into
    prover/verifier or production claim rewrite
  - `TASK-CHAIN-011` is complete: API/auth nonce advances now enter the
    durable shadow batch input transactionally, and `orders_root.nonce_root`
    is replayed truthfully from `NONCE_ADVANCED` journal inputs without
    widening the public-input shape
  - `TASK-CHAIN-012` is complete: the first proof-lane contract now keeps the
    current monotonic-floor nonce semantics for tranche 1, fixes the
    verifier-gated `FunnyRollupCore` acceptance boundary, and explicitly
    rejects treating operator-side auth checks or deprecated `/api/v1/sessions`
    as verifier-eligible proof truth
  - `TASK-CHAIN-013` is complete: canonical V2 trading-key registration now
    appends witness-only `TRADING_KEY_AUTHORIZED` entries, each
    `NONCE_ADVANCED` payload can carry verifier-eligible order-authorization
    witness material, and verifier-eligible proof tooling has moved to the
    `trading-keys` routes instead of deprecated `/api/v1/sessions`
  - `TASK-CHAIN-014` is complete: canonical V2 auth witness material is now
    normalized into a future verifier-lane contract, target-batch nonce auth
    rows are explicitly classified as `JOINED`, `MISSING_TRADING_KEY_AUTHORIZED`,
    or `NON_VERIFIER_ELIGIBLE`, and verifier-prep docs/runbooks now point
    truthful restore at `GET /api/v1/trading-keys` instead of deprecated
    `/api/v1/sessions`
  - `TASK-CHAIN-015` is complete: `FunnyRollupCore` now has one minimal
    Foundry-only `acceptVerifiedBatch(...)` lane that consumes the stable
    verifier-gated batch contract from `TASK-CHAIN-014`, keeps the public-input
    shape unchanged, and rejects batches whose auth proof contains any row that
    is not `JOINED` before advancing `latestAcceptedStateRoot`
  - `TASK-CHAIN-016` is complete: `BuildVerifierStateRootAcceptanceContract(...)`
    now exports a stable `solidity_export` contract for
    `FunnyRollupCore.acceptVerifiedBatch(...)`, and accepted batches must now
    anchor against prior `recordBatchMetadata(...)` instead of relying on
    self-consistent calldata
  - `TASK-CHAIN-017` is complete: `rollup.VerifierArtifactBundle` now freezes
    the first deterministic prover/verifier artifact lane, `FunnyRollupCore`
    passes a full verifier context instead of a bare hash stub, and Go/Solidity
    now pin one shared `verifierGateHash` parity fixture
  - `TASK-CHAIN-018` is complete: the first real `FunnyRollupVerifier`
    boundary now consumes `VerifierContext`, enforces
    `batchEncodingHash == keccak256("shadow-batch-v1")`, recomputes
    `verifierGateHash` onchain, and validates the current placeholder
    `verifierProof = abi.encode(proofTypeHash, verifierGateHash)` envelope
    without changing production truth
  - `TASK-CHAIN-019` is complete: `VerifierArtifactBundle` now exports one
    explicit outer proof/public-signal schema with stable version hashes,
    public-signal ordering, and verifier-facing proof bytes, while the current
    verifier decodes that schema and constrains
    `batchEncodingHash`/`authProofHash`/`verifierGateHash` against the
    unchanged `VerifierContext`
  - `TASK-CHAIN-020` is complete: inner `proofData-v1` is now fixed under the
    unchanged outer proof/public-signal envelope, exported deterministically by
    Go, and decoded/checked by the current verifier without reopening
    `VerifierContext`, `verifierGateHash`, or `shadow-batch-v1` public inputs
  - `TASK-CHAIN-021` is complete: the first real proving-system contract is
    now fixed as a Groth16-on-BN254 lane with a concrete verifier-facing
    `proofTypeHash`, fixed `proofBytes` ABI codec, and fixed 2x128 limb
    lifting from unchanged outer signals, while keeping `proofData-v1` and the
    outer envelope intact
  - `TASK-CHAIN-022` is complete: the first Foundry-only fixed-vk Groth16
    backend now exists behind the fixed `proofTypeHash`, `proofData-v1`
    carries non-empty fixture `proofBytes`, and Go/Foundry parity is pinned
    for limb splitting, proof codec, and one expected verifier `true` verdict
  - `TASK-CHAIN-023` is complete: the fixed-vk Groth16 lane now generates
    deterministic batch-specific proof artifacts from actual outer signals,
    while keeping the outer proof/public-signal envelope, `proofData-v1`, and
    production truth unchanged
  - `TASK-CHAIN-024` is complete: runtime-effective market status is now
    derived from stored `status + close_at`, so `OPEN` markets become
    truthfully `CLOSED` at the trading boundary across ingress/matching/read
    surfaces, while oracle markets still auto-resolve only from `resolve_at`
  - `TASK-CHAIN-025` is complete: unresolved non-oracle markets now become
    runtime `WAITING_RESOLUTION` only once they reach their adjudication
    window, ordinary operator resolve is restricted to that state, and oracle
    markets stay on the automatic oracle lane instead of sharing the manual
    resolve surface
  - `TASK-API-006` is paused: narrow the repo-structure cleanup to
    `internal/api`, splitting routes/handlers/store concerns into clearer
    module-owned packages without widening into a full repo directory
    migration or changing product behavior
  - `TASK-OFFCHAIN-018` is complete: post-`close_at` active resting orders are
    now proactively cancelled through the matching/order-event lane, and market
    detail shows connected-user order/fill state directly while dropping the
    duplicated left-side summary blocks
  - any on-chain contract surface added for the oracle lane should stay on the repo's existing Foundry toolchain, not a second Solidity framework
  - next auth cleanup, when worthwhile, should migrate repo proof tooling off deprecated `/api/v1/sessions` before retiring that blank-vault compatibility carrier
- Chain hardening: listener-driven local deposit proof is in place, and legacy local `chain_deposits` schema drift now has a documented repair path plus repair migration
