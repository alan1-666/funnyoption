# HANDSHAKE-CHAIN-011

## Task

- [TASK-CHAIN-011.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-011.md)

## Thread owner

- chain/rollup worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/tasks/TASK-CHAIN-010.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-010.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-010.md`
- `docs/architecture/mode-b-zk-rollup.md`
- `docs/sql/schema.md`
- `internal/rollup/**`
- `internal/api/**`
- `internal/shared/auth/**`
- `contracts/src/FunnyRollupCore.sol`
- this handshake
- `WORKLOG-CHAIN-011.md`

## Files in scope

- `internal/rollup/**`
- `internal/api/**` only where needed to expose canonical nonce shadow inputs
- `internal/shared/auth/**` only where needed to define nonce truth
- `migrations/**`
- `docs/sql/**`
- `docs/architecture/**`
- `contracts/src/**` only for metadata-only follow-up notes/placeholders if
  justified
- `contracts/test/**` only for metadata-only follow-up notes/placeholders if
  justified
- `docs/harness/handshakes/HANDSHAKE-CHAIN-011.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-011.md`

## Inputs from other threads

- `TASK-CHAIN-010` landed:
  - settlement-phase shadow capture
  - explicit `shadow-batch-v1` witness/public-input contract
  - minimal Foundry-only `FunnyRollupCore` batch metadata surface
- commander review accepted `TASK-CHAIN-010` as completed, with one explicit
  residual:
  - `orders_root.nonce_root` must be promoted from `ZeroNonceRoot()` into a
    truthful shadow namespace before prover/verifier work starts
- commander wants the next slice to close or narrowly bound that nonce/public-
  input gap before prover/verifier work starts

## Outputs back to commander

- changed files
- nonce/public-input truth contract
- validation commands
- residual limitations
- recommended prover/verifier follow-up

## Handoff notes

- API/auth nonce advances now enter the durable shadow lane through one narrow
  `NONCE_ADVANCED` journal payload emitted transactionally with
  `AdvanceSessionNonce`
- `orders_root.nonce_root` is no longer a zero placeholder:
  - replay rebuilds it only from durable `shadow-batch-v1` input
  - the current leaf contract is `(account_id, auth_key_id) -> next_nonce
    floor + scope + key_status`
  - `account_id` still mirrors the current `user_id`
- `shadow-batch-v1` witness/public-input shape stayed stable:
  - no public-input fields changed
  - nonce work landed as one narrow witness-entry extension plus truthful
    namespace semantics
- residual limits remain explicit:
  - this is still `shadow-only`, not production `Mode B`
  - current nonce semantics mirror the API/auth monotonic floor and still allow
    nonce gaps because the operator path only enforces `last_order_nonce <
    nonce`
  - proof-friendly signature verification, verifier-gated batch acceptance, and
    production withdrawal-claim rewrite are still out of scope
- no contract or prover/verifier rewrite was needed for this tranche

## Blockers

- do not claim the product is already Mode B
- do not widen into prover, verifier, or production withdrawal-claim rewrite
- if contracts are touched, stay on Foundry only

## Status

- completed
