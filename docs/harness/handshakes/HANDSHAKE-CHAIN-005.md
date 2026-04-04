# HANDSHAKE-CHAIN-005

## Task

- [TASK-CHAIN-005.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-005.md)

## Thread owner

- chain/design worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/architecture/direct-deposit-session-key.md`
- `docs/architecture/order-flow.md`
- `docs/architecture/market-taxonomy-and-options.md`
- `docs/sql/schema.md`
- `foundry.toml`
- `contracts/src/FunnyVault.sol`
- `contracts/src/MockUSDT.sol`
- `admin/components/market-studio.tsx`
- `admin/app/api/operator/markets/route.ts`
- `internal/api/handler/sql_store.go`
- `internal/settlement/service/processor.go`
- this handshake
- `WORKLOG-CHAIN-005.md`

## Files in scope

- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-005.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-005.md`
- optional narrow DTO / schema placeholders only if they directly support the
  design handoff
- optional narrow Foundry-side contract / test / script placeholders only if
  they directly support the design handoff

## Inputs from other threads

- current crypto markets are still operator-resolved manually
- user now wants crypto-class markets that can auto-settle from an external
  price source
- commander wants this lane to stay design-first so the implementation does not
  widen across admin, API, settlement, and chain services without one explicit
  contract
- the repo already carries a Foundry layout for Solidity work; if this lane
  needs any on-chain contract placeholder, it should stay on Foundry instead of
  introducing a parallel contract framework

## Outputs back to commander

- changed files
- final metadata / evidence / resolver contract
- recommended follow-up implementation slice
- residual risks and rejected options

## Handoff notes

- canonical design doc:
  - `docs/architecture/oracle-settled-crypto-markets.md`
- chosen resolver boundary:
  - dedicated oracle worker
- metadata contract:
  - `markets.metadata.resolution`
- evidence contract:
  - `market_resolutions.evidence`
- first-cut storage choice:
  - reuse existing `market_resolutions`; no new SQL table in the first slice
- first-cut chain choice:
  - no on-chain helper required; if a future adapter is needed it must stay in
    `contracts/src`, `contracts/test`, and `contracts/script` under Foundry
- safety rule:
  - manual operator resolve stays as fallback, but should be blocked once an
    oracle market reaches `OBSERVED` or `RESOLVED`

## Blockers

- keep manual operator resolve as the fallback / override lane
- do not widen into a full runtime oracle fetcher in this task unless the
  design is already explicit and the change stays narrow
- do not break current market creation / settlement semantics for non-oracle
  markets
- do not introduce a second Solidity toolchain; reuse the existing Foundry
  setup

## Status

- completed
