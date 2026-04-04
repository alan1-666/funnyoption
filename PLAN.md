# FunnyOption Master Plan

This file is the current top-level map for commander threads.
Detailed execution lives in `docs/harness/plans/active/`.

## Current source-of-truth files

- Active orchestration plan: [`docs/harness/plans/active/PLAN-2026-04-01-master.md`](/Users/zhangza/code/funnyoption/docs/harness/plans/active/PLAN-2026-04-01-master.md)
- Harness rollout task: [`docs/harness/tasks/TASK-HARNESS-001.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-HARNESS-001.md)
- Off-chain umbrella task: [`docs/harness/tasks/TASK-OFFCHAIN-001.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-001.md)
- Staging E2E task: [`docs/harness/tasks/TASK-STAGING-001.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-STAGING-001.md)
- Staging chain-listener unblock task: [`docs/harness/tasks/TASK-CHAIN-004.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-004.md)
- First-liquidity correctness task: [`docs/harness/tasks/TASK-API-005.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-API-005.md)
- Portfolio connected-user read task: [`docs/harness/tasks/TASK-OFFCHAIN-011.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-011.md)
- Local lifecycle wrapper alignment task: [`docs/harness/tasks/TASK-OFFCHAIN-012.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-012.md)
- GitHub CI/CD optimization task: [`docs/harness/tasks/TASK-CICD-003.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CICD-003.md)
- Thin-trigger CI/CD simplification task: [`docs/harness/tasks/TASK-CICD-004.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CICD-004.md)

## Strategic lanes

1. Off-chain MVP closeout
   - stabilize local dev
   - finish off-chain trade lifecycle
   - finish query/read surfaces
   - tighten websocket market surfaces

2. Harness operating model
   - slim `AGENTS.md`
   - formalize plans, tasks, handshakes, worklogs
   - separate commander and worker threads

3. Chain integration hardening
   - vault flows and claims
   - operator task queue reliability
   - chain state feedback into product UI

4. Dedicated admin service
   - extract operator tooling out of the public web shell
   - allow frontend and backend to stay coupled inside the admin service
   - harden wallet-gated operator actions and admin runtime

5. API service hardening
   - apply Gin-oriented middleware and routing best practices
   - add rate limiting and explicit auth boundaries
   - split route registration by module instead of one mixed handler file

## Commander constraints

- Commander threads plan and route work; they do not implement by default.
- Worker threads execute against one task file at a time.
- Every active worker should have:
  - one task file
  - one handshake file
  - one worklog file

## Status snapshot

- Harness framework: active
- Off-chain MVP: code-complete, with truthful local deposit proof and explicit first-liquidity now in place
- Admin/operator backend: dedicated admin service is converged and core privileged market mutations are now protected at the shared API boundary
- API service hardening: modular routing, middleware layering, rate limiting, bare-`user_id` fallback removal, same-proof bootstrap replay protection, and bootstrap semantic uniqueness are now in place
- Next worker focus: no blocking execution lane remains after `TASK-CICD-004` closure; staging deploy now uses a thin GitHub trigger plus a fixed host entrypoint, with exact-SHA deploys preserved and symbolic branch refs preferring freshly fetched remote-tracking refs over stale same-named local branches
- Chain hardening: listener-driven local deposit proof is in place, and legacy local `chain_deposits` schema drift now has a documented repair path plus repair migration
