# HANDSHAKE-CHAIN-014

## Task

- [TASK-CHAIN-014.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-014.md)

## Thread owner

- chain/rollup worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/tasks/TASK-CHAIN-013.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-013.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-013.md`
- `docs/architecture/mode-b-zk-rollup.md`
- `docs/architecture/direct-deposit-session-key.md`
- `docs/sql/schema.md`
- `internal/shared/auth/**`
- `internal/rollup/**`
- `internal/api/**`
- `contracts/src/FunnyRollupCore.sol`
- `docs/operations/local-full-flow-acceptance.md`
- this handshake
- `WORKLOG-CHAIN-014.md`

## Files in scope

- `internal/shared/auth/**`
- `internal/rollup/**`
- `internal/api/**` only where needed for verifier-gated auth/proof prep
- `contracts/src/**` and `contracts/test/**` only for narrow metadata-aligned verifier-gate prep if justified
- `docs/architecture/**`
- `docs/sql/**`
- `docs/operations/**` only where verifier-eligible proof tooling docs must be corrected
- `docs/harness/handshakes/HANDSHAKE-CHAIN-014.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-014.md`

## Inputs from other threads

- `TASK-CHAIN-013` landed:
  - `TRADING_KEY_AUTHORIZED` witness-only journal entries for canonical V2 registration
  - `NONCE_ADVANCED.payload.order_authorization` carrying exact order-intent witness material
  - verifier-eligible proof tooling moved onto `trading-keys` routes
- commander review accepted `TASK-CHAIN-013` as completed
- commander wants the next slice to make the auth witness lane consumable by a future verifier gate, without reopening public inputs or widening into full prover/runtime rewrite

## Outputs back to commander

- changed files
- verifier-gated auth/proof contract
- validation commands
- docs/runbook updates
- residual limitations
- recommended next verifier/state-root acceptance tranche

## Handoff notes

- first verifier-gated auth/proof prep landed without changing the
  `shadow-batch-v1` public-input shape:
  - `internal/shared/auth` now exposes one normalized verifier auth binding
    contract for canonical V2 auth witness material
  - `internal/rollup` now builds:
    - `BuildVerifierAuthProofContract(history, batch)`
    - `BuildVerifierGateBatchContract(history, batch)`
  - target-batch nonce auth rows are classified as:
    - `JOINED`
    - `MISSING_TRADING_KEY_AUTHORIZED`
    - `NON_VERIFIER_ELIGIBLE`
- the `TRADING_KEY_AUTHORIZED.authorization_ref` ->
  `NONCE_ADVANCED.payload.order_authorization` join is now explicit and tested:
  - matching prior-batch canonical auth witness => `JOINED`
  - missing auth witness => explicit `MISSING_TRADING_KEY_AUTHORIZED`
  - legacy `/api/v1/sessions` compat rows remain explicit
    `NON_VERIFIER_ELIGIBLE`
- verifier-eligible runbook docs no longer point truthful restore at
  deprecated `/api/v1/sessions`; the canonical restore/readback route is
  `GET /api/v1/trading-keys`
- product/runtime truth remains unchanged:
  - still not `Mode B`
  - no prover
  - no verifier
  - no production withdrawal-claim rewrite
  - `FunnyRollupCore` remains metadata-only until a later verifier/state-root
    worker consumes the new batch/auth contract boundary

## Blockers

- do not claim the product is already Mode B
- do not widen into full prover, full verifier, or production withdrawal-claim rewrite
- if contracts are touched, stay on Foundry only

## Status

- completed
