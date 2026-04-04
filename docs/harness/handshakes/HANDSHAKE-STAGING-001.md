# HANDSHAKE-STAGING-001

## Task

- [TASK-STAGING-001.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-STAGING-001.md)

## Thread owner

- implementation worker in staging validation mode

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/operations/core-business-test-flow.md`
- `docs/deploy/staging-bsc-testnet.md`
- `WORKLOG-API-004.md`
- this handshake
- `WORKLOG-STAGING-001.md`

## Files in scope

- `docs/harness/worklogs/WORKLOG-STAGING-001.md`
- `scripts/staging-concurrency-orders.mjs`
- no product code files unless commander explicitly creates a bugfix follow-up task

## Inputs from other threads

- deployed domains are:
  - `https://funnyoption.xyz/`
  - `https://admin.funnyoption.xyz/`
- `TASK-API-004` hardened bootstrap semantics:
  - same-terms second privileged bootstrap sells should be rejected even with a fresh `requested_at`
  - normal session-backed user orders should still work
- a funded BSC testnet operator key exists locally, but do not write its plaintext value into repo files or chat logs

## Outputs back to commander

- pass/fail matrix for the staging E2E flow
- script-driven concurrent order/matching result summary, including command line, concurrency settings, success/fail counts, trade/open-order counters, and latency summary
- created market / order / trade / payout ids and tx hashes
- any duplicate-fill, overfill, negative-balance, or stale-freeze evidence observed under concurrent order writes
- screenshots or response snippets if useful
- exact blockers and suggested follow-up task ownership

## Blockers

- do not modify `.secrets`
- do not print private keys or secret-bearing env values
- if a funded non-operator user wallet is missing for the user-order lane, record that explicitly instead of silently collapsing the flow into an admin-only test
- keep the concurrency script bounded and configurable; if the staging stack starts rate-limiting or returning transient overload errors, record the threshold and stop instead of escalating to an unbounded load test
- do not touch files owned by `TASK-CICD-001`
- commander closeout:
  - `TASK-CHAIN-004`, `TASK-API-005`, and `TASK-OFFCHAIN-011` are all validated on deployed staging
  - bounded concurrency and settlement evidence passed on staging without duplicate-fill / overfill / negative-balance / stale-freeze anomalies
  - deployment verification on `HEAD=125f9cd` confirmed duplicate bootstrap is now side-effect free, first-liquidity collateral debits `100 * quantity`, and `/portfolio` no-session / session-user reads are truthful
  - none open for `TASK-STAGING-001` closure

## Status

- completed
