# HANDSHAKE-CHAIN-012

## Task

- [TASK-CHAIN-012.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-012.md)

## Thread owner

- chain/rollup design worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/tasks/TASK-CHAIN-011.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-011.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-011.md`
- `docs/architecture/mode-b-zk-rollup.md`
- `docs/sql/schema.md`
- `internal/rollup/**`
- `internal/api/**`
- `internal/shared/auth/**`
- `contracts/src/FunnyRollupCore.sol`
- this handshake
- `WORKLOG-CHAIN-012.md`

## Files in scope

- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-012.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-012.md`
- `contracts/src/**` only for metadata-only boundary notes/placeholders if
  justified
- `contracts/test/**` only for metadata-only boundary notes/placeholders if
  justified
- `internal/rollup/**` only for doc-aligned placeholder comments if needed

## Inputs from other threads

- `TASK-CHAIN-011` landed:
  - truthful nonce shadowing from `NONCE_ADVANCED` durable inputs
  - stable `shadow-batch-v1` public-input shape
  - explicit residual that current nonce semantics still mirror the API/auth
    monotonic floor and allow gaps
- commander review accepted `TASK-CHAIN-011` as completed
- commander wants the next slice to settle one architecture decision before
  prover work:
  - is the current monotonic-floor nonce contract good enough for tranche 1
  - or must the lane tighten to gapless/auth-gadget semantics first

## Outputs back to commander

- changed files
- chosen nonce/auth contract
- verifier-gated acceptance boundary
- rejected options
- migration consequences
- recommended first prover/implementation tranche

## Handoff notes

- first proof-lane nonce/auth contract is now fixed:
  - keep the current `(account_id, auth_key_id) -> next_nonce floor + scope +
    key_status` monotonic-floor nonce leaf for tranche 1
  - do not force a gapless nonce/runtime rewrite before prover work
  - do require the future prover lane to prove canonical trading-key auth for
    each `NONCE_ADVANCED` transition instead of trusting the API's prior
    signature check
- verifier-gated `FunnyRollupCore` acceptance boundary is now fixed:
  - current `recordBatchMetadata(...)` remains metadata-only today
  - future acceptance-gated state-root advancement must stay on the current
    public-input surface:
    `batch_id`, `batch_data_hash`, `prev_state_root`, `balances_root`,
    `orders_root`, `positions_funding_root`, `withdrawals_root`,
    `next_state_root`
  - verifier gating must prove deterministic replay of ordered
    `shadow-batch-v1` input plus the chosen monotonic-floor nonce/auth
    contract
  - `withdrawals_root`, `insurance_root`, and the transitional
    `account_id == user_id` mirror remain explicit shadow-only limits
- rejected options recorded:
  - force gapless nonce semantics before proof work
  - treat current operator-side auth checks as sufficient for verifier-gated
    batches
  - reuse current durable shadow batch input without adding any auth witness
- migration consequences recorded:
  - repo proof tooling should migrate off deprecated `/api/v1/sessions` before
    verifier-gated batches become eligible
  - the first prover tranche needs one narrow auth witness lane, but not a
    public-input rewrite or production withdrawal rewrite
- recommended next tranche:
  - add durable or prover-consumable canonical trading-key auth witness
    material that binds `NONCE_ADVANCED` to order authorization
  - wire verifier-gated state-root advancement only after that witness contract
    is explicit

## Blockers

- do not claim the product is already Mode B
- do not widen into prover, verifier, or production withdrawal-claim rewrite
- if contracts are touched, stay on Foundry only

## Status

- completed
