# TASK-STAGING-001

## Summary

Run a full staging E2E business-flow pass on the deployed server environment and return a pass/fail matrix plus proof snippets.

## Scope

- use the deployed web/admin domains as the staging entrypoints:
  - `https://funnyoption.xyz/`
  - `https://admin.funnyoption.xyz/`
- validate the complete business flow in staging:
  - admin wallet login
  - admin market creation
  - admin first-liquidity bootstrap
  - user wallet/session authorization
  - user deposit and balance credit
  - user buy/sell order placement and matching
  - admin market resolution
  - user portfolio / open orders / settlement reads
- record a pass/fail matrix, exact URLs/endpoints, key response snippets, tx / market / order / trade / payout identifiers, and failure logs or screenshots
- explicitly verify the current bootstrap semantic-uniqueness policy in staging:
  - first bootstrap sell succeeds
  - second same-terms bootstrap sell with a fresh operator proof is rejected as duplicate
- do not patch business code in this task unless commander explicitly retasks after a concrete regression report

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/operations/core-business-test-flow.md](/Users/zhangza/code/funnyoption/docs/operations/core-business-test-flow.md)
- [/Users/zhangza/code/funnyoption/docs/deploy/staging-bsc-testnet.md](/Users/zhangza/code/funnyoption/docs/deploy/staging-bsc-testnet.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-004.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-004.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-STAGING-001.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-STAGING-001.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-STAGING-001.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-STAGING-001.md)

## Owned files

- `docs/harness/worklogs/WORKLOG-STAGING-001.md`
- no product code files by default

## Acceptance criteria

- staging E2E pass/fail matrix is recorded in the worklog
- at least one admin-created market goes through first liquidity, user order matching, and resolution
- proof snippets include the deployed URLs plus the core business identifiers needed to debug regressions
- bootstrap duplicate behavior is verified against the deployed API/admin flow
- if staging E2E is blocked by missing funded user-wallet credentials, RPC/network issues, or server errors, the handoff states the exact blocker and the smallest follow-up owner area

## Validation

- browser/API validation against `https://funnyoption.xyz/` and `https://admin.funnyoption.xyz/`
- `curl` / script-based checks for health and key read APIs when helpful
- optional Playwright/browser screenshots if useful for operator or user UI evidence

## Dependencies

- `TASK-API-004` output is the code-policy baseline
- deployed staging environment is already available on the two domains above

## Handoff

- return the E2E pass/fail matrix, proof snippets, and all created market/order/trade/payout ids
- redact any wallet private keys or secret-bearing values from worklog and chat
- suggest one narrow follow-up task per confirmed staging regression
