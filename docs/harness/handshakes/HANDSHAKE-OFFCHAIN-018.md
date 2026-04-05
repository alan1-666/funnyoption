# HANDSHAKE-OFFCHAIN-018

## Task

- [TASK-OFFCHAIN-018.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-018.md)

## Thread owner

- offchain/runtime worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/architecture/order-flow.md`
- `docs/architecture/oracle-settled-crypto-markets.md`
- `docs/sql/schema.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-024.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-024.md`
- `internal/matching/service/**`
- `internal/matching/engine/**`
- `internal/api/handler/**`
- `web/app/markets/[marketId]/**`
- `web/components/**`
- `web/lib/**`
- this handshake
- `WORKLOG-OFFCHAIN-018.md`

## Files in scope

- `internal/matching/**`
- `internal/api/handler/**` only if narrow lifecycle/read contract changes are required
- `internal/shared/kafka/**` only if a tiny event contract extension is required
- `web/app/markets/[marketId]/**`
- `web/components/**`
- `web/lib/**`
- `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-018.md`
- `docs/harness/worklogs/WORKLOG-OFFCHAIN-018.md`
- `docs/architecture/order-flow.md` only if lifecycle truth would otherwise be unclear
- `docs/sql/schema.md` only if the chosen close-time cancellation contract affects documented runtime truth

## Inputs from other threads

- `TASK-CHAIN-024` landed:
  - `close_at` now stops ingress and matching restore
  - ordinary markets read back as runtime `CLOSED` after `close_at`
  - oracle markets still auto-resolve only from `resolve_at`
- commander review kept one explicit residual:
  - already-loaded matcher orders become inert after `close_at`, but are not
    yet proactively cancelled
- product feedback from the user:
  - market detail currently does not make connected-user order/fill state
    obvious after placement
  - left-side market summary/info repeats data and should be removed

## Outputs back to commander

- changed files
- backend close-time cancellation contract
- detail-page UX before/after
- validation commands
- staging verification notes
- residual limitations

## Handoff notes

- keep the scope centered on the main product lane
- do not widen into repo-structure cleanup or rollup work
- it is acceptable to push/deploy this tranche straight to staging once local
  validation is green; this repo is the user's own project and staging is the
  intended verification target

## Blockers

- do not claim the product is already Mode B
- do not widen into auth UX or contract/prover changes unless a tiny shim is
  strictly required

## Status

- completed
