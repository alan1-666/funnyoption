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
- add a script-driven concurrent order-placement and matching pass against staging:
  - create one fresh market with first liquidity
  - launch concurrent session-signed buy/sell order requests across multiple taker users against the same market
  - record success/fail counts, latency summary, matched trade count, remaining open-order count, and any duplicate-fill / overfill / negative-balance / stale-freeze anomalies
  - keep concurrency bounded and configurable so the script can be rerun without accidentally hammering the shared staging environment
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
- `scripts/staging-concurrency-orders.mjs`
- no product code files by default

## Acceptance criteria

- staging E2E pass/fail matrix is recorded in the worklog
- at least one admin-created market goes through first liquidity, user order matching, and resolution
- proof snippets include the deployed URLs plus the core business identifiers needed to debug regressions
- bootstrap duplicate behavior is verified against the deployed API/admin flow
- one checked-in script can run the concurrent order/matching pass with configurable concurrency and prints a concise machine-readable summary plus human-readable anomalies
- the worklog records the exact script command, concurrency settings, aggregate counters, latency summary, and any order/trade/freeze consistency violations observed under concurrency
- if staging E2E is blocked by missing funded user-wallet credentials, RPC/network issues, or server errors, the handoff states the exact blocker and the smallest follow-up owner area

## Validation

- browser/API validation against `https://funnyoption.xyz/` and `https://admin.funnyoption.xyz/`
- `curl` / script-based checks for health and key read APIs when helpful
- `node scripts/staging-concurrency-orders.mjs ...` for concurrent order-placement and matching validation
- optional Playwright/browser screenshots if useful for operator or user UI evidence

## Dependencies

- `TASK-API-004` output is the code-policy baseline
- deployed staging environment is already available on the two domains above

## Handoff

- return the E2E pass/fail matrix, proof snippets, and all created market/order/trade/payout ids
- return the concurrency script path, command line, aggregate result summary, and any suspected consistency bug signatures
- redact any wallet private keys or secret-bearing values from worklog and chat
- suggest one narrow follow-up task per confirmed staging regression
