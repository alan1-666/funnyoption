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
- Wallet session UX optimization task: [`docs/harness/tasks/TASK-OFFCHAIN-013.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-013.md)
- Stark-style trading key auth design task: [`docs/harness/tasks/TASK-OFFCHAIN-014.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-014.md)
- Trading-key registration first-slice implementation task: [`docs/harness/tasks/TASK-OFFCHAIN-015.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-015.md)
- Vault-scoped trading-key durability follow-up task: [`docs/harness/tasks/TASK-OFFCHAIN-016.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-016.md)
- GitHub CI/CD optimization task: [`docs/harness/tasks/TASK-CICD-003.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CICD-003.md)
- Thin-trigger CI/CD simplification task: [`docs/harness/tasks/TASK-CICD-004.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CICD-004.md)
- Oracle-settled crypto market design task: [`docs/harness/tasks/TASK-CHAIN-005.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-005.md)
- Oracle-settled crypto market first-slice implementation task: [`docs/harness/tasks/TASK-CHAIN-006.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-006.md)
- Oracle dispatch retry follow-up task: [`docs/harness/tasks/TASK-CHAIN-007.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-007.md)

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
- Next worker focus:
  - `TASK-OFFCHAIN-014` is complete: V2 auth now explicitly rejects signature-derived deterministic trading keys and adopts wallet-authorized browser-local trading keys
  - `TASK-CHAIN-005` is complete: oracle-settled crypto markets now have one explicit metadata / evidence / resolver contract plus a Foundry-only contract boundary if any future on-chain helper is needed
  - `TASK-OFFCHAIN-015` is complete: the first V2 trading-key runtime slice now issues SQL-backed challenges, verifies `EIP-712` wallet authorization, keeps truthful browser restore, and restores `POST /api/v1/sessions` only as a deprecated proof-tool compatibility route
  - `TASK-CHAIN-006` is complete: the first oracle runtime slice now validates metadata, writes oracle observations, preserves manual fallback ownership truthfulness, and no longer republishes the same resolved `market.event` for an already-recorded `OBSERVED` oracle outcome
  - `TASK-OFFCHAIN-013` is complete: the browser restore UX now reconciles before reauthorization, surfaces restore-in-progress / reauthorization-needed states honestly, and keeps new browser registration on the canonical trading-key routes
  - `TASK-OFFCHAIN-016` is complete: `wallet_sessions` now durably scopes canonical trading-key rows by `wallet + chain + vault`, including uniqueness that allows reusing the same trading public key across two vaults on one wallet without cross-vault rotation or readback ambiguity
  - `TASK-CHAIN-007` is complete: oracle `OBSERVED` rows now carry a dispatch checkpoint so `publish failed after OBSERVED` can retry safely without replaying settlement/account side effects
  - any on-chain contract surface added for the oracle lane should stay on the repo's existing Foundry toolchain, not a second Solidity framework
  - next auth cleanup, when worthwhile, should migrate repo proof tooling off deprecated `/api/v1/sessions` before retiring that blank-vault compatibility carrier
- Chain hardening: listener-driven local deposit proof is in place, and legacy local `chain_deposits` schema drift now has a documented repair path plus repair migration
