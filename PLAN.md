# FunnyOption Master Plan

This file is the current top-level map for commander threads.
Detailed execution lives in `docs/harness/plans/active/`.

## Current source-of-truth files

- Active orchestration plan: [`docs/harness/plans/active/PLAN-2026-04-01-master.md`](/Users/zhangza/code/funnyoption/docs/harness/plans/active/PLAN-2026-04-01-master.md)
- Harness rollout task: [`docs/harness/tasks/TASK-HARNESS-001.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-HARNESS-001.md)
- Off-chain umbrella task: [`docs/harness/tasks/TASK-OFFCHAIN-001.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-001.md)
- Next execution task: [`docs/harness/tasks/TASK-API-004.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-API-004.md)

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
- API service hardening: modular routing, middleware layering, rate limiting, bare-`user_id` fallback removal, and same-proof bootstrap replay protection are now in place
- Next worker focus: define and enforce semantic uniqueness for privileged bootstrap orders so re-signing the same bootstrap sell order with a fresh `requested_at` cannot silently create a second accepted bootstrap order
- Chain hardening: active
