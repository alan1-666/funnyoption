# HANDSHAKE-ADMIN-002

## Task

- [TASK-ADMIN-002.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-ADMIN-002.md)

## Thread owner

- implementation worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/architecture/direct-deposit-session-key.md`
- `TASK-CHAIN-002.md`
- `WORKLOG-ADMIN-001.md`
- this handshake
- `WORKLOG-ADMIN-002.md`

## Files in scope

- `admin/**` as the preferred new dedicated service root
- `web/app/admin/**` as migration source or temporary redirect shell
- `web/components/admin-market-ops.tsx`
- `web/components/market-studio.tsx`
- `web/components/trading-session-provider.tsx`
- `web/lib/session-client.ts`
- `scripts/dev-up.sh`
- narrowly required API/session files

## Inputs from other threads

- `/web/admin` is functional today but is now considered transitional rather than the long-term operator surface
- product direction is for operator/admin actions to move into a dedicated admin service
- that dedicated service may keep frontend and backend coupled; the split is service-level, not necessarily FE/BE-level

## Outputs back to commander

- changed files
- explicit admin service shape and startup notes
- authorized and unauthorized validation notes
- explicit statement of the current admin access model

## Handoff notes

- chosen admin service shape:
  - standalone Next.js runtime rooted at `admin/`
  - local dev reuses `web/node_modules` through the `admin/package.json` `link-deps` step so no second dependency install is required
  - `scripts/dev-up.sh` now starts the dedicated admin service on `http://127.0.0.1:3001`
- current admin access model:
  - operator actions are wallet-gated at the dedicated admin service boundary
  - the connected wallet must be present in `FUNNYOPTION_OPERATOR_WALLETS`
  - create/resolve requests are signed with `personal_sign` and verified by admin-owned API routes before proxying to the shared backend
- remaining backend auth gap:
  - direct callers can still hit the core public API `POST /api/v1/markets` and `POST /api/v1/markets/:market_id/resolve` outside the admin service boundary
  - deeper core-API auth remains follow-up work

## Blockers

- do not widen into fresh-liquidity semantics or live deposit-listener work in this task

## Status

- completed
