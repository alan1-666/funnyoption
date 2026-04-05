# HANDSHAKE-OFFCHAIN-019

## Task

- [TASK-OFFCHAIN-019.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-019.md)

## Thread owner

- offchain/frontend worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-018.md`
- `docs/harness/worklogs/WORKLOG-OFFCHAIN-018.md`
- `docs/architecture/order-flow.md`
- `docs/architecture/oracle-settled-crypto-markets.md`
- the Worm reference page
- `web/app/markets/[marketId]/**`
- `web/components/live-market-panel*`
- `web/components/order-ticket*`
- `web/components/market-order-activity*`
- this handshake
- `WORKLOG-OFFCHAIN-019.md`

## Files in scope

- `web/app/markets/[marketId]/**`
- `web/components/**`
- `web/lib/**` only if a tiny presentation helper/type change is required
- `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-019.md`
- `docs/harness/worklogs/WORKLOG-OFFCHAIN-019.md`

## Inputs from other threads

- `TASK-OFFCHAIN-018` already added:
  - connected-user order visibility on the detail page
  - removal of one duplicated summary block cluster
  - close-time cancellation truth on the backend
- `TASK-CHAIN-025` already landed:
  - truthful runtime `WAITING_RESOLUTION` for unresolved non-oracle markets
  - manual resolution limited to the waiting-resolution lane
- latest product feedback:
  - current market detail visuals still feel weak and cluttered
  - user explicitly wants a Worm-inspired detail-page redesign

## Outputs back to commander

- changed files
- chosen visual contract
- before/after page behavior
- validation commands
- staging verification notes
- residual limitations

## Handoff notes

- use Worm as a hierarchy reference, not as a brand clone
- keep FunnyOption's actual product/lifecycle semantics intact
- prefer one strong redesign pass over many small cosmetic tweaks
- after local validation, it is acceptable to push and deploy straight to
  staging for verification

## Blockers

- do not widen into backend/contract lanes unless a tiny read-only shim is
  truly required
- do not regress the existing order-visibility contract from `TASK-OFFCHAIN-018`

## Status

- active
