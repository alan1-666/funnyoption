# TASK-OFFCHAIN-011

## Summary

Make `/portfolio` render balances, positions, orders, payouts, and profile for the connected session user instead of default user `1001`.

## Scope

- remove the current SSR/data default that causes `/portfolio` to show operator/user-1001 collections for a newly connected taker wallet
- keep disconnected-state UX explicit instead of silently falling back to a hard-coded user account
- make profile and collection reads use the same session user identity, or add a client refresh path that updates balances/positions/orders/payouts after the connected session is known
- preserve the existing read-error truthfulness behavior and do not collapse backend failures into fake empty user data

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-STAGING-001.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-STAGING-001.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-011.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-011.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-011.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-011.md)

## Owned files

- `web/app/portfolio/**`
- `web/components/portfolio-shell.tsx`
- `web/lib/api.ts`
- `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-011.md`
- `docs/harness/worklogs/WORKLOG-OFFCHAIN-011.md`

## Acceptance criteria

- after a wallet/session connect, `/portfolio` renders collection reads for that connected user, not default user `1001`
- a fresh generated taker from the staging script no longer sees operator-owned positions/orders as their own portfolio
- disconnected state is explicit and does not fetch private-ish collections under a hard-coded fallback user
- read failures still surface as unavailable/error UI rather than fake empty data

## Validation

- targeted web tests if available
- one browser or Playwright smoke that proves the connected session user changes balances/positions/orders/payouts rendered on `/portfolio`

## Dependencies

- `TASK-STAGING-001` supplies the wrong-user UI symptom and screenshots

## Handoff

- write the implementation summary, validation commands, and before/after UI evidence to `WORKLOG-OFFCHAIN-011.md`
