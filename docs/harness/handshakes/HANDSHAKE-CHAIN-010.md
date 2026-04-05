# HANDSHAKE-CHAIN-010

## Task

- [TASK-CHAIN-010.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-010.md)

## Thread owner

- chain/rollup worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/tasks/TASK-CHAIN-009.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-009.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-009.md`
- `docs/architecture/mode-b-zk-rollup.md`
- `docs/sql/schema.md`
- `internal/rollup/**`
- `internal/settlement/**`
- `internal/matching/**`
- `foundry.toml`
- this handshake
- `WORKLOG-CHAIN-010.md`

## Files in scope

- `internal/rollup/**`
- `internal/settlement/**` only where needed for shadow capture
- `migrations/**`
- `docs/sql/**`
- `docs/architecture/**`
- `contracts/src/**` only for minimal batch-metadata placeholders if justified
- `contracts/test/**` only for minimal batch-metadata placeholders if justified
- `foundry.toml` only if needed to keep minimal Foundry placeholder validation
  on the repo's existing toolchain
- `docs/harness/handshakes/HANDSHAKE-CHAIN-010.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-010.md`

## Inputs from other threads

- `TASK-CHAIN-009` landed the first shadow-rollup slice:
  - append-only sequencer journal
  - durable batch input
  - deterministic shadow roots
- commander review accepted `TASK-CHAIN-009` as completed, with one explicit
  residual:
  - `orders_root` still uses deterministic `ZeroNonceRoot()` until nonce
    shadowing or an explicit witness-level contract closes that gap
- commander wants this next slice to extend shadow replay into settlement and
  make the batch witness/public-input contract explicit before prover work

## Outputs back to commander

- changed files
- extended shadow settlement replay contract
- validation commands
- residual limitations
- recommended prover/L1 follow-up

## Handoff notes

- settlement-phase shadowing now extends the durable journal with:
  - `MARKET_RESOLVED`
  - settlement-triggered `ORDER_CANCELLED`
  - `SETTLEMENT_PAYOUT`
- `shadow-batch-v1` now has an explicit witness/public-input contract in
  `internal/rollup`, including the zero-nonce limitation as a tested
  witness-level constraint
- the current repo now has one minimal Foundry-only `FunnyRollupCore`
  placeholder that records `batch_data_hash / prev_state_root /
  next_state_root` and emits a batch metadata event without introducing prover
  or verifier logic
- `foundry.toml` now explicitly points tests at `contracts/test` so the new
  placeholder stays on the existing Foundry toolchain and can actually be
  validated in CI/local runs
- residual limits remain explicit:
  - `orders_root.nonce_root` is still a deterministic zero placeholder
  - `insurance_root` is still a deterministic zero placeholder
  - `withdrawals_root` still mirrors direct-vault queue state, not canonical
    claim-nullifier truth
  - no prover, verifier, L1 finality, or production withdrawal-claim rewrite

## Blockers

- do not claim the product is already Mode B
- do not widen into prover, verifier, or production withdrawal-claim rewrite
- if contracts are touched, stay on Foundry only

## Status

- completed
