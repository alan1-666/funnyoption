# HANDSHAKE-OFFCHAIN-009

## Task

- [TASK-OFFCHAIN-009.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-009.md)

## Thread owner

- implementation worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/operations/local-offchain-lifecycle.md`
- `WORKLOG-ADMIN-001.md`
- `TASK-ADMIN-002.md`
- this handshake
- `WORKLOG-OFFCHAIN-009.md`

## Files in scope

- `admin/**` as the preferred admin-service root
- `web/app/admin/**` only if a migration shim or redirect is required
- `web/components/market-studio.tsx`
- `web/components/admin-market-ops.tsx`
- `internal/api/dto/order.go`
- `internal/api/handler/order_handler.go`
- `internal/account/service/**`
- `internal/settlement/service/**`
- `cmd/local-lifecycle/**`
- lifecycle docs

## Inputs from other threads

- the current lifecycle proof still uses hidden seed logic for first opposing inventory
- this task should replace that with an explicit, operator-visible path inside the dedicated admin service rather than the transitional public-web admin shell

## Outputs back to commander

- changed files
- explicit bootstrap semantics
- exact validation steps and lifecycle proof notes
- any remaining gaps in market onboarding

## Handoff notes

- implemented bootstrap semantics: debit operator collateral once, issue paired `YES` / `NO` inventory explicitly, then let the first sell quote use the normal order ingress
- the dedicated `admin/` service now owns the explicit first-liquidity UX; `web/app/admin` only carries a migration note toward that service
- remaining gap is operator wallet gating for the new dedicated runtime, which still belongs to `TASK-ADMIN-002`

## Blockers

- do not widen into admin-service extraction or live deposit-listener work in this task

## Status

- completed
