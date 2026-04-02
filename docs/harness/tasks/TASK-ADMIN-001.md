# TASK-ADMIN-001

## Summary

Build a dedicated admin/operator route for market operations and add a reproducible local lifecycle runner that exercises the off-chain path from market creation through settlement.

## Scope

- create a separate `/admin` route inside the current `web` app instead of exposing operator tools in the public user flow
- move or reuse the existing market creation tooling behind the new admin surface
- add an admin-side market resolution flow so the same operator lane can close a market cleanly
- add a reproducible local lifecycle runner or runbook that demonstrates:
  - admin creates a market
  - a wallet-backed user session is authorized
  - a confirmed deposit is credited
  - opposing orders are placed and matched
  - the market is resolved and settlement completes
  - market, order, trade, balance, position, deposit, and payout reads reflect the terminal state
- keep the public app free of internal/operator copy and controls

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md](/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md)
- [/Users/zhangza/code/funnyoption/docs/topics/kafka-topics.md](/Users/zhangza/code/funnyoption/docs/topics/kafka-topics.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-008.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-008.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-008.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-008.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-008.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-008.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-ADMIN-001.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-ADMIN-001.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-001.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-001.md)

## Owned files

- `web/app/admin/**`
- `web/components/market-studio.tsx`
- `web/components/market-studio.module.css`
- `web/lib/api.ts`
- `web/lib/types.ts`
- `web/app/control/page.tsx`
- `cmd/**` or `scripts/**` for the local lifecycle runner
- related docs or runbooks needed to execute the local lifecycle proof

## Acceptance criteria

- `/admin` exists as a dedicated operator route and the public navigation still does not expose operator tooling
- admin can create a market from the new route without using `/control`
- admin can trigger market resolution from the same operator lane
- a reproducible local command or runbook exercises the off-chain lifecycle and captures evidence for:
  - market creation
  - session authorization
  - deposit credit
  - order placement
  - trade matching
  - settlement
  - post-settlement reads for market, orders, trades, balances, positions, deposits, and payouts
- the task writes concrete commands, outputs, and any remaining gaps back to the worklog

## Validation

- `cd /Users/zhangza/code/funnyoption/web && npm run build`
- targeted Go validation for any new runner or backend-adjacent code
- one proof that `/admin` renders and performs market creation against the local API
- one end-to-end lifecycle proof against the local stack with recorded terminal-state reads

## Dependencies

- `TASK-OFFCHAIN-008` output is the baseline

## Handoff

- return the admin route and lifecycle runner with exact usage notes
- state clearly which parts are true runtime behavior versus any local/testnet simulation needed for reproducibility
