# HANDSHAKE-OFFCHAIN-011

## Task

- [TASK-OFFCHAIN-011.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-011.md)

## Thread owner

- web/off-chain worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `WORKLOG-STAGING-001.md`
- this handshake
- `WORKLOG-OFFCHAIN-011.md`

## Files in scope

- `web/app/portfolio/**`
- `web/components/portfolio-shell.tsx`
- `web/lib/api.ts`
- `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-011.md`
- `docs/harness/worklogs/WORKLOG-OFFCHAIN-011.md`

## Inputs from other threads

- staging evidence:
  - Playwright snapshot on `/portfolio` still rendered wallet `0xc421d5ff322e4213a913ec257d6b4458af4255c6`, balance `0`, and old positions/orders for a generated taker user
  - `web/app/portfolio/page.tsx` calls `getBalances()`, `getPositions()`, `getOrders()`, and `getPayouts()` without a connected user id
  - `web/lib/api.ts` defaults those collection reads to `user_id=1001`

## Outputs back to commander

- patch summary and changed files
- validation commands and before/after UI/API evidence
- any residual limitation if SSR cannot know the connected session user and a client refresh path is used instead
- implementation result:
  - `/portfolio` SSR no longer calls private balances / positions / orders / payouts / profile reads without a session user
  - `PortfolioShell` now waits for `session.userId`, then refreshes balances / positions / orders / payouts / profile for that session user in parallel
  - disconnected or wallet-connected-but-not-authorized states render explicit copy and do not silently fetch user `1001`
  - private read `unavailable` / `error` states remain visible in the portfolio UI instead of being collapsed into fake empty collections
- SSR fallback strategy:
  - because the current trading session is restored from browser `localStorage`, the server render cannot know `session.userId`
  - SSR therefore renders only public markets metadata and an explicit disconnected / not-authorized portfolio state
  - after hydration, the client restores the local session and fetches user-scoped collections/profile with the recovered `session.userId`

## Blockers

- do not touch first-liquidity or chain-listener files owned by `TASK-API-005` / `TASK-CHAIN-004`
- preserve truthful unavailable/error UI for backend read failures
- none remaining for the declared portfolio ownership scope

## Status

- completed

## Deployment closeout

- committed on `main` as `125f9cd4af344680e78529c5a98358b39427e703` (`Deploy reviewed API-005 and OFFCHAIN-011 fixset`)
- GitHub Actions `staging-deploy` run `23977457019` completed `success`; `validate` and `deploy-staging` both passed
- staging server checkout `/opt/funnyoption-staging` now reports `HEAD=125f9cd`, `git status --short` clean, and `GET https://funnyoption.xyz/healthz` returns `{"env":"staging","service":"api","status":"ok"}`
